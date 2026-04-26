---
id: 92
title: File roles and placeholder grammar
status: "🔲"
summary: >-
  Replace per-file ignore entries for schema and template
  Markdown with a composable model: declared file roles
  plus a shared placeholder grammar that every rule can
  consult.
---
# File roles and placeholder grammar

## Goal

Remove the need to put schema, template, fragment, and
other "input-to-other-Markdown" files in `ignore:`.
Make every Markdown file in the repo lintable under a
declared **role**, and let rules opt into a shared
**placeholder grammar** so template tokens stop tripping
content checks.

## Background

Today `.mdsmith.yml` ignores four `proto.md` files
(`.claude/skills/`, `plan/`, `internal/rules/`,
`docs/security/`). Their front matter holds CUE schema
patterns and their bodies hold placeholder text like
`# ?`, `## ...`, and `{var}`.

The `required-structure` rule knows these files are
schemas, but other rules do not, so the only escape
hatch is `ignore:`. The same pressure recurs for any
file that seeds or validates other Markdown — templates,
includes, demo sources, future schemas.

The fix is two orthogonal concepts kept composable:

1. **File roles.** Promote the implicit categories to
   first-class roles (`document`, `schema`, `template`,
   `fragment`, `fixture`). Each rule declares how it
   reacts per role: applies, skips, or applies-with-
   relaxation. Roles are a *set*, so a file can be both
   `guide` and `fragment` and the disabled-rule sets
   union.
2. **Placeholder grammar.** Lift the ad-hoc escapes
   (`# ?`, `## ...`, `{var}`, CUE front-matter patterns,
   existing `<?allow-empty-section?>` markers) into one
   named vocabulary that every rule consults via a
   shared helper. Rules get two knobs:
   "recognize placeholders?" and "ignore placeholder
   tokens?".

## Design

### Composability rules

These invariants keep the model from collapsing into
per-file special cases:

1. **Role union, not override.** Combining roles only
   *removes* checks. A second role never silently
   re-enables a check the first role disabled.
2. **Lint-once.** Content pulled in via `<?include?>`
   or `<?catalog?>` is linted in its source file under
   its source role; the host file does not re-diagnose
   embedded bytes.
3. **Schema vs subject separation.** A schema file's
   front matter is validated as a *schema* (CUE
   patterns OK); referenced subject files are
   validated as *data*. Same mechanism applies to any
   future "describes another file" relationship.
4. **No per-file-type rule forks.** Rule code never
   branches on `if proto.md`. It branches on
   `role + placeholder presence` only.

### Role assignment

A file's role set is the union of:

- explicit `role:` field in front matter, and
- glob-based assignment in `.mdsmith.yml`, and
- implicit roles inferred from config references
  (e.g. any path used as a `required-structure: schema:`
  target gains the `schema` role automatically; any
  path referenced by `<?include?>` gains `fragment`).

Implicit assignment is what removes the four current
ignore entries without any user action.

### Placeholder grammar

Initial vocabulary (extensible, registered centrally):

- `# ?`, `## ?`, `### ?` — placeholder heading text
- `## ...` — placeholder section name
- `{identifier}` — variable interpolation token
- CUE-style front-matter values (string predicates,
  regex literals, disjunctions)
- existing `<?allow-empty-section?>` and other PI
  markers stay as-is

Each rule that opts in receives a token-stripped or
token-aware view of the AST so detection logic stays
local to one helper.

### Migration

Migration removes ignore entries one role at a time and
keeps the linter green at every step. Behavior on files
not yet tagged with a role is unchanged.

## Tasks

1. Inventory current `ignore:` and per-file `overrides:`
   entries; classify each by intended role
   (schema / template / fragment / fixture / doc).
2. Define the role enum and a `role:` front-matter
   field; wire glob-based and implicit assignment
   from config references.
3. Define the placeholder grammar registry and a
   shared helper rules can consult.
4. Add a `RoleAware` rule capability (analogous to
   `Configurable`): rules declare per-role behavior;
   default is "applies as today".
5. Teach the rules currently tripped by `proto.md`
   (`first-line-heading`, `heading-increment`,
   `no-emphasis-as-heading`,
   `cross-file-reference-integrity`,
   `paragraph-readability`, `paragraph-structure`,
   front-matter validation) to honor the placeholder
   grammar when role includes `schema` or `template`.
6. Implement lint-once for `<?include?>` and
   `<?catalog?>` host files so embedded fragment
   content is not re-diagnosed.
7. Drop the four `proto.md` entries from `ignore:` and
   confirm `mdsmith check .` stays green.
8. Document the role model and placeholder grammar
   under `docs/background/archetypes/` (new archetype
   page) and link from each affected rule README.
9. Add a config-error path for unknown roles and for
   placeholder tokens that escape into non-placeholder
   files.

## Acceptance Criteria

- [ ] `mdsmith check .` passes with the four
      `proto.md` entries removed from `.mdsmith.yml`
      `ignore:`.
- [ ] Schema files are linted under role `schema`:
      CUE-pattern front matter, `# ?` and `## ...`
      placeholders, and `{var}` tokens produce no
      diagnostics from rules that opted into the
      placeholder grammar.
- [ ] A file declaring two roles receives the **union**
      of disabled checks; no role silently re-enables a
      check another role disabled (covered by test).
- [ ] Content embedded into a host file via
      `<?include?>` or `<?catalog?>` produces
      diagnostics only against the source file, not the
      host (covered by test).
- [ ] Adding a new schema file under any directory
      requires no change to `ignore:` — the role is
      assigned implicitly from its
      `required-structure: schema:` reference.
- [ ] Rule code contains no `if filename == "proto.md"`
      style branches; behavior is selected via role and
      placeholder helpers only (enforced by review +
      grep test).
- [ ] Unknown role names and stray placeholder tokens
      in non-placeholder files produce clear config or
      lint errors.
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports no issues
