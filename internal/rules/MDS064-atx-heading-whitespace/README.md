---
id: MDS064
name: atx-heading-whitespace
status: ready
description: ATX heading whitespace and indentation.
category: heading
nature: style
maintainability: null
---
# MDS064: atx-heading-whitespace

ATX heading whitespace and indentation.

Flags malformed ATX headings. Checks for missing or extra spaces after
the opening hashes, incorrect spacing around a closing hash sequence,
and headings indented away from column 1.

## Config

Enable:

```yaml
rules:
  atx-heading-whitespace: true
```

Disable:

```yaml
rules:
  atx-heading-whitespace: false
```

## Examples

### Bad

Missing space after `#`:

```markdown
#Heading
```

Multiple spaces after `#`:

```markdown
#  Heading
```

Indented heading:

```markdown
   # Heading
```

Closed ATX with no space:

```markdown
#Heading#
```

Closed ATX with multiple spaces before closing `#`:

```markdown
# Heading  #
```

### Good

```markdown
# Heading

## Section

### Subsection
```

## Meta-Information

- **ID**: MDS064
- **Name**: `atx-heading-whitespace`
- **Status**: ready
- **Default**: enabled
- **Fixable**: yes
- **Implementation**: [source](./)
- **Category**: heading
