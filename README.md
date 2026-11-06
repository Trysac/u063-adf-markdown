# adf-markdown

[![Go Reference](https://pkg.go.dev/badge/github.com/Trysac/u063-adf-markdown.svg)](https://pkg.go.dev/github.com/Trysac/u063-adf-markdown)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/Trysac/u063-adf-markdown/actions/workflows/ci.yml/badge.svg)](https://github.com/Trysac/u063-adf-markdown/actions/workflows/ci.yml)

Converts Atlassian Document Format (ADF) JSON to compact, safe Markdown for a core node subset, with canonical normalization.

## Problem

Jira and Confluence APIs return rich text as Atlassian Document Format (ADF) JSON, which is unreadable in logs, prompts and terminals, and writing ADF by hand is tedious. There is no small Go library for it. This package converts ADF to compact, safe Markdown so the content is usable in CLIs and LLM prompts.

## Install

```sh
go get github.com/Trysac/u063-adf-markdown
```

Requires Go 1.23 or newer. Standard library only; no dependencies outside it.

## Usage

```go
package main
import "fmt"
import adfmarkdown "github.com/Trysac/u063-adf-markdown"
func main() {
md, err := adfmarkdown.ToMarkdown([]byte(`{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hi","marks":[{"type":"strong"}]}]}]}`), adfmarkdown.Options{})
if err != nil { fmt.Println(err); return }
fmt.Print(md) // **Hi**
}
```

Errors `ErrInvalidADF`, `ErrTooLarge` and `ErrTooDeep` are sentinels wrapped with detail; match them with `errors.Is`. With `UnknownError`, unsupported nodes return `*UnsupportedNodeError`, which is matched with `errors.As`.

## API summary

- `ToMarkdown(adfJSON []byte, opts Options) (string, error)`: decodes and converts one ADF document.
- `Normalize(doc Node) Node`: merges adjacent text nodes with identical marks, drops empty text nodes and orders marks as link, strong, em, strike, code.
- `type Node`: fields `type`, `attrs`, `content`, `text`, `marks` and `version`, all JSON-tagged with omitempty except `type`.
- `type Mark`: fields `Type string` and `Attrs map[string]any`.
- `type Options`: `Unknown UnknownPolicy`, `MaxBytes int` (default 5 MiB), `MaxDepth int` (default 64).
- `UnknownPlaceholder` (zero value), `UnknownSkip`, `UnknownError`: policies for unsupported nodes.
- Errors: `ErrInvalidADF` (root not `doc`, version not 1, invalid JSON), `ErrTooLarge` (input over MaxBytes), `ErrTooDeep` (nesting over MaxDepth), and `*UnsupportedNodeError{Type string}`.

Supported nodes: doc, paragraph, text, heading (levels 1-6, ATX), bulletList, orderedList (`attrs.order` as the first number), listItem, codeBlock (`attrs.language`), blockquote, rule, hardBreak, mention, emoji, inlineCard, status and date. Supported marks: strong, em, strike, code and link. Unknown marks are dropped and their text is kept.

## Limits

- Input is limited to 5 MiB by default before decoding; the limit is set by `Options.MaxBytes`.
- Nesting is limited to 64 levels by default, checked during the walk; the limit is set by `Options.MaxDepth`. Nesting beyond the 10000-level limit of `encoding/json` also returns `ErrTooDeep`.
- Only http, https and mailto links are kept. Relative hrefs are rendered as plain text.
- Markdown to ADF conversion, tables, panels, media, expand, layouts, task and decision lists, extensions, HTML output and setext headings are not supported.

## Why not Atlassian's JavaScript converters

Atlassian's own converters are written in JavaScript, so Go services would need a separate runtime. This package is a small Go library with no dependencies.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) and [decisions](docs/decisions/)
- [Testing](docs/TESTING.md) and [security](docs/SECURITY.md)
- [Project record](docs/PROJECT.md) and [stack](docs/STACK.md)
- [Changelog](CHANGELOG.md) and [contributing guide](CONTRIBUTING.md)

## License

[MIT](LICENSE) (c) 2026 Miguel Gomez
