---
id: MDS012
name: no-bare-urls
description: URLs must be wrapped in angle brackets or as a link, not left bare.
---
# MDS012: no-bare-urls

URLs must be wrapped in angle brackets or as a link, not left bare.

- **ID**: MDS012
- **Name**: `no-bare-urls`
- **Default**: enabled
- **Fixable**: yes
- **Implementation**:
  [source](../../internal/rules/nobareurls/)
- **Category**: link

## Config

```yaml
rules:
  no-bare-urls: true
```

## Examples

### Bad

```markdown
Visit https://example.com for more info.
```

### Good

```markdown
Visit <https://example.com> for more info.

Visit [example](https://example.com) for more info.
```
