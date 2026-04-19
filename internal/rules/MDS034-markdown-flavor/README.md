---
id: MDS034
name: markdown-flavor
status: ready
description: >-
  Flags Markdown syntax that the declared target
  flavor does not render.
---
# MDS034: markdown-flavor

Flags Markdown syntax that the declared target
flavor does not render.

- **ID**: MDS034
- **Name**: `markdown-flavor`
- **Status**: ready
- **Default**: disabled
- **Fixable**: no (fix pipeline lands in a follow-up)
- **Implementation**:
  [source](./)
- **Category**: meta

## Settings

| Key    | Type   | Description                                       |
|--------|--------|---------------------------------------------------|
| flavor | string | Target flavor: `commonmark`, `gfm`, or `goldmark` |

The flavor name is case-sensitive. The `goldmark`
profile is mdsmith-defined. It accepts GFM features
plus heading IDs. It does not accept optional
footnote, definition-list, or math extensions.

## Config

Enable with a target flavor:

```yaml
rules:
  markdown-flavor:
    flavor: gfm
```

Disable (default):

```yaml
rules:
  markdown-flavor: false
```

## Detected features

MDS034 tracks seven syntax features in this
increment. Each one is detected via the goldmark
AST with a built-in extension enabled:

| Feature            | commonmark | gfm | goldmark |
|--------------------|------------|-----|----------|
| tables             | no         | yes | yes      |
| task lists         | no         | yes | yes      |
| strikethrough      | no         | yes | yes      |
| bare-URL autolinks | no         | yes | yes      |
| footnotes          | no         | no  | no       |
| definition lists   | no         | no  | no       |
| heading IDs        | no         | no  | yes      |

Five further features need custom goldmark
extensions: superscript, subscript, block math,
inline math, and abbreviations. They are tracked
under
[plan 86](../../../plan/86_markdown-flavor-validation.md).

## Examples

### Good

<?include
file: good/gfm.md
wrap: markdown
?>

```markdown
# Heading

Text with ~~old~~ markup and a task list:

- [x] done
- [ ] todo
```

<?/include?>

### Bad

<?include
file: bad/commonmark-table.md
wrap: markdown
?>

```markdown
# Heading

| a | b |
| - | - |
| 1 | 2 |
```

<?/include?>
