---
id: MDS027
name: front-matter-validation
description: Front matter must satisfy required fields, types, and allowed values.
---
# MDS027: front-matter-validation

Front matter must satisfy required fields, types, and allowed
values.

- **ID**: MDS027
- **Name**: `front-matter-validation`
- **Default**: enabled, required: `[]`, fields: `{}`
- **Fixable**: no
- **Implementation**:
  [source](./)
- **Category**: meta

## Settings

| Setting  | Type | Default | Description                                             |
|----------|------|---------|---------------------------------------------------------|
| `required` | list | `[]`      | Required top-level front matter field names             |
| `fields`   | map  | `{}`      | Per-field schema with `type`, `enum`, and array constraints |

Supported field types: `string`, `int`, `number`, `bool`,
`array`, `object`.

Each `fields.<name>` value can be:

- a type string: `id: string`
- a mapping with `type` and/or `enum`
- array mappings can also set:
  `items`, `min-items`, `max-items`

## Config

```yaml
rules:
  front-matter-validation:
    required:
      - id
      - title
      - status
    fields:
      id: string
      title:
        type: string
      status:
        type: string
        enum:
          - draft
          - ready
      tags:
        type: array
        min-items: 1
        items:
          type: string
```

Disable:

```yaml
rules:
  front-matter-validation: false
```

## Examples

### Good

```markdown
---
id: plan-48
title: Front Matter Validation
status: draft
tags:
  - docs
  - metadata
---
# Front Matter Validation
```

### Bad

```markdown
---
title: Front Matter Validation
status: done
---
# Front Matter Validation
```

## Diagnostics

| Condition           | Message                                                                          |
|---------------------|----------------------------------------------------------------------------------|
| missing field       | `front matter missing required field "id"`                                         |
| type mismatch       | `front matter field "id" must be string, got int`                                  |
| invalid enum        | `front matter field "status" has invalid value "done" (allowed: "draft", "ready")` |
| array item mismatch | `front matter field "tags[1]" must be string, got int`                             |
| array size mismatch | `front matter field "tags" must have at least 1 items, got 0`                      |
| invalid YAML        | `front matter is not valid YAML: ...`                                              |
