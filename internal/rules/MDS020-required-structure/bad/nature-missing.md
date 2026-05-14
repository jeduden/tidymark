---
settings:
  inline-schema:
    frontmatter:
      nature: '"directive" | "generator" | "content" | "style" | "structure"'
diagnostics:
  - line: 1
    column: 1
    message: 'front matter does not satisfy schema CUE constraints: nature: incomplete value "directive" | "generator" | "content" | "style" | "structure"'
---
# MDS999: example-rule

Body content lacking the nature key in front matter.
