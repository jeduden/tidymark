# MDS044 - horizontal-rule-style

Enforce consistent horizontal rule (thematic break) delimiter style.

## Rationale

CommonMark accepts three delimiters for horizontal rules: `---`, `***`, and `___`. Using a single, consistent delimiter makes documents more predictable and easier to edit. Additionally, `---` can be confused with a setext heading underline when not properly separated by blank lines.

## Configuration

```yaml
rules:
  horizontal-rule-style:
    enabled: false  # Disabled by default
    style: dash     # dash | asterisk | underscore
    length: 3       # Exact number of delimiter characters
    require-blank-lines: true  # Require blank lines before/after
```

## Examples

### Bad

```markdown
# Title

Text before
***
Text after
```

With `style: dash`, the asterisk delimiter is wrong.

```markdown
# Title

Text before
- - -
Text after
```

Internal spaces are not allowed.

```markdown
# Title

Text before
-----
Text after
```

With `length: 3`, five dashes is wrong.

```markdown
# Title

Text before
---
Text after
```

With `require-blank-lines: true`, missing blank lines above/below violate the rule.

### Good

```markdown
# Title

Text before

---

Text after
```

With default settings (`style: dash`, `length: 3`, `require-blank-lines: true`), this is correct.

```markdown
# Title

Text before

***

Text after
```

With `style: asterisk`, this is correct.

## Fixable

Yes. This rule can automatically replace horizontal rules with the configured delimiter and add missing blank lines.

## Category

`whitespace`

## See Also

- [MDS002 - heading-style](../MDS002-heading-style/README.md) - Control ATX vs setext heading style (setext uses `---` which can collide with horizontal rules)
