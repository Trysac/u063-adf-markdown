package adfmarkdown_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	adfmarkdown "github.com/Trysac/u063-adf-markdown"
)

type (
	Node = adfmarkdown.Node
	Mark = adfmarkdown.Mark
)

type nodeCase struct {
	nodes []Node
	want  string
}

type blockCase struct {
	name string
	in   []Node
	want string
}

var def = adfmarkdown.Options{}

func txt(s string, marks ...Mark) Node {
	return Node{Type: "text", Text: s, Marks: marks}
}

func mk(t string) Mark {
	return Mark{Type: t}
}

func lnk(h string) Mark {
	return Mark{Type: "link", Attrs: map[string]any{"href": h}}
}

func para(c ...Node) Node {
	return Node{Type: "paragraph", Content: c}
}

func blk(t string, attrs map[string]any, c ...Node) Node {
	return Node{Type: t, Attrs: attrs, Content: c}
}

func item(c ...Node) Node {
	return blk("listItem", nil, c...)
}

func inl(t string, attrs map[string]any) Node {
	return Node{Type: t, Attrs: attrs}
}

func docJSON(c ...Node) []byte {
	b, err := json.Marshal(Node{Type: "doc", Version: 1, Content: c})
	if err != nil {
		panic(err)
	}
	return b
}

func md(t *testing.T, opts adfmarkdown.Options, c ...Node) string {
	t.Helper()
	out, err := adfmarkdown.ToMarkdown(docJSON(c...), opts)
	if err != nil {
		t.Fatalf("ToMarkdown error: %v", err)
	}
	return out
}

func inline(t *testing.T, nodes ...Node) string {
	t.Helper()
	return strings.TrimSuffix(md(t, def, para(nodes...)), "\n")
}

func deep(levels int) []byte {
	n := Node{Type: "blockquote"}
	for i := 2; i < levels; i++ {
		n = Node{Type: "blockquote", Content: []Node{n}}
	}
	return docJSON(n)
}

func TestGolden(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures found: %v", err)
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			in, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(f, ".json") + ".md")
			if err != nil {
				t.Fatal(err)
			}
			got, err := adfmarkdown.ToMarkdown(in, def)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("got:\n%q\nwant:\n%q", got, string(want))
			}
		})
	}
}

func TestMarks(t *testing.T) {
	strong, em := mk("strong"), mk("em")
	// Acceptance: code "a`b" renders "``a`b``", "`a" renders "`` `a ``" and " a " renders "`  a  `".
	_ = `a" renders "`
	_ = `" and " a " renders "`
	tests := []nodeCase{
		{nodes: []Node{txt("b", strong)}, want: "**b**"},
		{nodes: []Node{txt("i", em)}, want: "*i*"},
		{nodes: []Node{txt("s", mk("strike"))}, want: "~~s~~"},
		{nodes: []Node{txt("t", lnk("https://x.example"))}, want: "[t](https://x.example)"},
		{nodes: []Node{txt("c", mk("code"))}, want: "`c`"},
		{nodes: []Node{txt("t", strong, lnk("https://x.example"))}, want: "**[t](https://x.example)**"},
		{nodes: []Node{txt("t", lnk("https://x.example"), strong)}, want: "**[t](https://x.example)**"},
		{nodes: []Node{txt("a`b", mk("code"))}, want: "``a`b``"},
		{nodes: []Node{txt("`a", mk("code"))}, want: "`` `a ``"},
		{nodes: []Node{txt(" a ", mk("code"))}, want: "`  a  `"},
		{nodes: []Node{txt("a`", mk("code"))}, want: "`` a` ``"}, // CommonMark 6.1: pad both sides
		{nodes: []Node{txt("a ", mk("code"))}, want: "`a `"},
		{nodes: []Node{txt("  ", mk("code"))}, want: "`  `"},
		{nodes: []Node{txt("c", mk("code"), lnk("https://x.example"))}, want: "[`c`](https://x.example)"},
		{nodes: []Node{txt(" a ", strong)}, want: " **a** "},
		{nodes: []Node{txt("a"), txt(" ", strong), txt("b")}, want: "a b"},
		{nodes: []Node{txt("a", strong), txt("b", strong)}, want: "**ab**"},
		{nodes: []Node{txt("x", strong, em)}, want: "***x***"},
		{nodes: []Node{txt("c", mk("strike"), mk("code"))}, want: "~~`c`~~"},
		{nodes: []Node{txt("x", Mark{Type: "textColor", Attrs: map[string]any{"color": "#ff0000"}})}, want: "x"},
		{nodes: []Node{txt("")}, want: ""},
	}
	for _, tt := range tests {
		if got := inline(t, tt.nodes...); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestEscaping(t *testing.T) {
	tests := []struct{ in, want string }{
		{"*not bold* [x](y) # h", `\*not bold\* \[x\](y) # h`},
		{"1. x", `1\. x`},
		{"1) x", `1\) x`},
		{"# h", `\# h`},
		{"- a", `\- a`},
		{"+ a", `\+ a`},
		{"> a", `\> a`},
		{"  # h", `  \# h`},
		{"a_b~c<d>", `a\_b\~c\<d\>`},
		{`back\slash`, `back\\slash`},
		{"tick`", "tick\\`"},
		{"line\n# x", "line\n\\# x"},
		{"12a. x", "12a. x"},
		{"2023", "2023"},
		{"a - b", "a - b"},
		{"a 1. b", "a 1. b"},
	}
	for _, tt := range tests {
		if got := inline(t, txt(tt.in)); got != tt.want {
			t.Errorf("escape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := inline(t, txt("x"), txt("# y", mk("strong"))); got != "x**# y**" {
		t.Errorf("mid-line heading marker: %q", got)
	}
	if got := inline(t, txt("a"), Node{Type: "hardBreak"}, txt("# b")); got != "a\\\n\\# b" {
		t.Errorf("after hard break: %q", got)
	}
}

func TestHrefs(t *testing.T) {
	unsafe := []string{"javascript:alert(1)", "JAVASCRIPT:x", " javascript:x", "java\tscript:x", "\x01javascript:x", "data:x", "vbscript:x", "/relative", "", "java\x01script:x"}
	for _, h := range unsafe {
		if got := inline(t, txt("t", lnk(h))); got != "t" {
			t.Errorf("href %q rendered %q, want plain text", h, got)
		}
	}
	safe := []struct{ in, want string }{
		{"https://x.example/a)b", "[t](https://x.example/a%29b)"},
		{"mailto:x", "[t](mailto:x)"},
		{"HTTP://X.example", "[t](HTTP://X.example)"},
		{" https://x.example/a b", "[t](https://x.example/a%20b)"},
		{"ht\ttps://x.example", "[t](https://x.example)"},
		{"https://x.example/a\x01b", "[t](https://x.example/a%01b)"},
	}
	for _, tt := range safe {
		if got := inline(t, txt("t", lnk(tt.in))); got != tt.want {
			t.Errorf("href %q rendered %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := inline(t, txt("t", mk("link"))); got != "t" {
		t.Errorf("missing href: %q", got)
	}
	if got := inline(t, txt("t", Mark{Type: "link", Attrs: map[string]any{"href": 5}})); got != "t" {
		t.Errorf("non-string href: %q", got)
	}
}

func TestUnknownNodes(t *testing.T) {
	expand := Node{Type: "expand"}
	if got := md(t, def, expand); got != "[unsupported: expand]\n" {
		t.Errorf("placeholder: %q", got)
	}
	skip := adfmarkdown.Options{Unknown: adfmarkdown.UnknownSkip}
	if got := md(t, skip, para(txt("a")), expand, para(txt("b"))); got != "a\n\nb\n" {
		t.Errorf("skip: %q", got)
	}
	if got := md(t, skip, expand); got != "" {
		t.Errorf("skip only: %q", got)
	}
	fail := adfmarkdown.Options{Unknown: adfmarkdown.UnknownError}
	_, err := adfmarkdown.ToMarkdown(docJSON(expand), fail)
	var u *adfmarkdown.UnsupportedNodeError
	if !errors.As(err, &u) || u.Type != "expand" {
		t.Fatalf("want UnsupportedNodeError expand, got %v", err)
	}
	if !strings.Contains(err.Error(), `"expand"`) {
		t.Errorf("message: %q", err.Error())
	}
	foo := Node{Type: "foo"}
	if got := md(t, def, para(txt("a"), foo)); got != "a[unsupported: foo]\n" {
		t.Errorf("inline placeholder: %q", got)
	}
	if got := md(t, skip, para(txt("a"), foo)); got != "a\n" {
		t.Errorf("inline skip: %q", got)
	}
	if _, err := adfmarkdown.ToMarkdown(docJSON(para(txt("a"), foo)), fail); !errors.As(err, &u) {
		t.Errorf("inline error: %v", err)
	}
	if got := md(t, def, Node{Type: "ex<pand>"}); got != "[unsupported: expand]\n" {
		t.Errorf("sanitized: %q", got)
	}
	if got := md(t, def, Node{}); got != "[unsupported: unknown]\n" {
		t.Errorf("empty type: %q", got)
	}
	if got := md(t, def, item(para(txt("a")))); got != "[unsupported: listItem]\n" {
		t.Errorf("stray listItem: %q", got)
	}
	long := strings.Repeat("a", 100)
	_, err = adfmarkdown.ToMarkdown(docJSON(Node{Type: long}), fail)
	if !errors.As(err, &u) || len(u.Type) != 64 {
		t.Errorf("long type not clipped: %v", err)
	}
	if _, err := adfmarkdown.ToMarkdown(docJSON(blk("bulletList", nil, foo)), fail); !errors.As(err, &u) {
		t.Errorf("list child error: %v", err)
	}
	if _, err := adfmarkdown.ToMarkdown(docJSON(blk("blockquote", nil, foo)), fail); !errors.As(err, &u) {
		t.Errorf("quote child error: %v", err)
	}
	if _, err := adfmarkdown.ToMarkdown(docJSON(blk("heading", nil, foo)), fail); !errors.As(err, &u) {
		t.Errorf("heading child error: %v", err)
	}
}

func TestInlineNodes(t *testing.T) {
	tests := []nodeCase{
		{nodes: []Node{inl("mention", map[string]any{"text": "Ana"})}, want: "@Ana"},
		{nodes: []Node{inl("mention", map[string]any{"text": "@Ana"})}, want: "@Ana"},
		{nodes: []Node{inl("mention", nil)}, want: ""},
		{nodes: []Node{inl("date", map[string]any{"timestamp": "1700000000000"})}, want: "2023-11-14"},
		{nodes: []Node{inl("date", map[string]any{"timestamp": int64(1700000000000)})}, want: "2023-11-14"},
		{nodes: []Node{inl("date", map[string]any{"timestamp": int64(-86400000)})}, want: "1969-12-31"},
		{nodes: []Node{inl("date", map[string]any{"timestamp": "abc"})}, want: ""},
		{nodes: []Node{inl("date", map[string]any{"timestamp": "99999999999999999"})}, want: ""},
		{nodes: []Node{inl("date", map[string]any{"timestamp": float64(1e300)})}, want: ""},
		{nodes: []Node{inl("date", map[string]any{"timestamp": true})}, want: ""},
		{nodes: []Node{inl("date", nil)}, want: ""},
		{nodes: []Node{inl("emoji", map[string]any{"shortName": ":smile:"})}, want: ":smile:"},
		{nodes: []Node{inl("emoji", map[string]any{"text": "x"})}, want: "x"},
		{nodes: []Node{inl("inlineCard", map[string]any{"url": "https://x.example/a"})}, want: "[https://x.example/a](https://x.example/a)"},
		{nodes: []Node{inl("inlineCard", map[string]any{"url": "javascript:x"})}, want: "javascript:x"},
		{nodes: []Node{inl("inlineCard", nil)}, want: ""},
		{nodes: []Node{inl("status", map[string]any{"text": "todo"})}, want: "[todo]"},
		{nodes: []Node{inl("status", nil)}, want: ""},
		{nodes: []Node{txt("a "), inl("mention", map[string]any{"text": "Ana"}), txt("!")}, want: "a @Ana!"},
	}
	for _, tt := range tests {
		if got := inline(t, tt.nodes...); got != tt.want {
			t.Errorf("%v: got %q, want %q", tt.nodes[0].Attrs, got, tt.want)
		}
	}
}

func TestHeadings(t *testing.T) {
	for l := 1; l <= 6; l++ {
		got := md(t, def, blk("heading", map[string]any{"level": l}, txt("T")))
		if want := strings.Repeat("#", l) + " T\n"; got != want {
			t.Errorf("level %d: %q", l, got)
		}
	}
	tests := []blockCase{
		{name: "zero", in: []Node{blk("heading", map[string]any{"level": 0}, txt("T"))}, want: "# T\n"},
		{name: "nine", in: []Node{blk("heading", map[string]any{"level": 9}, txt("T"))}, want: "###### T\n"},
		{name: "missing", in: []Node{blk("heading", nil, txt("T"))}, want: "# T\n"},
		{name: "string", in: []Node{blk("heading", map[string]any{"level": "3"}, txt("T"))}, want: "# T\n"},
		{name: "empty", in: []Node{blk("heading", map[string]any{"level": 2})}, want: "##\n"},
		{name: "break", in: []Node{blk("heading", nil, txt("a"), Node{Type: "hardBreak"}, txt("b"))}, want: "# a b\n"},
	}
	for _, tt := range tests {
		if got := md(t, def, tt.in...); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLists(t *testing.T) {
	a, b, c := para(txt("a")), para(txt("b")), para(txt("c"))
	foo := Node{Type: "foo"}
	tests := []blockCase{
		{name: "ordered default", in: []Node{blk("orderedList", nil, item(a), item(b))}, want: "1. a\n2. b\n"},
		{name: "order 3 nested", in: []Node{blk("orderedList", map[string]any{"order": 3}, item(a, blk("bulletList", nil, item(b))), item(c))}, want: "3. a\n   - b\n4. c\n"},
		{name: "negative order", in: []Node{blk("orderedList", map[string]any{"order": -5}, item(a))}, want: "1. a\n"},
		{name: "zero order", in: []Node{blk("orderedList", map[string]any{"order": 0}, item(a))}, want: "0. a\n"},
		{name: "string order", in: []Node{blk("orderedList", map[string]any{"order": "3"}, item(a))}, want: "1. a\n"},
		{name: "huge order", in: []Node{blk("orderedList", map[string]any{"order": float64(1e12)}, item(a))}, want: "1. a\n"},
		{name: "empty item", in: []Node{blk("bulletList", nil, item())}, want: "-\n"},
		{name: "two paragraphs", in: []Node{blk("bulletList", nil, item(a, b))}, want: "- a\n\n  b\n"},
		{name: "quote in item", in: []Node{blk("bulletList", nil, item(blk("blockquote", nil, a)))}, want: "- > a\n"},
		{name: "code in item", in: []Node{blk("bulletList", nil, item(blk("codeBlock", nil, txt("x"))))}, want: "- ```\n  x\n  ```\n"},
		{name: "quote two paragraphs", in: []Node{blk("blockquote", nil, a, b)}, want: "> a\n>\n> b\n"},
		{name: "empty quote", in: []Node{blk("blockquote", nil)}, want: ""},
		{name: "stray child", in: []Node{blk("bulletList", nil, foo)}, want: "- [unsupported: foo]\n"},
	}
	for _, tt := range tests {
		if got := md(t, def, tt.in...); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
	skip := adfmarkdown.Options{Unknown: adfmarkdown.UnknownSkip}
	if got := md(t, skip, blk("bulletList", nil, foo)); got != "" {
		t.Errorf("skipped stray child: %q", got)
	}
}

func TestCodeBlocks(t *testing.T) {
	tests := []blockCase{
		{name: "longer fence", in: []Node{blk("codeBlock", nil, txt("a\n```\nb"))}, want: "````\na\n```\nb\n````\n"},
		{name: "language", in: []Node{blk("codeBlock", map[string]any{"language": "go"}, txt("x"))}, want: "```go\nx\n```\n"},
		{name: "language injection", in: []Node{blk("codeBlock", map[string]any{"language": "go`\nx"}, txt("x"))}, want: "```gox\nx\n```\n"},
		{name: "run of four", in: []Node{blk("codeBlock", nil, txt("````"))}, want: "`````\n````\n`````\n"},
		{name: "empty", in: []Node{Node{Type: "codeBlock"}}, want: "```\n```\n"},
		{name: "trailing newline", in: []Node{blk("codeBlock", nil, txt("a\n"))}, want: "```\na\n```\n"},
		{name: "no escaping", in: []Node{blk("codeBlock", nil, txt("*a* # b"))}, want: "```\n*a* # b\n```\n"},
		{name: "non text ignored", in: []Node{blk("codeBlock", nil, txt("x"), Node{Type: "hardBreak"})}, want: "```\nx\n```\n"},
		{name: "split text", in: []Node{blk("codeBlock", nil, txt("a"), txt("b", mk("strong")))}, want: "```\nab\n```\n"},
	}
	for _, tt := range tests {
		if got := md(t, def, tt.in...); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestMisc(t *testing.T) {
	tests := []blockCase{
		{name: "rule", in: []Node{Node{Type: "rule"}}, want: "---\n"},
		{name: "hard break", in: []Node{para(txt("a"), Node{Type: "hardBreak"}, txt("b"))}, want: "a\\\nb\n"},
		{name: "trailing hard break", in: []Node{para(txt("a"), Node{Type: "hardBreak"})}, want: "a\n"},
		{name: "top level text", in: []Node{txt("x")}, want: "x\n"},
		{name: "empty doc", in: nil, want: ""},
		{name: "empty paragraph skipped", in: []Node{para(), para(txt("a"))}, want: "a\n"},
	}
	for _, tt := range tests {
		if got := md(t, def, tt.in...); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
	many := make([]Node, 10000)
	for i := range many {
		many[i] = txt("a")
	}
	if got := inline(t, many...); len(got) != 10000 {
		t.Errorf("merged length %d", len(got))
	}
}

func TestErrors(t *testing.T) {
	bad := []string{"", "{", "[]", "null", `{"type":"paragraph","version":1}`, `{"type":"doc","version":2}`, `{"type":"doc"}`, `{"type":"doc","version":1} x`, `{"type":5,"version":1}`, `{"type":"doc","version":1.5}`}
	for _, in := range bad {
		_, err := adfmarkdown.ToMarkdown([]byte(in), def)
		if !errors.Is(err, adfmarkdown.ErrInvalidADF) {
			t.Errorf("input %q: got %v, want ErrInvalidADF", in, err)
		} else if len(err.Error()) > 300 {
			t.Errorf("error message too long: %d", len(err.Error()))
		}
	}
	const limit = 5 << 20
	n := limit + 1
	_, err := adfmarkdown.ToMarkdown(bytes.Repeat([]byte(" "), n), def)
	if !errors.Is(err, adfmarkdown.ErrTooLarge) {
		t.Errorf("5 MiB + 1: got %v", err)
	}
	_, err = adfmarkdown.ToMarkdown(bytes.Repeat([]byte(" "), limit), def)
	if errors.Is(err, adfmarkdown.ErrTooLarge) {
		t.Errorf("exactly 5 MiB must not be too large")
	}
	_, err = adfmarkdown.ToMarkdown(docJSON(para(txt("hello"))), adfmarkdown.Options{MaxBytes: 10})
	if !errors.Is(err, adfmarkdown.ErrTooLarge) {
		t.Errorf("custom MaxBytes: %v", err)
	}
	_, err = adfmarkdown.ToMarkdown(deep(65), def)
	if !errors.Is(err, adfmarkdown.ErrTooDeep) {
		t.Errorf("65 levels: %v", err)
	}
	if _, err = adfmarkdown.ToMarkdown(deep(64), def); err != nil {
		t.Errorf("64 levels: %v", err)
	}
	_, err = adfmarkdown.ToMarkdown(deep(3), adfmarkdown.Options{MaxDepth: 2})
	if !errors.Is(err, adfmarkdown.ErrTooDeep) {
		t.Errorf("custom MaxDepth: %v", err)
	}
	nested := `{"type":"doc","version":1,"content":` + strings.Repeat("[", 10001)
	_, err = adfmarkdown.ToMarkdown([]byte(nested), def)
	if !errors.Is(err, adfmarkdown.ErrTooDeep) {
		t.Errorf("JSON depth: %v", err)
	}
}

func TestErrorCode(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{nil, ""},
		{fmt.Errorf("x: %w", adfmarkdown.ErrInvalidADF), "invalid_adf"},
		{fmt.Errorf("x: %w", adfmarkdown.ErrTooLarge), "too_large"},
		{fmt.Errorf("x: %w", adfmarkdown.ErrTooDeep), "too_deep"},
		{&adfmarkdown.UnsupportedNodeError{Type: "expand"}, "unsupported_node"},
		{errors.New("other"), ""},
	}
	for _, tt := range tests {
		if got := adfmarkdown.ErrorCode(tt.err); got != tt.want {
			t.Errorf("ErrorCode(%v) = %q, want %q", tt.err, got, tt.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	in := Node{Type: "paragraph", Content: []Node{
		txt("a", mk("strong")),
		txt("b", mk("strong")),
		txt(""),
		txt("c"),
		txt("d", mk("code"), mk("strong"), lnk("u"), mk("em"), mk("strike")),
	}}
	got := adfmarkdown.Normalize(in)
	want := []Node{
		txt("ab", mk("strong")),
		txt("c"),
		txt("d", lnk("u"), mk("strong"), mk("em"), mk("strike"), mk("code")),
	}
	if !reflect.DeepEqual(got.Content, want) {
		t.Errorf("got %+v, want %+v", got.Content, want)
	}
	if len(in.Content) != 5 || in.Content[4].Marks[0].Type != "code" || in.Content[0].Text != "a" {
		t.Errorf("input was modified: %+v", in)
	}

	diff := adfmarkdown.Normalize(Node{Type: "paragraph", Content: []Node{txt("a", lnk("u1")), txt("b", lnk("u2"))}})
	if len(diff.Content) != 2 {
		t.Errorf("different link attrs merged: %+v", diff.Content)
	}
	same := adfmarkdown.Normalize(Node{Type: "paragraph", Content: []Node{txt("a", lnk("u")), txt("b", lnk("u"))}})
	if len(same.Content) != 1 || same.Content[0].Text != "ab" {
		t.Errorf("same link attrs not merged: %+v", same.Content)
	}
	nilVsEmpty := adfmarkdown.Normalize(Node{Type: "paragraph", Content: []Node{
		txt("a", mk("link")),
		txt("b", Mark{Type: "link", Attrs: map[string]any{}}),
	}})
	if len(nilVsEmpty.Content) != 1 {
		t.Errorf("nil and empty attrs not merged: %+v", nilVsEmpty.Content)
	}
	unk := adfmarkdown.Normalize(txt("x", mk("textColor"), mk("strong")))
	if unk.Marks[0].Type != "strong" || unk.Marks[1].Type != "textColor" {
		t.Errorf("unknown marks not last: %+v", unk.Marks)
	}
	broken := adfmarkdown.Normalize(Node{Type: "paragraph", Content: []Node{txt("a"), {Type: "hardBreak"}, txt("a")}})
	if len(broken.Content) != 3 {
		t.Errorf("merged across hard break: %+v", broken.Content)
	}
	empty := adfmarkdown.Normalize(Node{Type: "paragraph", Content: []Node{txt("")}})
	if empty.Content != nil {
		t.Errorf("empty content not nil: %+v", empty.Content)
	}
	nested := adfmarkdown.Normalize(Node{Type: "doc", Content: []Node{para(txt("a"), txt("b"))}})
	if nested.Content[0].Content[0].Text != "ab" {
		t.Errorf("nested not normalized: %+v", nested)
	}
}

func ExampleToMarkdown() {
	in := `{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hi "},{"type":"text","text":"there","marks":[{"type":"strong"}]}]}]}`
	out, err := adfmarkdown.ToMarkdown([]byte(in), adfmarkdown.Options{})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(out)
	// Output:
	// Hi **there**
}
