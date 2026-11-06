# Architecture

## Purpose

`github.com/Trysac/u063-adf-markdown` is a small, dependency-free Go library (package `adfmarkdown`, repository root) that converts Atlassian Document Format JSON to compact, safe Markdown for a core node subset, with canonical normalization.

## Shape

```text
caller code --> package adfmarkdown (repository root)
                    |
                    +-- pure functions, no I/O, no global mutable state
                    +-- internal/ (optional private helpers)
```

- **Public API**: the exported identifiers of the root package: `ToMarkdown`, `Normalize`, the types `Node`, `Mark` and `Options`, the policies `UnknownPlaceholder`, `UnknownSkip` and `UnknownError`, the sentinels `ErrInvalidADF`, `ErrTooLarge` and `ErrTooDeep`, and `*UnsupportedNodeError`. Unexported identifiers and `internal/` packages may change without a major version.
- **Pipeline**: the input is checked against the byte limit before decoding, decoded into `Node` values, then walked with the depth limit enforced during the walk. Text is escaped, marks are applied, and links are filtered by scheme.
- **Errors**: invalid input returns an error; exported sentinel errors let callers branch with `errors.Is`. The library does not panic on input, print or exit.
- **Side effects**: none. Numeric attributes (`order`, `level`, `timestamp`) are handled as `float64`, as `encoding/json` decodes them.

## Quality contract

`scripts/check` is the single definition of the gates; `fabrica.json` tells the factory and CI how to run it. See [TESTING.md](TESTING.md).

## Decisions

Architecture decisions are recorded in [decisions/](decisions/) as numbered ADRs. Known deviations from the specification are listed in the README and the project record: ordered list items indent by their marker width, and relative hrefs are dropped.
