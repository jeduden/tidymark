---
id: TM019
name: catalog
description: Catalog content must reflect selected front matter fields from files matching its glob.
---
# TM019: catalog

Catalog content must reflect selected front matter fields
from files matching its glob.

- **ID**: TM019
- **Name**: `catalog`
- **Default**: enabled
- **Fixable**: yes
- **Implementation**:
  [source](../../internal/rules/catalog/)
- **Category**: meta
- **Archetype**:
  [generated-section](../../archetypes/generated-section/)

## Marker Syntax

The catalog directive uses `<!-- catalog ... -->` and
`<!-- /catalog -->` markers:

```text
<!-- catalog
key: value
-->
...generated content...
<!-- /catalog -->
```

## Directive: `catalog`

Lists files matching a glob pattern with configurable output.

The directive operates in *template mode* when the YAML body
contains a `row` key, and in *minimal mode* otherwise. See
Minimal mode below.

### Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `glob` | yes | -- | File glob pattern, resolved relative to the directory of the file containing the marker. Supports `*`, `?`, `[...]`, and `**` (recursive). Absolute paths are rejected. Parent traversal (`..`) is not supported. Brace expansion (`{a,b}`) is not supported. Dotfiles are matched by `*` and `**`; to exclude them, add dotfiles to the project's ignore list. |
| `sort` | no | `path` | Sort key with optional `-` prefix for descending order. See Sort behavior. |
| `columns` | no | -- | Per-column width and wrapping configuration for table rows. See Column constraints. |

### Template fields

The `row` section uses Go `text/template` syntax:
`{{.fieldname}}`.

- `{{.filename}}` -- the matched file's path, relative to
  the directory of the file containing the marker. Never
  includes a leading `./` prefix. Note: despite the name,
  this is a relative path (e.g., `docs/api-reference.md`),
  not a basename.
- All other names (e.g., `{{.title}}`,
  `{{.description}}`, `{{.author}}`) -- looked up in the
  matched file's YAML front matter (delimited by `---` lines
  at the start of the file)
- Missing front matter field -> empty string
- Non-string front matter value -> converted to string via
  Go's default formatting. Complex types (arrays, maps)
  produce unhelpful strings; prefer flat string fields in
  front matter for template use.

### Column constraints

The `columns` parameter configures per-column width limits
and wrapping behavior for table rows. Each key is a template
field name (matching a `{{.fieldname}}` in the `row`
template), and each value is a map with the following options:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `max-width` | int | -- | Maximum character width for the column content. When the rendered value exceeds this width, it is truncated or wrapped according to the `wrap` option. |
| `wrap` | string | `truncate` | Wrapping mode: `truncate` (shorten with `...` ellipsis) or `br` (split at word boundaries with `<br>` tags). |

The column name in the `columns` map corresponds to the
primary template field used in that column of the `row`
template. For example, if the row template is
`"| [{{.id}}]({{.filename}}) | {{.description}} |"`, then
`description` refers to the second column.

Truncation and wrapping respect markdown formatting: links
`[text](url)` and inline code `` `code` `` are not split in
the middle. When a markdown span exceeds the column width, the
text is broken before the span if possible.

Example:

```yaml
columns:
  description:
    max-width: 50
```

Example with `<br>` wrapping:

```yaml
columns:
  description:
    max-width: 50
    wrap: br
```

### Sort behavior

The `sort` value has the format `[-]KEY` where:

- `-` prefix (optional): descending order. Without prefix,
  ascending order. Only the first `-` is consumed as the
  direction prefix; `sort: --priority` means descending by
  key `-priority`.
- `KEY`: a non-empty string without whitespace. No whitespace
  trimming is performed on the key after stripping the prefix.
  One of the built-in keys or any front matter field name. A
  sort value containing whitespace (e.g., `sort: "foo bar"`)
  triggers the "invalid sort value" diagnostic.

If the sort value is empty (`sort: ""`), the "empty sort
value" diagnostic is emitted. If the key name is empty after
stripping the `-` prefix (i.e., `sort: "-"`), the "invalid
sort value" diagnostic is emitted.

#### Built-in sort keys

| Key | Description |
|-----|-------------|
| `path` | Relative file path (default) |
| `filename` | File basename |

Any other key name is looked up in the matched file's YAML
front matter. Missing front matter values sort as empty
string.

Comparison is case-insensitive. When values are equal, files
are secondarily sorted by relative file path (ascending,
case-insensitive) for deterministic output.

Examples:

- `sort: path` -- ascending by relative file path (default)
- `sort: -path` -- descending by relative file path
- `sort: title` -- ascending by front matter `title`
- `sort: -date` -- descending by front matter `date`
- `sort: filename` -- ascending by file basename

### Minimal mode

When the YAML body contains no `row` key (and no `header` or
`footer`), the directive produces a plain bullet list with one
entry per matched file:

```text
- [<basename>](<relative-path>)
```

Link text is the file's basename (e.g., `api-reference.md`).
Link target is the file's path relative to the directory of
the file containing the marker (the same value as
`{{.filename}}` in template mode). The link target never
includes a leading `./` prefix.

Front matter is read only when the sort key references a front
matter field. Otherwise, no front matter extraction is
performed.

In minimal mode, each entry is terminated by a `\n`.

### Rendering logic

1. If glob matches files: output = `header` + (`row` rendered
   per file) + `footer`. The `empty` value is ignored when
   files are matched.
2. If glob matches no files and `empty` is defined: output =
   `empty` text.
3. If glob matches no files and no `empty` key: output is
   empty (zero lines between markers).

See the
[archetype documentation](../../archetypes/generated-section/)
for details on newline handling, chomp indicators, and
validation order.

## Config

```yaml
rules:
  catalog: true
```

Disable:

```yaml
rules:
  catalog: false
```

## Examples

### Good -- minimal (glob only, no template)

```markdown
<!-- catalog
glob: docs/*.md
-->
- [api-reference.md](docs/api-reference.md)
- [getting-started.md](docs/getting-started.md)
<!-- /catalog -->
```

### Good -- list with front matter

```markdown
<!-- catalog
glob: docs/*.md
row: "- [{{.title}}]({{.filename}}) -- {{.description}}"
-->
- [API Reference](docs/api-reference.md) -- Complete API documentation
- [Getting Started](docs/getting-started.md) -- How to get started
<!-- /catalog -->
```

### Good -- table with header

```markdown
<!-- catalog
glob: docs/*.md
header: |
  | Title | Description |
  |-------|-------------|
row: "| [{{.title}}]({{.filename}}) | {{.description}} |"
-->
| Title | Description |
|-------|-------------|
| [API Reference](docs/api-reference.md) | Complete API documentation |
| [Getting Started](docs/getting-started.md) | How to get started |
<!-- /catalog -->
```

### Good -- table with header and footer

The blank line before `---` comes from the leading empty line
in the `footer: |` block scalar.

```markdown
<!-- catalog
glob: docs/*.md
header: |
  | Title | Description |
  |-------|-------------|
row: "| [{{.title}}]({{.filename}}) | {{.description}} |"
footer: |

  ---
-->
| Title | Description |
|-------|-------------|
| [API Reference](docs/api-reference.md) | Complete API documentation |
| [Getting Started](docs/getting-started.md) | How to get started |

---
<!-- /catalog -->
```

### Good -- multi-line row (rich layout)

Use `|+` to preserve the trailing blank line between entries:

```markdown
<!-- catalog
glob: docs/*.md
row: |+
  ### [{{.title}}]({{.filename}})

  {{.description}}

-->
### [API Reference](docs/api-reference.md)

Complete API documentation

### [Getting Started](docs/getting-started.md)

How to get started

<!-- /catalog -->
```

### Good -- empty fallback only

`empty` can appear without `row` to provide fallback text for
an empty glob:

```markdown
<!-- catalog
glob: docs/*.md
empty: No documents found.
-->
No documents found.
<!-- /catalog -->
```

### Good -- empty fallback with template

When the glob matches zero files, the `empty` text is rendered
regardless of whether `header`, `row`, and `footer` are also
defined:

```markdown
<!-- catalog
glob: docs/*.md
header: |
  | Title | Description |
  |-------|-------------|
row: "| [{{.title}}]({{.filename}}) | {{.description}} |"
empty: No documents found.
-->
No documents found.
<!-- /catalog -->
```

### Good -- descending sort by front matter field

```markdown
<!-- catalog
glob: docs/*.md
sort: -title
row: "- [{{.title}}]({{.filename}})"
-->
- [Getting Started](docs/getting-started.md)
- [API Reference](docs/api-reference.md)
<!-- /catalog -->
```

### Good -- table with column truncation

Long description values are truncated with `...` when they
exceed `max-width`:

```markdown
<!-- catalog
glob: docs/*.md
columns:
  description:
    max-width: 30
header: |
  | Title | Description |
  |-------|-------------|
row: "| [{{.title}}]({{.filename}}) | {{.description}} |"
-->
| Title | Description |
|-------|-------------|
| [API Reference](docs/api-reference.md) | Complete API documentation... |
<!-- /catalog -->
```

### Good -- table with br wrapping

Long values are wrapped with `<br>` tags when `wrap: br` is
configured:

```markdown
<!-- catalog
glob: docs/*.md
columns:
  description:
    max-width: 30
    wrap: br
header: |
  | Title | Description |
  |-------|-------------|
row: "| [{{.title}}]({{.filename}}) | {{.description}} |"
-->
| Title | Description |
|-------|-------------|
| [API Reference](docs/api-reference.md) | Complete API<br>documentation for developers |
<!-- /catalog -->
```

### Bad -- stale section (content does not match)

A new file `docs/tutorial.md` was added but the section was
not regenerated:

```markdown
<!-- catalog
glob: docs/*.md
row: "- [{{.title}}]({{.filename}}) -- {{.description}}"
-->
- [API Reference](docs/api-reference.md) -- Complete API documentation
<!-- /catalog -->
```

Diagnostic: `generated section is out of date`

### Bad -- unclosed marker

```markdown
<!-- catalog
glob: docs/*.md
-->
- [api-reference.md](docs/api-reference.md)
```

Diagnostic: `generated section has no closing marker`

### Bad -- nested markers

```markdown
<!-- catalog
glob: docs/*.md
-->
<!-- catalog
glob: other/*.md
-->
<!-- /catalog -->
```

Diagnostic:
`nested generated section markers are not allowed`

### Bad -- template sections without `row`

```markdown
<!-- catalog
glob: docs/*.md
header: |
  | Title | Description |
  |-------|-------------|
-->
<!-- /catalog -->
```

Diagnostic:
`generated section template missing required "row" key`

### Bad -- invalid YAML

```markdown
<!-- catalog
glob: docs/*.md
row: [invalid
-->
<!-- /catalog -->
```

Diagnostic: `generated section has invalid YAML: ...`

### Bad -- non-string YAML value

```markdown
<!-- catalog
glob: 42
-->
<!-- /catalog -->
```

Diagnostic:
`generated section has non-string value for key "glob"`

### Bad -- orphaned end marker

```markdown
Some text
<!-- /catalog -->
```

Diagnostic: `unexpected generated section end marker`

### Bad -- absolute glob path

```markdown
<!-- catalog
glob: /etc/*.md
-->
<!-- /catalog -->
```

Diagnostic:
`generated section directive has absolute glob path`

## Diagnostics

| Condition | Message |
|-----------|---------|
| Missing `glob` | `generated section directive missing required "glob" parameter` |
| Empty `glob` value | `generated section directive has empty "glob" parameter` |
| Absolute glob path | `generated section directive has absolute glob path` |
| Glob with `..` | `generated section directive has glob pattern with ".." path traversal` |
| Invalid glob | `generated section directive has invalid glob pattern: ...` |
| Empty sort value | `generated section directive has empty "sort" value` |
| Invalid sort value | `generated section directive has invalid sort value "VALUE"` |

See the
[archetype documentation](../../archetypes/generated-section/)
for shared diagnostics (content mismatch, unclosed markers,
nested markers, YAML errors, template errors).

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| File has no front matter | `{{.filename}}` works; other fields -> empty string |
| Matched file has invalid front matter | Treated as no front matter (all fields -> empty string) |
| Front matter field missing | Empty string in template output |
| Unreadable matched file | Silently skipped (not included in output) |
| Glob matches a directory | Silently skipped |
| Glob matches the linted file | Included (uses current on-disk content for front matter) |
| Matched file is binary or non-Markdown | Included; no front matter extracted; `{{.filename}}` resolves |
| Symlinks in glob results | Followed (doublestar handles cycles) |
| Dotfiles | Matched by `*` and `**`; exclude via ignore list if needed |
| Absolute glob path | Diagnostic |
| Glob with `..` | Diagnostic (parent traversal not supported by `io/fs.FS`) |
| Brace expansion in glob | Supported (handled by `doublestar` library) |
| Empty glob value | Diagnostic |
| Empty sort value | Diagnostic |
| `sort: "-"` (dash only) | Diagnostic (invalid sort value) |
| Sort value with whitespace | Diagnostic (invalid sort value) |
| Glob matches no files | Empty text or `empty` fallback text |
| Glob matches files + `empty` defined | `empty` is ignored; header+rows+footer rendered |

See the
[archetype documentation](../../archetypes/generated-section/)
for shared edge cases (markers in code blocks, multiple marker
pairs, line endings, template errors).
