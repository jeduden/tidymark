---
id: TMXXX
name: rule-name
description: One-sentence description ending with period.
---
# TMXXX: rule-name

<!-- Rule README template. Copy this file, replace placeholders,
     delete sections and comments that don't apply.
     Front matter is required. The catalog directive reads
     id, name, description to generate the rules table.
     Repeat the description verbatim. Use prescriptive voice,
     present tense: "Headings must ..." not "Checks that ...". -->

One-sentence description ending with period.

- **ID**: TMXXX
- **Name**: `rule-name`
- **Default**: enabled
- **Fixable**: no
- **Implementation**:
  [source](../../internal/rules/rulename/)
- **Category**: CATEGORY
- **Archetype**:
  [NAME](../../archetypes/NAME/)

<!-- Bullets in this order: ID, Name, Default, Fixable,
     Implementation, Category, Archetype (if applicable).
     Default may include key settings: "enabled, max: 80".
     Categories: code, heading, line, link, list, meta,
     whitespace. Delete Archetype bullet if not used. -->

## Settings

<!-- Include only when rule implements Configurable.
     Type: int, string, list. Description: fragment,
     no period. Delete entire section otherwise. -->

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `key` | type | default | what it does |

## Config

<!-- Show enable, disable, and (if configurable) custom
     settings as separate labeled yaml blocks. -->

```yaml
rules:
  rule-name: true
```

Disable:

```yaml
rules:
  rule-name: false
```

## Examples

<!-- Use ```markdown fenced blocks. Minimal context.
     Invisible chars: trailing spaces as ···, tabs as →.
     Complex rules: multiple subsections labeled
     "### Good -- description". -->

### Good

```markdown
Passing example.
```

### Bad

```markdown
Failing example.
```

## Diagnostics

<!-- Include when the rule emits more than one distinct
     message. Delete for single-message rules.
     Messages: all lowercase, no trailing punctuation,
     5-15 words, describe the problem not the fix,
     quote param names: "glob", values in parens: (120 > 80). -->

| Condition | Message |
|-----------|---------|
| when X | lowercase message no trailing punctuation |

## Edge Cases

<!-- Include for complex rules. Delete otherwise. -->

| Scenario | Behavior |
|----------|----------|
| edge case | what happens |
