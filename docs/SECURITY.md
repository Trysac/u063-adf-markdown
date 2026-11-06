# Security notes

## Threat model

`github.com/Trysac/u063-adf-markdown` is a pure library without I/O. It converts untrusted JSON to Markdown in memory.

Defended:

- **Oversized input**: input larger than `MaxBytes` (default 5 MiB) is rejected before decoding with `ErrTooLarge`.
- **Deep nesting**: nesting beyond `MaxDepth` (default 64) is rejected during the walk with `ErrTooDeep`; decoder nesting beyond the 10000-level limit of `encoding/json` also maps to `ErrTooDeep`.
- **Unsafe links**: hrefs with schemes other than http, https and mailto are not emitted as links. The scheme is taken after removing tabs, LF and CR and trimming leading and trailing C0 controls and spaces, so obfuscated forms such as `java\tscript:` and `\x01javascript:` are caught. Relative hrefs are dropped. Unsafe inlineCard URLs are rendered as plain escaped text.
- **Markdown injection in text**: markup characters are backslash-escaped, and line-start list and heading markers are escaped. Code fences are longer than any backtick run in the code.
- **Malformed input**: invalid JSON, a root that is not `doc`, and version other than 1 return `ErrInvalidADF`.

Not defended:

- The output is Markdown, not sanitized HTML; a consumer that renders it as HTML is responsible for its own sanitization.
- Limits are configurable; callers who raise `MaxBytes` or `MaxDepth` accept the higher memory and CPU cost.
- Unsupported node types are not converted. They become a placeholder, are skipped, or return an error, depending on `Unknown`.
- No authentication, authorization or rate limiting exists; the library has no network surface.

Supply chain: standard library only, `go.sum` verification, `govulncheck` in the local gates, Dependabot weekly updates, actions pinned by commit SHA, and minimal workflow permissions. Secrets: `gitleaks` runs locally and in CI, and `.env` files are git-ignored. The library never uses `unsafe` or `os/exec`.

## Publishing

Releases are git tags served through the Go module proxy and checksum database. No registry token is stored in the repository.

To report a vulnerability, see [SECURITY.md](../SECURITY.md).
