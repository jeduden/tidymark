// Package types is the canonical vocabulary of named
// field-type shortcuts that inline schemas (plan 146)
// and proto.md frontmatter values reference by short
// name (e.g. `created: date`).
//
// Each definition resolves to a plain CUE expression.
// mdsmith embeds this file via go:embed and reads the
// definitions to build the runtime registry used by
// schema parsing. See plan/148_named-field-type-shortcuts.md.
//
// The user-visible contract is the import path
// `github.com/jeduden/mdsmith/types` and the symbol
// names. The literal CUE `import` syntax is reserved
// for a future plan; today's surface is the bare-name
// YAML scalar (`created: date`).
package types

// #date matches an ISO-8601 calendar date in YYYY-MM-DD
// form. The check is shape-only; February 30th still
// matches.
#date: =~"^\\d{4}-\\d{2}-\\d{2}$"

// #datetime matches an ISO-8601 timestamp with seconds,
// either `Z` or a `±HH:MM` offset. Fractional seconds
// and basic-format strings are rejected.
#datetime: =~"^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(Z|[+-]\\d{2}:\\d{2})?$"

// #time matches `HH:MM` with optional `:SS` seconds.
// No timezone suffix.
#time: =~"^\\d{2}:\\d{2}(:\\d{2})?$"

// #email matches `local@domain.tld` where neither side
// contains `@` or whitespace and the domain has at
// least one `.`. Internationalised addresses and
// quoted local parts are rejected.
#email: =~"^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$"

// #url matches any string starting with `http://` or
// `https://`. The remainder is not validated.
#url: =~"^https?://"

// #filename matches a Markdown filename of safe
// characters: ASCII letters, digits, `.`, `_`, `-`,
// and a trailing `.md`. Path separators are rejected.
#filename: =~"^[A-Za-z0-9._-]+\\.md$"

// #nonEmpty is a non-empty string. Useful for fields
// that must carry content (titles, summaries).
#nonEmpty: string & !=""
