---
id: 156
title: 'Composable required-structure schemas across multiple kinds'
status: '✅'
summary: >-
  Two kinds whose required-structure schemas
  differ overwrite each other under deep-merge.
  This plan adds composition so a
  directive-rule-readme kind layers on top of
  rule-readme without losing either constraint
  set.
model: 'opus'
depends-on: [97, 146]
---
# Composable required-structure schemas across kinds

## ...

<?allow-empty-section?>

## Goal

A file resolved by N kinds gets one effective
required-structure schema. That schema must be the
composition of each kind. Today the last-written
schema wins.

The driving use case is the new
directive-rule-readme kind from PR #274. It must
layer a required Pattern section onto rule-readme.
The same shape applies later to runbook + on-call
or plan + epic pairs.

## ...

<?allow-empty-section?>

## Background

The config merge layer in `internal/config/`
already deep-merges rule settings across layers
(defaults, kinds, overrides). Scalar replacement
plus map-key merging covers most rules. See
[merge.go](../internal/config/merge.go) for the
implementation.

required-structure is the outlier. Its setting is
either `schema:` (a path to a `proto.md` file) or
`inline-schema:` (a parsed Schema). Two kinds that
both set `schema:` deep-merge as scalars and the
second wins. One kind sets `schema:` and another
sets `inline-schema:`: MDS020 today picks one and
drops the other. Neither path composes.

## ...

<?allow-empty-section?>

## Tasks

1. ✅ Baseline note landed in the
   [cross-system doc](../docs/development/architecture/cross-system.md).
   It describes the new composition rule. The
   note doubles as the public contract.

2. ✅ Chose composition rule option 1.

  - Sections concatenate.
  - Same-heading scopes merge.
  - Frontmatter conjoins via CUE `&`.
  - Stricter `closed:` wins.
  - `Require.Filename` picks the first
    non-empty pattern.
  - Conflicting patterns error.
  - Acceptance test:
    [compose_test.go](../internal/schema/compose_test.go).

3. ✅ Merge layer accumulates a
   `schema-sources` list across layers. Each
   layer that sets `schema:` or `inline-schema:`
   contributes one entry. The rule loads the
   list and calls
   [`schema.Compose`](../internal/schema/compose.go)
   at check time.

   Tests in
   [schema_kinds_test.go](../internal/config/schema_kinds_test.go)
   cover disjoint sections, disjoint frontmatter,
   and the kind-plus-override path.
   [compose_test.go](../internal/rules/requiredstructure/compose_test.go)
   exercises the end-to-end Check path.

4. ✅ The four directive READMEs now resolve to
   both `rule-readme` and `directive-rule-readme`
   in [.mdsmith.yml](../.mdsmith.yml). The
   [directive-proto.md](../internal/rules/directive-proto.md)
   schema lost the duplicated headings. It now
   declares only the `Pattern` section. The
   `nature` frontmatter narrows to
   `"directive"`. Each directive README moved
   `Pattern` to the end so the composed section
   order matches: `rule-readme` first, then
   `directive-rule-readme`.

5. ✅ Composition rule lives in
   [schemas.md](../docs/guides/schemas.md) with
   the worked
   `rule-readme + directive-rule-readme`
   example. The
   [MDS020 README](../internal/rules/MDS020-required-structure/README.md)
   points at the guide.

## ...

<?allow-empty-section?>

## Acceptance Criteria

- [x] A file resolving to two kinds with disjoint
  required sections fails `mdsmith check` until
  both sets are present.
- [x] A file resolving to two kinds with disjoint
  required front-matter keys fails `mdsmith check`
  until both sets are present.
- [x] `internal/rules/directive-proto.md` no
  longer duplicates rule-readme's `Config`,
  `Examples`, and `Meta-Information` headings; it
  declares only Pattern additions.
- [x] [docs/guides/schemas.md](../docs/guides/schemas.md)
  documents the composition rule with a two-kind
  worked example.
- [x] All tests pass: `go test ./...`.
- [x] `go tool golangci-lint run` reports no
  issues.

## ...

<?allow-empty-section?>
