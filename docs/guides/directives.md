---
title: Directive Overview
summary: >-
  Quick reference for all directives with placement
  rules and links to use-case guides.
---
# Directive Overview

mdsmith uses HTML processing instructions as
directives to generate content and enforce structure.

## Quick Reference

| Directive                 | Purpose                              | Closing tag | Fixable | Guide                                         |
|---------------------------|--------------------------------------|-------------|---------|-----------------------------------------------|
| `<?catalog?>`             | Build a file index from front matter | yes         | yes     | [Generating Content](generating-content.md)   |
| `<?include?>`             | Embed content from another file      | yes         | yes     | [Generating Content](generating-content.md)   |
| `<?require?>`             | Constrain filenames (schema-only)    | no          | no      | [Enforcing Structure](enforcing-structure.md) |
| `<?allow-empty-section?>` | Suppress empty-section diagnostic    | no          | no      | [Enforcing Structure](enforcing-structure.md) |

Two kinds of directives:

- **With closing tag** (`<?name?>...<?/name?>`):
  `fix` regenerates the body between markers.
  `check` reports when the body is out of date.
- **Without closing tag** (`<?name?>`): `check`
  validates a condition. Nothing to regenerate.

## Placement Rules

Directives are only recognized at **document root**
(parent must be the Document node in the AST).
Maximum indent is 3 spaces.

Directives are **not** recognized inside:

- list items
- blockquotes
- tables
- fenced code blocks
- HTML blocks

**4-space indent footgun**: At document root, 4 or
more leading spaces turns the line into an indented
code block. The directive is silently ignored with
**no error emitted**. Always use 0-3 spaces of
indent.

```markdown
<!-- Good: recognized -->
<?catalog
glob: "docs/*.md"
?>
<?/catalog?>

<!-- Bad: 4 spaces = code block, silently ignored -->
    <?catalog
    glob: "docs/*.md"
    ?>
    <?/catalog?>
```

## Guides

- [Generating Content](generating-content.md) —
  how to use `<?catalog?>` and `<?include?>` to build
  indexes and embed file content
- [Enforcing Structure](enforcing-structure.md) —
  how to use schemas, `<?require?>`, and
  `<?allow-empty-section?>` to validate documents
- [Coming from Hugo](hugo-migration.md) —
  key differences for Hugo users
- [Rule Directory](rule-directory.md) —
  all 33 rules with status and description
