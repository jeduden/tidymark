# Generated Section (Archetype)

A **generated section** is a region of Markdown delimited by
HTML-comment markers. A linting rule checks that the content
between the markers matches what the directive would produce,
and a fix command regenerates it in place.

This archetype documents the shared mechanics. Individual rules
(e.g., [TM019 catalog](../../rules/TM019-catalog/)) define
their own parameters, template fields, and behaviors.

## Marker Syntax

Markers are HTML comments that delimit a generated section.
The marker name is the rule name (e.g., `catalog`):

```text
<!-- catalog
key: value
-->
...generated content...
<!-- /catalog -->
```

The opening comment has two parts:

1. **First line** -- `<!-- NAME` opens the directive. The
   marker name must be lowercase. Leading and trailing
   whitespace after the marker name is trimmed. Additional
   text on the first line after the marker name is ignored.
2. **YAML body** -- all subsequent lines until `-->` are
   parsed as YAML. This body contains both parameters
   (e.g., `glob`, `sort`) and template sections (e.g.,
   `header`, `row`, `footer`, `empty`).

The closing `-->` is recognized when a line, after trimming
leading and trailing whitespace, equals `-->`. The YAML body
may contain any valid YAML except that such a `-->` line
terminates the comment. If `-->` appears on its own line
within a YAML value, the HTML comment terminates prematurely,
likely causing an invalid YAML diagnostic. Avoid `-->` inside
YAML values.

If `-->` appears on the same line as the marker name (e.g.,
`<!-- catalog -->`), the YAML body is empty. This is valid
syntax but will typically trigger a missing-parameter
diagnostic.

The end marker is recognized when a line, after trimming
leading and trailing whitespace, equals `<!-- /NAME -->`.
It must appear on its own line (no other content on that
line).

## YAML Body Parsing

Parameters and template sections share the same YAML
namespace. All values must be strings. Non-string values
(null, numbers, booleans, arrays, maps) produce a diagnostic
per key.

Duplicate YAML keys produce an invalid YAML diagnostic (the
Go `yaml.v3` parser rejects them). YAML anchors, aliases,
and merge keys are supported (standard YAML features).

### Template sections

| Key | Required | Description |
|-----|----------|-------------|
| `header` | no | Static text rendered once at the top. No template expansion; `{{...}}` appears literally. |
| `row` | conditional | Per-file block, rendered once per matched entry. Uses Go `text/template` syntax. Required when `header` or `footer` is present. Must be non-empty (empty string or whitespace-only produces diagnostic). |
| `footer` | no | Static text rendered once at the bottom. No template expansion; `{{...}}` appears literally. |
| `empty` | no | Fallback text rendered instead of header+rows+footer when no entries match. No template expansion; `{{...}}` appears literally. Can appear alone without `row`. However, if `header` or `footer` is also present, `row` is required regardless of whether `empty` is defined. |

Single-line values use YAML string syntax. Multi-line values
use YAML literal block scalars (`|`, `|+`, or `|-`):

```yaml
row: "- [{{.title}}]({{.filename}}) -- {{.description}}"
```

```yaml
header: |
  | Title | Description |
  |-------|-------------|
```

Note: YAML requires quoting values that contain special
characters like `{`, `}`, `|`, `:`, or `#`. Use double quotes
for single-line templates containing `{{...}}`.

The YAML parser strips the leading indentation from literal
block content, so markdown indentation is preserved correctly
(e.g., a 4-space code block inside a 2-space-indented `|`
block comes out with the correct 4 spaces).

Use `|+` (keep chomp) to preserve trailing blank lines in
multi-line values. Use `|` (clip chomp, default) when no
trailing blank line is needed. Use `|-` (strip chomp) to
remove all trailing newlines; the implicit trailing newline
rule (see Rendering logic) will add one back.

## Template Rendering Pipeline

The `row` section uses Go `text/template` syntax:
`{{.fieldname}}`. Available fields depend on the specific
rule implementing this archetype.

### Rendering logic

1. If entries exist: output = `header` + (`row` rendered per
   entry) + `footer`. The `empty` value is ignored.
2. If no entries exist and `empty` is defined: output =
   `empty` text.
3. If no entries exist and no `empty` key: output is empty
   (zero lines between markers).

Each rendered value (`header`, `row`, `footer`, `empty`) is
terminated by a `\n` character. If the value (as parsed from
YAML) already ends with `\n`, no additional `\n` is appended.
This applies uniformly to all sections.

Use `|+` in the row template to include a trailing blank line
between entries.

Validation is sequential. Structural errors (invalid YAML,
non-string values) short-circuit further validation. Parameter
validation is performed only after YAML parsing succeeds.

No custom template functions are registered. Go's built-in
template functions (e.g., `print`, `len`, `index`) are
available.

Template output is not HTML-escaped. Values containing `<`,
`>`, `&`, or other markdown-significant characters appear
literally in the output.

## Fix Behavior

- Replace content between valid marker pairs with freshly
  generated content
- Leave malformed marker regions unchanged
- Leave sections unchanged when template execution fails
- Idempotent: fixing an up-to-date file produces no changes
- When fixing multiple marker pairs in one file, each pair's
  generation uses the on-disk filesystem state, not the
  partially-fixed in-memory content

## General Rules

- Markers inside fenced code blocks or HTML blocks are
  ignored.
- Multiple independent marker pairs per file are supported.
- Content between markers starts on the line immediately
  after the start marker's closing `-->` line and ends on
  the line immediately before the end marker line. Comparison
  is performed on the exact text between the markers
  (preserving original line endings) versus the freshly
  generated text (which uses `\n` line endings). Any
  difference, including line-ending mismatches, constitutes
  a mismatch.
- All diagnostics are reported at column 1.
- Diagnostics are reported on the start marker line, except
  for orphaned end markers which are reported on the end
  marker line.

## Common Diagnostics

These diagnostics are shared across all generated-section
rules:

| Condition | Message | Reported on |
|-----------|---------|-------------|
| Content mismatch | `generated section is out of date` | start marker line |
| No closing marker | `generated section has no closing marker` | start marker line |
| Orphaned end marker | `unexpected generated section end marker` | end marker line |
| Nested start markers | `nested generated section markers are not allowed` | nested start line |
| Invalid YAML body | `generated section has invalid YAML: ...` | start marker line |
| Non-string YAML value | `generated section has non-string value for key "KEY"` | start marker line |
| Empty `row` value | `generated section directive has empty "row" value` | start marker line |
| `header`/`footer` without `row` | `generated section template missing required "row" key` | start marker line |
| Invalid template syntax | `generated section has invalid template: ...` | start marker line |
| Template execution error | `generated section template execution failed: ...` | start marker line |

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| No entries exist | Empty text or `empty` fallback text |
| Entries exist + `empty` defined | `empty` is ignored; header+rows+footer rendered |
| No filesystem context (stdin or in-memory) | Rule skipped entirely (cannot resolve relative paths) |
| Markers inside fenced code blocks | Ignored |
| Markers inside HTML blocks | Ignored |
| Multiple marker pairs in one file | Each processed independently |
| Non-string YAML values | Diagnostic per key |
| `empty` without `row` | Valid; only `header`/`footer` require `row` |
| `empty` + `header` without `row` | Diagnostic (missing `row` still fires) |
| Duplicate YAML keys | Invalid YAML diagnostic (`yaml.v3` rejects duplicates) |
| Single-line start marker | Valid; empty YAML body triggers missing-parameter diagnostic |
| Windows-style line endings (`\r\n`) | Generated content uses `\n`; will flag `\r\n` files as stale |
| Template execution error | Diagnostic emitted; fix leaves section unchanged |
| Unknown YAML keys | Ignored (forward-compatible with future parameters) |
