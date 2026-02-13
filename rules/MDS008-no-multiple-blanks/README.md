---
id: MDS008
name: no-multiple-blanks
description: No more than one consecutive blank line.
---
# MDS008: no-multiple-blanks

No more than one consecutive blank line.

- **ID**: MDS008
- **Name**: `no-multiple-blanks`
- **Default**: enabled
- **Fixable**: yes
- **Implementation**:
  [source](../../internal/rules/nomultipleblanks/)
- **Category**: whitespace

## Config

```yaml
rules:
  no-multiple-blanks: true
```

## Examples

### Bad

```markdown
First paragraph.



Second paragraph.
```

### Good

```markdown
First paragraph.

Second paragraph.
```
