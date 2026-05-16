---
id: 169
title: 'Enforce terminal Meta-Information and render it from frontmatter'
status: "🔲"
model: opus
depends-on: [156]
summary: >-
  The rule-readme schema allows sections after
  Meta-Information and its bullets are hand-typed.
  Make the schema reject any later section and sync
  the bullets from frontmatter via {field}.
---
# Rule-README Meta-Information ordering and sync

## Goal

Every rule README must end with its
`## Meta-Information` section. That section's
bullets must come from the README front matter, not
hand-typed values. The rule-readme schema enforces
both. A section after Meta-Information fails
`mdsmith check`. A bullet that disagrees with front
matter is flagged and fixed.

## Background

The rule-readme kind loads its required-structure
schema from
[internal/rules/proto.md](../internal/rules/proto.md).
That schema ends with a trailing `## ...` wildcard
after `## Meta-Information`. So a README may add
sections after it and still pass. The schema does
not enforce Meta-Information as the last section.

The Meta-Information body is hand-authored. Its
`ID`, `Name`, `Status`, and `Category` bullets
restate the `id`, `name`, `status`, and `category`
front-matter keys. Nothing checks that they agree.

[docs/guides/schemas.md](../docs/guides/schemas.md)
notes that MDS020's file-schema check still uses
its legacy parser. A `{field}` token in a
`proto.md` body is treated as a wildcard, not
resolved against front matter. Frontmatter-body
`{field}` sync is listed there as an unwired
follow-up. This plan wires it for the
Meta-Information case.

## Tasks

1. Drop the trailing `## ...` wildcard in
   [internal/rules/proto.md](../internal/rules/proto.md).
   The parsed schema then treats Meta-Information as
   the last scope. Add a test that a later section
   reports `got <present>, expected not declared in
   schema`.

2. Wire MDS020's file-schema path so a `{field}`
   token in the Meta-Information body resolves the
   document front-matter value. Reuse the
   `fmvar(...)` resolution the heading matcher
   already uses. `mdsmith fix` rewrites a stale
   bullet to the front-matter value.

3. Update
   [internal/rules/proto.md](../internal/rules/proto.md)
   so the bullets use `{id}`, `{name}`, `{status}`,
   `{category}`. Drop the hand-kept `CATEGORY`
   literal.

4. Revalidate every `internal/rules/MDS*/README.md`.
   Move any content after Meta-Information ahead of
   it. Reconcile bullet values with front matter so
   `mdsmith check .` passes.

5. Update
   [docs/guides/schemas.md](../docs/guides/schemas.md)
   and the
   [section-schema reference](../docs/reference/section-schema.md).
   Record that frontmatter-body `{field}` sync is
   now wired for file schemas.

## Acceptance Criteria

- [ ] A rule README with any section after
  `## Meta-Information` fails `mdsmith check`.
- [ ] A stale `ID`, `Name`, `Status`, or `Category`
  bullet is flagged and `mdsmith fix` rewrites it
  from front matter.
- [ ] [internal/rules/proto.md](../internal/rules/proto.md)
  has no trailing wildcard after Meta-Information
  and no `CATEGORY` literal.
- [ ] All `internal/rules/MDS*/README.md` pass
  `mdsmith check .`.
- [ ] All tests pass: `go test ./...`.
- [ ] `go tool golangci-lint run` reports no issues.
