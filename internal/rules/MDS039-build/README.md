---
id: MDS039
name: build
status: ready
description: >-
  Validate <?build?> directive parameters and keep the section body
  in sync with the recipe's rendered body_template.
---
# MDS039: build

Validate <?build?> directive parameters and keep the section body
in sync with the recipe's rendered body_template.

## What it detects

MDS039 validates each `<?build?>` directive in a Markdown file:

1. **`recipe` resolves** — the recipe name must be a built-in
   (`screenshot`, `vhs`) or declared in `build.recipes`.
2. **`output` is present and safe** — `output` is required and
   must not contain `..` path components.
3. **Required params present** — params listed as required by the
   recipe schema must all be supplied.
4. **No unknown params** (warning) — params not in the recipe's
   `required` or `optional` lists produce a warning.
5. **Body in sync** — the section body must equal the rendered
   `body_template`; MDS039 reports `generated section is out of date`
   when it diverges.

`mdsmith fix` rewrites the body using the rendered `body_template`.
No external tool is executed.

## Directive syntax

```text
<?build
recipe: screenshot
url: /inbox
output: docs/inbox.png
?>
![screenshot output: docs/inbox.png](docs/inbox.png)
<?/build?>
```

```text
<?build
recipe: vhs
input: demo.tape
output: demo.gif
?>
![vhs output: demo.gif](demo.gif)
<?/build?>
```

### Common parameters

| Name     | Required | Description                                                     |
|----------|----------|-----------------------------------------------------------------|
| `recipe` | yes      | Built-in or user-declared recipe name                           |
| `output` | yes      | Artifact path relative to the Markdown file; no `..` components |

`output` accepts any file extension; MDS039 applies no extension
filter.

## Built-in recipe schemas

### `screenshot`

| Param      | Required | Default    |
|------------|----------|------------|
| `url`      | yes      | —          |
| `selector` | no       | full page  |
| `viewport` | no       | `1280x800` |
| `wait`     | no       | `0` ms     |
| `click`    | no       | —          |
| `hide`     | no       | `[]`       |

### `vhs`

| Param   | Required |
|---------|----------|
| `input` | yes      |

## Generated body

Each recipe has a `body_template` rendered by `mdsmith fix`. The
`{output}` placeholder is replaced with the `output` param value.
The `{alt}` placeholder defaults to `"{recipe} output: {output}"`.

| Recipe       | Default `body_template` |
|--------------|-------------------------|
| `screenshot` | `![{alt}]({output})`    |
| `vhs`        | `![{alt}]({output})`    |
| custom       | `[{output}]({output})`  |

The default `body_template` for built-in recipes produces a
Markdown image with non-empty alt text, satisfying MDS032.

User-declared recipes may override `body_template` via
`build.recipes.NAME.body-template` in `.mdsmith.yml`.

## Config

```yaml
rules:
  build: false  # disable MDS039
```

User-declared recipes are read from `build.recipes` (plan 100):

```yaml
build:
  recipes:
    chart:
      body-template: "![{alt}]({output})"
      params:
        required: [data]
        optional: [title]
```

## Examples

### Good

```markdown
<?build
recipe: vhs
input: demo.tape
output: demo.gif
?>
![vhs output: demo.gif](demo.gif)
<?/build?>
```

### Bad — stale body

```markdown
<?build
recipe: vhs
input: demo.tape
output: demo.gif
?>
outdated content
<?/build?>
```

MDS039 reports: `generated section is out of date`

### Bad — unknown recipe

```markdown
<?build
recipe: nonexistent
output: out.png
?>
content
<?/build?>
```

MDS039 reports: `build directive references unknown recipe "nonexistent"`

### Bad — missing required param

```markdown
<?build
recipe: screenshot
output: out.png
?>
content
<?/build?>
```

MDS039 reports:
`build directive recipe "screenshot": missing required parameter "url"`

## Meta-Information

- **ID**: MDS039
- **Name**: `build`
- **Status**: ready
- **Default**: enabled
- **Fixable**: yes (body only)
- **Implementation**:
  [source](./)
- **Category**: meta
