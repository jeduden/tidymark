---
id: MDS036
name: section-size-limits
status: ready
description: Section length must not exceed per-level or per-heading limits.
---
# MDS036: section-size-limits

Section length must not exceed per-level or per-heading limits.

- **ID**: MDS036
- **Name**: `section-size-limits`
- **Status**: ready
- **Default**: disabled
- **Fixable**: no
- **Implementation**:
  [source](./)
- **Category**: heading

A section spans from its heading line up to (but not including) the next
heading line of any level, or the end of file. Nested subsections are
measured separately from their parent, so the limit applies to the
content written directly under each heading.

## Settings

| Setting       | Type | Default | Description                                        |
|---------------|------|---------|----------------------------------------------------|
| `max`         | int  | 0       | Default line limit; zero disables the global rule. |
| `per-level`   | map  | `{}`    | Map from heading level (1-6) to line limit.        |
| `per-heading` | list | `[]`    | Regex patterns with per-heading line limits.       |

Lookup order for a heading: `per-heading` (first matching regex wins),
then `per-level`, then `max`. A resolved limit of zero disables the
check for that heading.

## Config

Enable with a default limit:

```yaml
rules:
  section-size-limits:
    max: 100
```

Per-level and per-heading overrides:

```yaml
rules:
  section-size-limits:
    max: 100
    per-level:
      1: 200
      2: 80
    per-heading:
      - pattern: "^Changelog$"
        max: 500
```

Disable:

```yaml
rules:
  section-size-limits: false
```

## Examples

### Good

<?include
file: good/default.md
wrap: markdown
?>

```markdown
# Title

Short section within the limit.

## Subsection

Also short. Each section is bounded by the next heading of any level.

## Another

Stays under the limit.
```

<?/include?>

### Bad

<?include
file: bad/default.md
wrap: markdown
?>

```markdown
# Too Long

line a
line b
line c
line d
```

<?/include?>
