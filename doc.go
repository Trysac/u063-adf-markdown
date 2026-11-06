// Package adfmarkdown converts Atlassian Document Format (ADF) JSON, as returned
// by the Jira and Confluence APIs, into compact and safe Markdown.
//
// It supports doc, paragraph, text, heading, bulletList, orderedList, listItem,
// codeBlock, blockquote, rule, hardBreak, the inline nodes mention, emoji,
// inlineCard, status and date, and the marks strong, em, strike, code and link.
// Only http, https and mailto links are emitted. Input size and nesting depth
// are bounded. The conversion is one way: Markdown to ADF is out of scope.
package adfmarkdown
