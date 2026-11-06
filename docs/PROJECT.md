# Project record

## Objective

Convert Atlassian Document Format JSON to compact, safe Markdown for a core node subset, with canonical normalization, so that Jira and Confluence rich text is usable in CLIs and LLM prompts.

## Scope

- In scope: the public API documented in [README.md](../README.md), documented and tested; the node subset doc, paragraph, text, heading, bulletList, orderedList, listItem, codeBlock, blockquote, rule, hardBreak, mention, emoji, inlineCard, status and date; the marks strong, em, strike, code and link.
- Out of scope: Markdown to ADF parsing and round trips, tables, panels, media, expand, layouts, task and decision lists, extensions, HTML output and setext headings; anything that needs dependencies or I/O without an accepted ADR.
- Known deviations from the specification: ordered list items indent by their marker width (for example 3 spaces under `3. `), because 2 spaces would not nest in CommonMark; bullet lists use 2 spaces. Relative hrefs are dropped and rendered as plain text, as the acceptance behaviour requires.

## Timeline

| Date | Event |
|---|---|
| 2026 | Repository created from the factory utility template. |

## Effort metrics

Measured values only; a value that has not been measured stays "pending measurement".

| Metric | Value |
|---|---|
| Tokens (input / output) | 524114 / 83711 |
| Hours (human) | pending measurement |
| Cost (USD) | 0.7228 |

## Factory run u063b (measured from API usage)

| Phase | Model | Batch | Input | Output | Cache write | Cache read | Cost (USD) |
|---|---|---|---|---|---|---|---|
| generation | claude-sonnet-5-5 | yes | 2255 | 63892 | 4362 | 0 | 0.3304 |
| fix-1 | claude-sonnet-5-5 | no | 10 | 905 | 16167 | 60809 | 0.0556 |
| fix-2 | claude-sonnet-5-5 | no | 10 | 1038 | 7573 | 71683 | 0.0365 |
| fix-3 | claude-sonnet-5-5 | no | 8 | 847 | 6486 | 54992 | 0.0302 |
| fix-4 | claude-sonnet-5-5 | no | 8 | 677 | 6623 | 55647 | 0.0289 |
| fix-1 | claude-sonnet-5-5 | no | 8 | 1019 | 6916 | 54556 | 0.0330 |
| audit | claude-sonnet-5-5 | no | 8 | 3104 | 30293 | 77562 | 0.1145 |
| fix-2 | claude-sonnet-5-5 | no | 10 | 2741 | 11007 | 82609 | 0.0632 |
| fix-3 | claude-sonnet-5-5 | no | 8 | 829 | 4563 | 56844 | 0.0254 |
| docs | claude-haiku-5-5 | no | 7087 | 8659 | 0 | 0 | 0.0050 |

Total: $0.7228. Prices: Anthropic list prices.
