---
id: MDS031
name: unclosed-code-block
status: ready
description: Fenced code blocks must have a matching closing fence.
---
# MDS031: unclosed-code-block

Fenced code blocks must have a matching closing fence.

- **ID**: MDS031
- **Name**: `unclosed-code-block`
- **Status**: ready
- **Default**: enabled
- **Fixable**: no
- **Implementation**:
  [source](./)
- **Category**: code

This rule detects fenced code blocks (backtick or tilde)
that are opened but never closed. An unclosed fence
consumes all subsequent content, breaking the document
structure.

## Prior Art

- gomarklint `unclosed-code-block` detects unclosed
  fenced code blocks.
- markdownlint MD040 checks code block language but does
  not detect unclosed fences.

## Config

```yaml
rules:
  unclosed-code-block: true
```

Disable:

```yaml
rules:
  unclosed-code-block: false
```

## Examples

### Good

```markdown
` `` go
fmt.Println("hello")
` ``
```

### Bad

```markdown
` `` go
fmt.Println("hello")
```

## Diagnostics

- `fenced code block opened at line N is never closed`
