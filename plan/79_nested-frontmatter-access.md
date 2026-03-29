---
id: 79
title: Nested front-matter access
status: "🔲"
summary: >-
  Support dot-separated keys in {field} and
  {{.field}} for nested YAML front matter.
---
# Nested front-matter access

Part of the user-model work from
[plan 73](73_unify-template-directives.md).
Related to
[plan 75](75_single-brace-placeholders.md)
(`{field}` syntax).

Depends on: plan 75 (single-brace syntax must
land first so both syntaxes are updated
together).

## Goal

`{params.sub}` in a schema heading and
`{{.params.sub}}` in a catalog row both resolve
nested YAML front matter. A document with:

```yaml
---
params:
  subtitle: Overview
---
```

matches `# {params.subtitle}` in the schema
and renders `{{.params.subtitle}}` as
`Overview` in catalog output.

## Context

Today both required-structure and catalog use
a flat `map[string]string` for front matter.
Nested YAML is silently flattened or ignored.
Hugo users expect `.Params.subtitle` to work;
mdsmith currently does not support this.

## Design

Replace the flat `map[string]string` with a
nested `map[string]any` and resolve dot-
separated keys by walking the map.

- `{a.b.c}` splits on `.` and walks:
  `fm["a"].(map)["b"].(map)["c"]`
- If any step is not a map, emit a diagnostic:
  `front-matter key "a.b" is not a map`
- Top-level keys with literal dots (e.g.
  `"a.b": value`) take precedence over nested
  lookup to avoid breaking existing configs
- Catalog `{{.a.b}}` already works in Go
  `text/template` if the data is a nested map;
  only the data structure needs to change

## Tasks

1. Change front-matter parsing in
   `internal/lint/frontmatter.go` to return
   `map[string]any` instead of
   `map[string]string`.
2. Add a `resolvePath(fm map[string]any,
   path string) (string, error)` helper that
   splits on `.` and walks nested maps.
3. Update `requiredstructure/rule.go`:

  - `resolveFields` uses `resolvePath`
  - `docFM` type changes to `map[string]any`

4. Update `catalog/generate.go`:

  - Template data uses `map[string]any`
  - Go `text/template` handles nested access
    natively via `.a.b`

5. Update CUE schema derivation in
   `requiredstructure/rule.go` to handle
   nested front matter (nested CUE structs).
6. Add unit tests for nested access in both
   required-structure and catalog.
7. Add fixtures with nested front matter.
8. Run `mdsmith check .` to verify.

## Acceptance Criteria

- [ ] `{a.b}` in a schema heading resolves
      nested front-matter key `a.b`
- [ ] `{{.a.b}}` in catalog row resolves
      nested front-matter key `a.b`
- [ ] Literal dot-key `"a.b": val` takes
      precedence over nested lookup
- [ ] Missing nested key emits a diagnostic
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports no
      issues
- [ ] `mdsmith check .` passes
