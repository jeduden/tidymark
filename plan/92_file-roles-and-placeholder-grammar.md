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
   projects declare whatever vocabulary they need
   (`schema`, `template`, `agent-prompt`, …). Names
   like `schema` appear in this RFC because the
   linter looks for them when wiring implicit role
   assignment, but their config is up to the project.
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
are bound to roles separately.

```yaml
roles:
  schema:
    front-matter: false
    rules:
      first-line-heading:
        placeholders: [var-token, heading-question]
      cross-file-reference-integrity:
        placeholders: [var-token]
      paragraph-readability: false
      paragraph-structure: false
      no-emphasis-as-heading: false
      required-structure:
        mode: schema
```

Roles merge with the same rules as overrides — later
wins, settings deep-merge. mdsmith ships no roles by
default. Each project declares the roles it needs.
Starter configurations live in
`docs/background/archetypes/file-roles/`.

### Role assignment

A file's role set is the union of:

- explicit `role:` field in front matter;
- glob-based assignment in `.mdsmith.yml`
  (`role-assignment:` map, separate from role
  declaration);
- implicit roles inferred from config references
  (any path used as the `required-structure.schema`
  target gains role `schema`; any path referenced by
  `<?include?>` gains `fragment`).

Implicit assignment removes the four current ignore
entries. The project declares a `schema` role once.
Every `required-structure.schema` reference then
auto-tags its target. Referencing an undeclared role
name is a config error.

### Composability

1. **Role merge follows override merge.** Roles apply
   in declared order; the file's own glob overrides
   apply last. Resolution is deterministic and
   inspectable.
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
  schema:
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

### Explicit role in front matter

```markdown
---
role: schema
id: '=~"^MDS[0-9]{3}$"'
status: '"ready" | "not-ready"'
---
# {id}: {name}
```

### Glob-based assignment

Role assignment lives next to role declarations:

```yaml
role-assignment:
  schema:
    - "**/proto.md"
  template:
    - ".github/PULL_REQUEST_TEMPLATE.md"
  fragment:
    - "docs/_partials/**"
```

### Implicit role from a config reference

```yaml
overrides:
  - files: ["plan/*.md"]
    rules:
      required-structure:
        schema: plan/proto.md   # plan/proto.md → schema
```

The path `plan/proto.md` gains role `schema`
implicitly. No `ignore:`, no front-matter tag, no
glob needed.

### Adding a project-specific role

No rule code changes are needed; declare the role and
assign it:

```yaml
roles:
  agent-prompt:
    rules:
      paragraph-readability: false
      first-line-heading:
        placeholders: [var-token]

role-assignment:
  agent-prompt:
    - ".claude/prompts/**"
```

### Composability — merged role config

`docs/_partials/setup-snippet.md` carries both
`fragment` and `template`. The effective config is a
deep-merge of both role bodies in declared order.
File-glob overrides apply last. Inspect the result
with `mdsmith config show <path>`.

### Lint-once for embeds

```text
docs/index.md           role: document
└── <?include file: _partials/intro.md ?>

docs/_partials/intro.md role: fragment
```

`docs/_partials/intro.md` is linted under `fragment`.
The embedded bytes inside `docs/index.md` are not
re-checked.

### Schema vs subject

| File                                            | Role     | Front matter `id` validated as           |
|-------------------------------------------------|----------|------------------------------------------|
| `plan/proto.md`                                 | schema   | CUE pattern (`int & >=1`)                |
| `plan/92_file-roles-and-placeholder-grammar.md` | document | data; matched against `proto.md` pattern |

## Tasks

1. Inventory current `ignore:` and per-file
   `overrides:` entries; classify each by intended
   role.
2. Add the `roles:` and `role-assignment:` config
   keys; reuse the existing override merge to apply a
   role's body.
3. Add the front-matter `role:` field and the
   implicit-from-reference assignment path
   (`required-structure.schema` and `<?include?>`).
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
   roles it needs (`schema`, `fragment`, `template`,
   …) and drop the four `proto.md` entries from
   `ignore:`; confirm `mdsmith check .` stays green.
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
      (covered by test); no role names are present in
      mdsmith's shipped default config.
- [ ] Schema files declared by the project are
      linted under their `schema` role: CUE-pattern
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
