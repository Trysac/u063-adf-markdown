# Testing

## Running

```sh
go test -cover .          # tests with coverage summary
go run ./scripts/check    # every gate, as CI and the factory run it
```

## What the tests cover

- Table-driven tests over `testdata` pairs: each case has a `.json` ADF input and an expected `.md` output, compared byte for byte. One golden pair covers headings 1-6, paragraphs, nested bullet and ordered lists (order 3), a go code block, a blockquote, a rule and a hardBreak.
- Mark rendering: strong, em, strike, link and code, including a bold link and inline code with backtick and space padding.
- Escaping of text containing Markdown syntax and of line-start markers such as `1.` and `1)`.
- Link safety: unsafe schemes, mixed-case and whitespace-obfuscated schemes, and parentheses in URLs.
- Unknown nodes and marks under each `Unknown` policy, including `errors.As` on `*UnsupportedNodeError`.
- Mention, date and emoji attribute handling, including string and numeric timestamps.
- Sentinel errors for invalid input, input over the byte limit, and nesting over the depth limit; code fences longer than any backtick run inside the code.

Tests live in package `adfmarkdown_test` and use the public API only. Usage is documented with runnable `Example` functions verified through `// Output:` comments.

## Policy

- Statement coverage of the library (every package except `scripts/`) must stay at or above 80 %; the gate fails below it.
- The race detector runs whenever cgo and a C compiler are available; the gate prints why when it has to run without it.
- Each bug fix starts with a failing test that reproduces it.
- Tests are deterministic: no network, no wall clock, no dependence on test order.

## Status

The gofmt gate and the test suite have not been run for the current revision; results are pending.

## Layout

Fixtures and golden files go in `testdata/`.
