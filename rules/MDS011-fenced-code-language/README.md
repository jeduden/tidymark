---
id: MDS011
name: fenced-code-language
description: Fenced code blocks must specify a language.
---
# MDS011: fenced-code-language

Fenced code blocks must specify a language.

- **ID**: MDS011
- **Name**: `fenced-code-language`
- **Default**: enabled
- **Fixable**: no
- **Implementation**:
  [source](../../internal/rules/fencedcodelanguage/)
- **Category**: code

## Config

```yaml
rules:
  fenced-code-language: true
```

## Examples

### Bad

````markdown
```
some code without a language
```
````

### Good

````markdown
```go
fmt.Println("hello")
```
````
