---
id: 79
title: Nested front-matter access
status: "🔲"
summary: >-
  Use CUE path syntax in {field} placeholders
  for nested and quoted front-matter access.
---
# Nested front-matter access

Follow-on to the user-model work in
[plan 73](73_unify-template-directives.md).
Related to
[plan 75](75_single-brace-placeholders.md)
(`{field}` syntax).

Depends on: plan 75 (single-brace syntax must
land first so both syntaxes are updated
together).

## Goal

Use CUE path syntax for field access in schema
headings. A document with:

```yaml
---
params:
  subtitle: Overview
my-key: value
---
```

matches `# {params.subtitle}` and
`{"my-key"}` resolves to `value`. Catalog
rows use Go template dot access:
`{{.params.subtitle}}`.

## Context

Required-structure parses front matter into
`map[string]any` but stringifies values for
sync checks. Catalog reads front matter via
`readFrontMatter` in `catalog/rule.go`, also
flattening to strings. Neither path resolves
dotted keys into nested maps. Hugo users
expect `.Params.subtitle`; mdsmith does not
support this yet.

## Design

Use CUE path syntax inside `{...}` placeholders
and for catalog field resolution. One path
grammar across the whole tool.

- `{a.b.c}` resolves nested maps:
  `fm["a"].(map)["b"].(map)["c"]`
- `{"my-key".sub}` quotes non-identifier keys
  (hyphens, dots, spaces), same as CUE
- `{"a.b"}` is one key with a literal dot,
  distinct from `{a.b}` (two nested keys)
- If any step is not a map, emit a diagnostic:
  `front-matter key "a.b" is not a map`
- Catalog `{{.a.b}}` uses Go `text/template`
  native dot access for nested maps; quoted
  CUE-style keys (`{{"my-key"}}`) are not
  supported in catalog rows since Go templates
  have their own quoting via `index`

### Why CUE paths

The tool already uses CUE for front-matter
schema validation. Reusing CUE path syntax
for field access means one grammar to learn:

| Context             | Syntax              | Example             |
|---------------------|---------------------|---------------------|
| Schema front matter | CUE expr            | `'string & != ""'`  |
| Schema heading      | CUE path in `{...}` | `{params.title}`    |
| Quoted key          | CUE quoting         | `{"my-key".sub}`    |
| Catalog row         | Go template         | `{{.params.title}}` |

Catalog rows keep Go template syntax (they
already use it). Schema headings adopt CUE
paths. The two contexts remain visually
distinct (`{...}` vs `{{...}}`).

## Tasks

1. Update front-matter handling in
   `catalog/rule.go` (`readFrontMatter`) and
   `requiredstructure/rule.go`
   (`readDocFrontMatterRaw`/`stringifyFrontMatter`)
   to preserve nested `map[string]any` values.
2. Add a `resolveCUEPath(fm map[string]any,
   path string) (string, error)` helper that
   parses a CUE path (dot-separated, with
   quoted labels) and walks nested maps.
3. Update `requiredstructure/rule.go`:

  - `resolveFields` uses `resolveCUEPath`
  - `docFM` type changes to `map[string]any`
  - Update `fieldPattern` regex to capture
    CUE paths inside `{...}`: identifiers,
    dots, and quoted strings
    (e.g. `{a.b}`, `{"my-key".sub}`)

4. Update `catalog/generate.go`:

  - Template data uses `map[string]any`
  - Go `text/template` handles nested access
    natively via `.a.b`

5. Verify CUE schema derivation in
   `requiredstructure/rule.go` already handles
   nested front matter; only adjust if gaps
   remain for nested placeholder resolution.
6. Add unit tests for nested access in both
   required-structure and catalog.
7. Add fixtures with nested front matter.
8. Run `mdsmith check .` to verify.

## Acceptance Criteria

- [ ] `{a.b}` in a schema heading resolves
      nested front-matter key `a.b`
- [ ] `{"my-key".sub}` in a schema heading
      resolves quoted non-identifier key
- [ ] `{{.a.b}}` in catalog row resolves
      nested front-matter key `a.b`
- [ ] `{"a.b"}` resolves a single key with
      a literal dot (CUE quoting)
- [ ] Missing nested key emits a diagnostic
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports no
      issues
- [ ] `mdsmith check .` passes
