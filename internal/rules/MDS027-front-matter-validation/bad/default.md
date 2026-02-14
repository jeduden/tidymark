---
settings:
  required:
    - id
    - status
  fields:
    status:
      type: string
      enum:
        - draft
        - ready
diagnostics:
  - line: 1
    column: 1
    message: front matter missing required field "id"
  - line: 1
    column: 1
    message: 'front matter field "status" has invalid value "done" (allowed: "draft", "ready")'
---
---
status: done
---
# Bad Fixture
