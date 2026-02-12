---
id: TM021
name: include
description: Include section content must match the referenced file.
---
# TM021: include

Include section content must match the referenced file.

- **ID**: TM021
- **Name**: `include`
- **Default**: enabled
- **Fixable**: yes
- **Implementation**:
  [source](../../internal/rules/include/)
- **Category**: meta
- **Archetype**:
  [generated-section](../../archetypes/generated-section/)

## Marker Syntax

```text
<!-- include
file: path/to/file.md
strip-frontmatter: "true"
wrap: markdown
-->
...included content...
<!-- /include -->
```

## Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `file` | yes | -- | Relative path to include |
| `strip-frontmatter` | no | `"true"` | Remove YAML frontmatter |
| `wrap` | no | -- | Wrap in code fence (value = language) |

## Config

```yaml
rules:
  include: true
```

Disable:

```yaml
rules:
  include: false
```

## Examples

### Good

```markdown
<!-- include
file: data.md
-->
Hello world
<!-- /include -->
```

### Bad

```markdown
<!-- include
file: data.md
-->
Outdated content
<!-- /include -->
```

## Diagnostics

| Condition | Message |
|-----------|---------|
| content mismatch | generated section is out of date |
| missing file | include file "x.md" not found |
| no file param | include directive missing required "file" parameter |
| absolute path | include directive has absolute file path |
| path traversal | include directive has file path with ".." traversal |
