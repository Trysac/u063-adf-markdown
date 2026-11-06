package adfmarkdown

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const defaultMaxBytes = 5 << 20

const defaultMaxDepth = 64

const maxDateMillis = 8640000000000000

const maxTypeLen = 64

const maxOrder = 999999999

var schemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// ErrInvalidADF is returned when the input is not a valid ADF document
// (malformed JSON, root that is not "doc", or version other than 1).
var ErrInvalidADF = errors.New("adfmarkdown: invalid ADF document")

// ErrTooLarge is returned when the input exceeds Options.MaxBytes.
var ErrTooLarge = errors.New("adfmarkdown: input too large")

// ErrTooDeep is returned when the document nests deeper than Options.MaxDepth.
var ErrTooDeep = errors.New("adfmarkdown: document too deeply nested")

// UnsupportedNodeError is returned with UnknownError when a node type is not supported.
type UnsupportedNodeError struct {
	Type string
}

// Error implements the error interface.
func (e *UnsupportedNodeError) Error() string {
	return fmt.Sprintf("adfmarkdown: unsupported node type %q", e.Type)
}

// ErrorCode returns a stable machine-readable code for an error produced by this
// package: invalid_adf, too_large, too_deep or unsupported_node. Other errors give "".
func ErrorCode(err error) string {
	var u *UnsupportedNodeError
	switch {
	case errors.Is(err, ErrInvalidADF):
		return "invalid_adf"
	case errors.Is(err, ErrTooLarge):
		return "too_large"
	case errors.Is(err, ErrTooDeep):
		return "too_deep"
	case errors.As(err, &u):
		return "unsupported_node"
	}
	return ""
}

// Mark is an ADF inline mark such as strong, em, strike, code or link.
type Mark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// Node is an ADF node. Numeric attributes are float64, as encoding/json decodes them.
type Node struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []Node         `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []Mark         `json:"marks,omitempty"`
	Version int            `json:"version,omitempty"`
}

// UnknownPolicy selects what happens with unsupported nodes.
type UnknownPolicy int

const (
	// UnknownPlaceholder renders "[unsupported: type]" (the zero value).
	UnknownPlaceholder UnknownPolicy = iota
	// UnknownSkip omits unsupported nodes.
	UnknownSkip
	// UnknownError makes ToMarkdown return an *UnsupportedNodeError.
	UnknownError
)

// Options configures ToMarkdown. Zero values select the defaults:
// MaxBytes 5 MiB and MaxDepth 64.
type Options struct {
	Unknown  UnknownPolicy
	MaxBytes int
	MaxDepth int
}

// ToMarkdown converts an ADF JSON document to compact, safe Markdown.
func ToMarkdown(adfJSON []byte, opts Options) (string, error) {
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}
	if len(adfJSON) > maxBytes {
		return "", fmt.Errorf("%w: %d bytes exceeds limit of %d", ErrTooLarge, len(adfJSON), maxBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(adfJSON))
	var root Node
	if err := dec.Decode(&root); err != nil {
		if strings.Contains(err.Error(), "exceeded max depth") {
			return "", fmt.Errorf("%w: JSON nesting limit exceeded", ErrTooDeep)
		}
		return "", fmt.Errorf("%w: %s", ErrInvalidADF, clip(err.Error(), 100))
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("%w: trailing data after document", ErrInvalidADF)
	}
	if root.Type != "doc" {
		return "", fmt.Errorf("%w: root node must be doc", ErrInvalidADF)
	}
	if root.Version != 1 {
		return "", fmt.Errorf("%w: version must be 1", ErrInvalidADF)
	}
	if err := checkDepth(root, 1, maxDepth); err != nil {
		return "", err
	}
	norm := Normalize(root)
	r := &renderer{policy: opts.Unknown}
	bs, err := r.blocks(norm.Content)
	if err != nil {
		return "", err
	}
	s := joinBlocks(bs, false)
	if s == "" {
		return "", nil
	}
	return s + "\n", nil
}

func checkDepth(n Node, depth, limit int) error {
	if depth > limit {
		return fmt.Errorf("%w: more than %d levels", ErrTooDeep, limit)
	}
	for _, c := range n.Content {
		if err := checkDepth(c, depth+1, limit); err != nil {
			return err
		}
	}
	return nil
}

func markRank(t string) int {
	switch t {
	case "link":
		return 0
	case "strong":
		return 1
	case "em":
		return 2
	case "strike":
		return 3
	case "code":
		return 4
	}
	return 5
}

func normalizeMarks(ms []Mark) []Mark {
	if len(ms) == 0 {
		return nil
	}
	out := append([]Mark(nil), ms...)
	sort.SliceStable(out, func(i, j int) bool {
		return markRank(out[i].Type) < markRank(out[j].Type)
	})
	return out
}

func marksEqual(a, b []Mark) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type {
			return false
		}
		if len(a[i].Attrs) == 0 && len(b[i].Attrs) == 0 {
			continue
		}
		if !reflect.DeepEqual(a[i].Attrs, b[i].Attrs) {
			return false
		}
	}
	return true
}

// Normalize returns a canonical copy of doc: adjacent text nodes with identical
// marks are merged, empty text nodes are dropped and marks are ordered link,
// strong, em, strike, code (unknown marks last). The input is not modified.
func Normalize(doc Node) Node {
	out := doc
	out.Marks = normalizeMarks(doc.Marks)
	if len(doc.Content) == 0 {
		out.Content = nil
		return out
	}
	res := make([]Node, 0, len(doc.Content))
	var pending *strings.Builder
	flush := func() {
		if pending != nil {
			res[len(res)-1].Text = pending.String()
			pending = nil
		}
	}
	for _, c := range doc.Content {
		child := Normalize(c)
		if child.Type == "text" {
			if child.Text == "" {
				continue
			}
			if pending != nil && marksEqual(res[len(res)-1].Marks, child.Marks) {
				pending.WriteString(child.Text)
				continue
			}
			flush()
			res = append(res, child)
			pending = &strings.Builder{}
			pending.WriteString(child.Text)
			continue
		}
		flush()
		res = append(res, child)
	}
	flush()
	if len(res) == 0 {
		res = nil
	}
	out.Content = res
	return out
}

type renderer struct {
	policy UnknownPolicy
}

type blk struct {
	s    string
	list bool
}

func (r *renderer) blocks(nodes []Node) ([]blk, error) {
	var out []blk
	for _, n := range nodes {
		s, err := r.block(n)
		if err != nil {
			return nil, err
		}
		if s == "" {
			continue
		}
		out = append(out, blk{s: s, list: n.Type == "bulletList" || n.Type == "orderedList"})
	}
	return out, nil
}

func joinBlocks(bs []blk, itemMode bool) string {
	var sb strings.Builder
	for i, b := range bs {
		if i > 0 {
			if itemMode && b.list {
				sb.WriteString("\n")
			} else {
				sb.WriteString("\n\n")
			}
		}
		sb.WriteString(b.s)
	}
	return sb.String()
}

func (r *renderer) block(n Node) (string, error) {
	switch n.Type {
	case "paragraph":
		return r.inline(n.Content)
	case "heading":
		s, err := r.inline(n.Content)
		if err != nil {
			return "", err
		}
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\\\n", " "), "\n", " ")
		hashes := strings.Repeat("#", headingLevel(n.Attrs))
		if s == "" {
			return hashes, nil
		}
		return hashes + " " + s, nil
	case "bulletList":
		return r.list(n, false)
	case "orderedList":
		return r.list(n, true)
	case "codeBlock":
		return renderCode(n), nil
	case "blockquote":
		bs, err := r.blocks(n.Content)
		if err != nil {
			return "", err
		}
		s := joinBlocks(bs, false)
		if s == "" {
			return "", nil
		}
		return indent(s, "> ", "> "), nil
	case "rule":
		return "---", nil
	case "text", "hardBreak", "mention", "emoji", "inlineCard", "status", "date":
		return r.inline([]Node{n})
	}
	return r.unknown(n)
}

func (r *renderer) unknown(n Node) (string, error) {
	switch r.policy {
	case UnknownSkip:
		return "", nil
	case UnknownError:
		return "", &UnsupportedNodeError{Type: clip(n.Type, maxTypeLen)}
	}
	return "[unsupported: " + safeType(n.Type) + "]", nil
}

func (r *renderer) list(n Node, ordered bool) (string, error) {
	num := 1
	if ordered {
		num = orderStart(n.Attrs)
	}
	items := make([]string, 0, len(n.Content))
	for _, it := range n.Content {
		var body string
		var err error
		if it.Type == "listItem" {
			var bs []blk
			bs, err = r.blocks(it.Content)
			body = joinBlocks(bs, true)
		} else {
			body, err = r.unknown(it)
			if err == nil && body == "" {
				continue
			}
		}
		if err != nil {
			return "", err
		}
		marker := "- "
		if ordered {
			marker = strconv.Itoa(num) + ". "
		}
		num++
		items = append(items, indent(body, marker, strings.Repeat(" ", len(marker))))
	}
	return strings.Join(items, "\n"), nil
}

func indent(s, first, rest string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		prefix := rest
		if i == 0 {
			prefix = first
		}
		if line == "" {
			lines[i] = strings.TrimRight(prefix, " ")
		} else {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func orderStart(attrs map[string]any) int {
	f, ok := attrs["order"].(float64)
	if !ok || math.IsNaN(f) || f < 0 || f > maxOrder {
		return 1
	}
	return int(f)
}

func headingLevel(attrs map[string]any) int {
	f, ok := attrs["level"].(float64)
	if !ok || f < 1 {
		return 1
	}
	if f > 6 {
		return 6
	}
	return int(f)
}

func safeLang(s string) string {
	var sb strings.Builder
	for _, r := range clip(s, 64) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == '_', r == '+', r == '-', r == '.', r == '#':
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func longestRun(s string, ch byte) int {
	best, cur := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	return best
}

func renderCode(n Node) string {
	var sb strings.Builder
	for _, c := range n.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	text := strings.TrimSuffix(sb.String(), "\n")
	size := longestRun(text, '`') + 1
	if size < 3 {
		size = 3
	}
	fence := strings.Repeat("`", size)
	lang, _ := n.Attrs["language"].(string)
	head := fence + safeLang(lang)
	if text == "" {
		return head + "\n" + fence
	}
	return head + "\n" + text + "\n" + fence
}

func (r *renderer) inline(nodes []Node) (string, error) {
	var sb strings.Builder
	lineStart := true
	for i, n := range nodes {
		var piece string
		switch n.Type {
		case "text":
			piece = renderText(n, lineStart)
		case "hardBreak":
			if i == len(nodes)-1 {
				continue
			}
			piece = "\\\n"
		case "mention":
			t, _ := n.Attrs["text"].(string)
			if t == "" {
				continue
			}
			if !strings.HasPrefix(t, "@") {
				t = "@" + t
			}
			piece = escapeText(t, false)
		case "emoji":
			s, _ := n.Attrs["shortName"].(string)
			if s == "" {
				s, _ = n.Attrs["text"].(string)
			}
			piece = escapeText(s, false)
		case "inlineCard":
			u, _ := n.Attrs["url"].(string)
			if href, ok := safeHref(u); ok {
				piece = "[" + escapeText(u, false) + "](" + href + ")"
			} else {
				piece = escapeText(u, false)
			}
		case "status":
			t, _ := n.Attrs["text"].(string)
			if t == "" {
				continue
			}
			piece = "[" + escapeText(t, false) + "]"
		case "date":
			piece = formatDate(n.Attrs["timestamp"])
		default:
			p, err := r.unknown(n)
			if err != nil {
				return "", err
			}
			piece = p
		}
		if piece != "" {
			sb.WriteString(piece)
			lineStart = strings.HasSuffix(piece, "\n")
		}
	}
	return sb.String(), nil
}

func renderText(n Node, lineStart bool) string {
	var linkMark *Mark
	var code, strong, em, strike bool
	for i := range n.Marks {
		switch n.Marks[i].Type {
		case "link":
			linkMark = &n.Marks[i]
		case "strong":
			strong = true
		case "em":
			em = true
		case "strike":
			strike = true
		case "code":
			code = true
		}
	}
	var s string
	if code {
		s = codeSpan(n.Text)
	} else {
		s = escapeText(n.Text, lineStart)
	}
	if linkMark != nil {
		href, _ := linkMark.Attrs["href"].(string)
		if safe, ok := safeHref(href); ok {
			s = "[" + s + "](" + safe + ")"
		}
	}
	if strong {
		s = wrap(s, "**")
	}
	if em {
		s = wrap(s, "*")
	}
	if strike {
		s = wrap(s, "~~")
	}
	return s
}

func wrap(s, d string) string {
	core := strings.TrimLeft(s, " \t\n")
	lead := s[:len(s)-len(core)]
	core2 := strings.TrimRight(core, " \t\n")
	trail := core[len(core2):]
	if core2 == "" {
		return s
	}
	return lead + d + core2 + d + trail
}

func codeSpan(s string) string {
	fence := strings.Repeat("`", longestRun(s, '`')+1)
	pad := ""
	if s != "" {
		first, last := s[0], s[len(s)-1]
		bothSpaces := first == ' ' && last == ' ' && strings.Trim(s, " ") != ""
		if first == '`' || last == '`' || bothSpaces {
			pad = " "
		}
	}
	return fence + pad + s + pad + fence
}

func escapeText(s string, lineStart bool) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = escapeLine(line, i > 0 || lineStart)
	}
	return strings.Join(lines, "\n")
}

func escapeLine(line string, atStart bool) string {
	first, delim := -1, -1
	if atStart {
		i := 0
		for i < len(line) && line[i] == ' ' {
			i++
		}
		if i < len(line) {
			c := line[i]
			switch {
			case c == '#' || c == '-' || c == '+' || c == '>':
				first = i
			case c >= '0' && c <= '9':
				j := i
				for j < len(line) && line[j] >= '0' && line[j] <= '9' {
					j++
				}
				if j < len(line) && (line[j] == '.' || line[j] == ')') {
					delim = j
				}
			}
		}
	}
	var sb strings.Builder
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch c {
		case '\\', '`', '*', '_', '[', ']', '~', '<', '>':
			sb.WriteByte('\\')
		default:
			if i == first || i == delim {
				sb.WriteByte('\\')
			}
		}
		sb.WriteByte(c)
	}
	return sb.String()
}

func safeHref(raw string) (string, bool) {
	s := strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, raw)
	s = strings.TrimFunc(s, func(r rune) bool {
		return r <= 0x20
	})
	loc := schemeRe.FindString(s)
	if loc == "" {
		return "", false
	}
	switch strings.ToLower(loc[:len(loc)-1]) {
	case "http", "https", "mailto":
	default:
		return "", false
	}
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
			sb.WriteString("%20")
		case c == ')':
			sb.WriteString("%29")
		case c < 0x20 || c == 0x7f:
			fmt.Fprintf(&sb, "%%%02X", c)
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String(), true
}

func formatDate(v any) string {
	var ms int64
	switch t := v.(type) {
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return ""
		}
		ms = n
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) || math.Abs(t) > maxDateMillis {
			return ""
		}
		ms = int64(t)
	default:
		return ""
	}
	if ms > maxDateMillis || ms < -maxDateMillis {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format("2006-01-02")
}

func clip(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	return string([]rune(s)[:limit])
}

func safeType(t string) string {
	var sb strings.Builder
	for _, r := range clip(t, maxTypeLen) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == '_', r == '-':
			sb.WriteRune(r)
		}
	}
	if sb.Len() == 0 {
		return "unknown"
	}
	return sb.String()
}
