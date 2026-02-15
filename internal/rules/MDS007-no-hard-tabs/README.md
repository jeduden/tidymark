---
id: MDS007
name: no-hard-tabs
status: ready
description: No tab characters. Use spaces instead.
---
# MDS007: no-hard-tabs

No tab characters. Use spaces instead.

- **ID**: MDS007
- **Name**: `no-hard-tabs`
- **Status**: ready
- **Default**: enabled
- **Fixable**: yes
- **Implementation**:
  [source](./)
- **Category**: whitespace

## Config

Enable:

```yaml
rules:
  no-hard-tabs: true
```

Disable:

```yaml
rules:
  no-hard-tabs: false
```

## Examples

### Bad

```markdown
→Indented with a tab.
```

### Good

```markdown
  Indented with spaces.
```
