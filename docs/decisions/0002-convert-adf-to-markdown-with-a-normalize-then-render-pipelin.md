# Convert ADF to Markdown with a normalize-then-render pipeline and a deny-by-default link policy

Status: accepted (generated, pending human review)

## Context

Atlassian ADF is returned as JSON by Jira and Confluence and is unreadable in logs, prompts and terminals. Atlassian's converters are JavaScript, so Go callers have no small library for this. The input is untrusted, and the Markdown may be shown in terminals or fed to LLM prompts or renderers, so output must be safe and bounded.

## Decision

ToMarkdown does the following: (1) checks the byte limit before decoding; (2) decodes with encoding/json into plain Node and Mark structs where numeric attrs are float64; (3) validates the root (doc, version 1); (4) checks nesting depth against MaxDepth, before any recursive work; (5) runs Normalize, which merges adjacent text nodes with identical marks, drops empty text and orders marks; (6) renders blocks and inline nodes. Links follow the WHATWG URL rules: strip tab, LF and CR, trim C0 controls and spaces, and read the scheme. Only http, https and mailto are kept; relative and other hrefs degrade to plain text. Special characters in text are always escaped, and code spans and fences are made longer than any backtick run inside. Unsupported nodes follow a policy: placeholder (default), skip or typed error. Nested list content is indented by the marker width so that it stays valid CommonMark. Errors are sentinels wrapped with %w, plus ErrorCode for stable machine-readable codes.

## Consequences

Output is deterministic and safe by construction, and every limit is typed. Over-escaping can add a few harmless backslashes. Relative links are dropped on purpose. Tables, panels, media and other nodes are not converted yet; they are reported through the unknown-node policy. The conversion is one way only. Deeply nested documents above the limit are rejected rather than truncated.
