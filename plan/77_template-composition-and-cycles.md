---
id: 77
title: Template composition and cycle detection
status: "✅"
summary: >-
  Allow include directives in schema files for
  template composition, add cycle detection for
  all include paths, rename template to schema.
---
# Template composition and cycle detection

Part of the user-model work from
[plan 73](73_unify-template-directives.md).
Addresses
[#71](https://github.com/jeduden/mdsmith/issues/71)
(cycle detection) and
[#73](https://github.com/jeduden/mdsmith/issues/73)
(Hugo comparison -- composition gap).
Also addresses
[#68](https://github.com/jeduden/mdsmith/issues/68)
(`schema` rename clarifies user model).

Depends on: none (independent of other plans).
Update plan 74 guide after landing.

## Goal

Let schema files use `<?include?>` to share
structure across templates. Add cycle detection
to all include resolution paths so composition
is safe. Rename the config key from `template`
to `schema` to stop Hugo users reading it as a
rendering template.

## Context

Hugo-user blind trials (plan 73) found:

- 5/5 confused by the word "template" meaning
  validation schema, not rendering.
- 5/5 expected composition (Hugo partials and
  base templates compose freely).
- Issue #71 notes there is no cycle detection
  for include directives today.

Allowing `<?include?>` in schema files solves
the composition gap but also introduces a new
cycle risk: schema A includes fragment B which
includes fragment C which includes A. Cycle
detection must land in the same PR.

## Design

### Schema composition via `<?include?>`

During `parseTemplate`, process `<?include?>`
directives in the schema file. The included
fragment's headings are spliced into the schema
heading list at the include position. Front
matter from fragments is ignored (only the
root schema's front matter defines the CUE
constraints). `<?require?>` in a fragment is
merged into the root schema's constraints.

Example schema (`plan/proto.md`):

```markdown
---
id: 'int & >=1'
title: 'string & != ""'
---
<?require
filename: "[0-9]*_*.md"
?>
# ?

## Goal

One-sentence summary.

<?include
file: common/acceptance-criteria.md
?>
```

Where `common/acceptance-criteria.md` has:

```markdown
## Acceptance Criteria

- [ ] All tests pass: `go test ./...`
```

The parsed schema would have headings:
`# ?`, `## Goal`, `## Acceptance Criteria`.

Fragment files must be in the `ignore:` list
(they are not standalone documents).

### Cycle detection

Add a visited-file set to include resolution.
Applies to three paths:

1. Normal-file `<?include?>` (MDS021): track
   included paths during `fix` and `check`.
   If a file appears twice in the chain, emit
   `cyclic include: A.md -> B.md -> A.md`.
2. Schema `<?include?>` (new): track paths
   during `parseTemplate`. Same diagnostic.
3. Catalog glob (MDS019): if a matched file
   includes (via MDS021) the file containing
   the catalog, the catalog body would contain
   itself. Detect by checking whether the
   catalog-owning file appears in any matched
   file's include chain.

Also add a max depth (default 10) as a safety
net. Depth exceeding the limit is an error even
without a detected cycle.

### Config key rename: `template` -> `schema`

Rename the config key in required-structure
settings. No deprecation; update all config
and docs in this PR.

```yaml
# Before
required-structure:
  template: plan/proto.md

# After
required-structure:
  schema: plan/proto.md
```

### Diagnostic for misplaced `<?require?>`

When `<?require?>` appears in a file that is
not being parsed as a schema, emit:

```text
MDS020 <?require?> is only recognized in
schema files; this directive has no effect here
```

## Tasks

1. ~~Add visited-file tracking to MDS021 include
   resolution in `internal/rules/include/`.~~
   Done.
2. ~~Extend `parseTemplate` in
   `internal/rules/requiredstructure/rule.go`
   to process `<?include?>` directives.~~
   Done.
3. ~~Add diagnostic for `<?require?>` in
   non-schema files.~~ Done.
4. ~~Rename `template` to `schema` in
   required-structure config.~~ Done.
5. ~~Update fixtures.~~ Done.
6. ~~Update docs.~~ Done.
7. ~~Run `mdsmith check .` to verify.~~ Done.

## Acceptance Criteria

- [x] Schema files can use `<?include?>` to
      pull in heading fragments
- [x] Cycle detection works for normal-file
      includes (direct and indirect)
- [x] Cycle detection works for schema-file
      includes
- [x] Max include depth (10) is enforced
- [x] `<?require?>` in a non-schema file emits
      a warning
- [x] Config key is `schema`, not `template`
- [x] All `.mdsmith.yml` overrides updated
- [x] All tests pass: `go test ./...`
- [x] `go tool golangci-lint run` reports no
      issues
- [x] `mdsmith check .` passes
