---
id: MDS063
name: descriptive-link-text
status: ready
description: >-
  Link text must be descriptive. Non-descriptive phrases like "click here",
  "here", "link", and "more" fail screen readers and link-list navigation.
category: prose
nature: style
maintainability: null
---
# MDS063: descriptive-link-text

Link text must be descriptive. Non-descriptive phrases like "click here",
"here", "link", and "more" fail screen readers and link-list navigation.

The comparison is case- and whitespace-insensitive. Links whose text is a
single inline code span (an API symbol) are exempt.

## Config

Enable:

```yaml
rules:
  descriptive-link-text: true
```

Disable:

```yaml
rules:
  descriptive-link-text: false
```

Replace the default banned list:

```yaml
rules:
  descriptive-link-text:
    banned: ["read more", "learn more"]
```

## Examples

### Bad

<?include
file: bad/default.md
wrap: markdown
strip-frontmatter: "true"
?>

```markdown
# Title

[click here](https://example.com)
```

<?/include?>

### Good

<?include
file: good/default.md
wrap: markdown
?>

```markdown
# Title

Visit [the documentation](https://example.com) for more information.

See [`SomeAPI`](https://example.com/api) for details.

[![logo](logo.png)](https://example.com)
```

<?/include?>

## Meta-Information

- **ID**: MDS063
- **Name**: `descriptive-link-text`
- **Status**: ready
- **Default**: disabled (opt-in)
- **Fixable**: no
- **Implementation**:
  [source](../descriptivelinktext/)
- **Category**: prose
