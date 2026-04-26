---
id: 92
title: File roles and placeholder grammar
status: "🔲"
summary: >-
  Replace per-file ignore entries for schema and template
  Markdown with a composable model: user-declarable file
  roles that reuse the existing rule-config syntax, plus
  a placeholder grammar exposed as per-rule settings.
---
# File roles and placeholder grammar

## Goal

Remove the need to put schema, template, fragment, and
other "input-to-other-Markdown" files in `ignore:`.
Every Markdown file should be lintable under a declared
**role**. Roles are configured with the same syntax as
the existing rule configuration. No new bespoke schema.

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

The fix is two pieces, both built on existing plumbing:

1. **User-declared roles.** A role is a named block
   with the same shape as an `overrides:` entry minus
   `files:`. mdsmith ships **no built-in roles**;
   each project declares its own vocabulary. Examples
   below use fictional names (`recipe`, `tip`,
   `worksheet`) — the names belong to the project,
   not to mdsmith. Implicit assignment is itself
   user-configured (see `implicit-roles:`).
2. **Placeholder support as rule settings.** Each rule
   that wants to recognize template tokens (`# ?`,
   `## ...`, `{var}`, CUE front-matter values) exposes
   a `placeholders:` setting through the existing
   `Configurable` interface. Roles enable it the same
   way any other rule setting is enabled.

## Design

### Role declaration

A role is a named block under `roles:`. Its body has
the **same shape as an override entry** (`rules:`,
`front-matter:`, etc.) — minus `files:`, since files
are bound to roles separately. The role name (`recipe`
below) is whatever the project picks:

```yaml
roles:
  recipe:                      # project-chosen name
    front-matter: false
    rules:
      first-line-heading:
        placeholders: [var-token, heading-question]
      cross-file-reference-integrity:
        placeholders: [var-token]
      paragraph-readability: false
      paragraph-structure: false
      no-emphasis-as-heading: false
```

Roles merge with the same rules as overrides — later
wins, settings deep-merge.

### Role assignment

A file's effective role list is built from three
sources, concatenated in this order:

1. front-matter `roles:` field (a YAML list; a
   single-role file still uses a one-item list);
2. matching entries in `role-assignment:` (config
   order; each entry's roles in the order listed);
3. matching entries in `implicit-roles:` (config
   order).

Duplicate names are dropped after their first
occurrence. Referencing an undeclared role is a
config error.

`implicit-roles:` maps **reference sources** to a
role name. A reference source is any place where one
Markdown file references another. The linter's core
has no hardcoded role names; the project picks them:

```yaml
implicit-roles:
  - source: required-structure.schema  # mdsmith field
    role: recipe                       # project name
  - source: include
    role: tip
  - source: catalog
    role: tip
```

With this in place, declaring a `recipe` role plus
`required-structure.schema: plan/proto.md` is enough
to drop `plan/proto.md` from `ignore:`.

### Composability

1. **Role merge follows override merge.** Roles
   apply in *effective-list* order: front-matter,
   then `role-assignment:`, then `implicit-roles:`,
   each in config order; duplicates dropped after
   first occurrence. The file's own glob overrides
   apply last. The result is deterministic across
   runs because order is driven by lists, not maps.
2. **Lint-once.** Content pulled in via `<?include?>`
   or `<?catalog?>` is linted in its source file
   under its source role; the host file does not
   re-diagnose embedded bytes.
3. **Schema vs subject separation.** A schema file's
   front matter is validated as a *schema* (CUE
   patterns OK); referenced subject files are
   validated as *data*.
4. **No role names in rule code.** Rules read their
   own settings; they never branch on a role name.
   New roles cannot regress existing behavior.

### Placeholder support

Placeholder grammars are named tokens (`var-token`
for `{identifier}`, `heading-question` for `# ?`,
`cue-frontmatter`, …). Each rule that opts in declares
a `placeholders:` setting via `Configurable` — same
machinery as any other rule setting. Roles enable the
right tokens per rule:

```yaml
roles:
  recipe:
    rules:
      cross-file-reference-integrity:
        placeholders: [var-token]
```

Adding a new placeholder grammar is a code change in
one helper plus opt-ins per rule; it is not coupled
to any role. Migration removes `ignore:` entries one
role at a time, keeping the linter green at each
step; untagged files behave as today.

## Examples

> Example role names below (`recipe`, `tip`,
> `worksheet`) are deliberately fictional — they
> aren't mdsmith built-ins. Real projects pick
> names that fit their domain.

### Explicit roles in front matter

```markdown
---
roles: [recipe]
id: '=~"^MDS[0-9]{3}$"'
status: '"ready" | "not-ready"'
---
# {id}: {name}
```

A file with multiple roles uses a multi-element list:
`roles: [tip, worksheet]`. Merge order matches list
order.

### Glob-based assignment

```yaml
role-assignment:
  recipe:
    - "**/proto.md"
  worksheet:
    - ".github/PULL_REQUEST_TEMPLATE.md"
  tip:
    - "docs/_partials/**"
```

### Implicit role from a config reference

```yaml
overrides:
  - files: ["plan/*.md"]
    rules:
      required-structure:
        schema: plan/proto.md   # → role 'recipe'
```

Combined with the `implicit-roles:` block above,
`plan/proto.md` gains role `recipe` automatically.
No `ignore:`, no front-matter tag, no glob needed.

### Composability — merged role config

`docs/_partials/setup-snippet.md` carries both `tip`
(it's included elsewhere) and `worksheet` (it has
placeholders). The effective config is a deep-merge
of both role bodies in declared order. File-glob
overrides apply last. Inspect with
`mdsmith config show <path>`.

### Lint-once for embeds

`docs/_partials/intro.md` (role `tip`) is included by
`docs/index.md` via `<?include?>`. The fragment is
linted once, in its own file, under `tip`. The
embedded bytes inside the host file are not
re-checked.

## Tasks

1. Inventory current `ignore:` and per-file
   `overrides:` entries; classify each by intended
   role.
2. Add the `roles:`, `role-assignment:`, and
   `implicit-roles:` config keys; reuse the existing
   override merge to apply a role's body. The
   linter's core must reference no role names.
3. Add the front-matter `roles:` list field and wire
   the implicit-from-reference assignment driven by
   `implicit-roles:` (supported sources:
   `required-structure.schema`, `include`,
   `catalog`).
4. Add a placeholder helper (`var-token`,
   `heading-question`, `cue-frontmatter`, …) and a
   `placeholders:` setting on each rule that opts in
   (`first-line-heading`, `heading-increment`,
   `no-emphasis-as-heading`,
   `cross-file-reference-integrity`,
   `paragraph-readability`, `paragraph-structure`,
   front-matter validation).
5. Implement lint-once for `<?include?>` and
   `<?catalog?>` host files.
6. Update this repo's `.mdsmith.yml` to declare the
   roles it needs and drop the four `proto.md`
   entries from `ignore:`; confirm `mdsmith check .`
   stays green.
7. Document the model under
   `docs/background/archetypes/file-roles/` (new
   archetype page) with starter role declarations
   that projects can copy. Link from each affected
   rule README.
8. Add CLI commands `mdsmith config roles` (list
   declared roles with their merged bodies) and
   `mdsmith config show <file>` (print the resolved
   role set and merged rule config for a file).
9. Emit a clear config error when role assignment
   references an undeclared role name, with the
   suggested fix ("declare role `<name>` in
   `.mdsmith.yml`").

## Acceptance Criteria

- [ ] `mdsmith check .` passes with the four
      `proto.md` entries removed from `.mdsmith.yml`
      `ignore:`.
- [ ] A role declaration parses with the same syntax
      as an override entry (minus `files:`); override
      merge is reused for role merge (verified by
      test).
- [ ] Two project-declared roles compose correctly
      with each other and with file-glob overrides
      (covered by test).
- [ ] A file declaring multiple roles via
      `roles: [a, b]` in front matter merges them in
      list order (covered by test).
- [ ] Implicit role assignment is configured by
      `implicit-roles:`; no role name is referenced
      by mdsmith's core or shipped default config
      (enforced by grep test).
- [ ] Schema files declared by the project are
      linted under their chosen role: CUE-pattern
      front matter, `# ?` and `## ...` placeholders,
      and `{var}` tokens produce no diagnostics from
      rules whose `placeholders:` setting permits
      them.
- [ ] Content embedded into a host file via
      `<?include?>` or `<?catalog?>` produces
      diagnostics only against the source file, not
      the host (covered by test).
- [ ] Adding a new schema file under any directory
      requires no change to `ignore:` — the role is
      assigned implicitly from its
      `required-structure.schema` reference.
- [ ] Rule code contains no `if role == "..."`
      branches; rule behavior is selected through
      `Configurable` settings only (enforced by grep
      test).
- [ ] Referencing an undeclared role name produces a
      clear config error.
- [ ] `mdsmith config roles` lists declared roles
      with their merged bodies.
- [ ] `mdsmith config show <file>` prints the
      resolved role set and the merged rule config.
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports no issues
