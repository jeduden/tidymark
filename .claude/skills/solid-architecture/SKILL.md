---
name: solid-architecture
description: >-
  Apply SOLID principles and clean architecture to
  mdsmith's Go core (cmd/, internal/) and TypeScript
  VS Code extension (editors/vscode/). Use when
  generating or revising a plan in plan/ so the plan
  respects package boundaries; when designing a new
  internal/ package, splitting an existing one, or
  wiring a new cross-system surface (LSP, VS Code
  commands, npm/PyPI/asdf/mise shims, Claude plugin);
  and when auditing the main branch for accumulated
  boundary violations that should be scheduled as
  fixes. Trigger phrases include "design a package",
  "is this clean architecture", "refactor the
  boundary", "architecture audit", "plan a feature",
  "check main for arch debt", "review architecture",
  "solid principles".
user-invocable: true
argument-hint: "[mode: design | plan | audit]"
---

# SOLID and clean architecture for mdsmith

Keep mdsmith's Go core, TypeScript extension,
and integration surfaces aligned with SOLID
and clean architecture.

The Go linter lives in `cmd/` and `internal/`.
The VS Code extension lives in
`editors/vscode/`. The distribution shims
live in `npm/` and `python/`. The Claude
plugin lives in `.claude-plugin/`. Each layer
depends on stable contracts. Drift is
expensive to unwind later.

This skill operates in three modes: design,
plan, and audit. Pick a mode based on the
current task. Then defer to the
language-specific reference for depth.

## Modes

Pass the mode as the skill argument
(`/solid-architecture design`, `… plan`, or
`… audit`). If no argument is supplied, infer
from the conversation: editing a plan file →
plan; writing new Go or TypeScript → design;
sweeping `origin/main` → audit.

### Design mode

Use design mode for code-shape decisions. It
fits any of:

- A new `internal/` package, or splitting an
  existing one.
- A new rule or fix variant.
- A new LSP capability.
- A new cross-system surface: VS Code
  command, distribution channel, plugin
  entry.

Workflow:

1. Identify which layer the change lives in
   (see "Project layering" below).
2. List the interface(s) the change crosses
   on a layer boundary. Confirm the
   dependency direction matches the layering
   map.
3. Walk the principle checklist in
   [references/go.md](references/go.md) or
   [references/typescript.md](references/typescript.md).
4. Reject package names that mix
   responsibilities (`util`, `common`,
   `helpers`, `lib`). Each package name must
   answer one question.
5. Write the failing test first (Red), then
   the smallest passing implementation
   (Green). This matches the TDD discipline
   in CLAUDE.md.

### Plan mode

Use when generating or revising a file under
`plan/`. Plans drive multi-step work and lock
in architectural decisions; getting the layer
boundaries wrong here ripples into every
commit that follows.

Workflow:

1. For each task in the plan, name the
   package(s) it touches and the layer(s) it
   crosses.
2. Mark any task that pushes a dependency the
   wrong direction (e.g. `internal/engine`
   importing `internal/rules/...`).
3. If a plan introduces a new interface,
   add a task for the test that locks the
   contract.
4. Annotate acceptance criteria with the
   SOLID principle each verifies, when one
   applies.
5. After editing, run
   `mdsmith fix PLAN.md` so the catalog
   table stays current (per CLAUDE.md).

### Audit mode

Use to sweep `origin/main` for accumulated
violations. Suited to a recurring schedule
via `/loop 1d /solid-architecture audit`.

Workflow:

1. Read the last audit checkpoint from
   `docs/development/architecture-audit.md`
   front matter (`audit-from:`). If the file
   does not exist, create it from
   [references/audit-checklist.md](references/audit-checklist.md)
   §"Initial file" and start from one month
   back.
2. Diff that SHA against `origin/main` and
   walk
   [references/audit-checklist.md](references/audit-checklist.md)
   over the touched files.
3. Append findings to the audit log with
   severity (`blocker`, `tax`,
   `nice-to-have`), the violating files, and
   the principle violated.
4. Do not fix in this run. For each blocker,
   create a new plan file under `plan/` that
   references the audit entry. For tax and
   nice-to-have, leave entries in the audit
   log.
5. Update `audit-from:` to the current
   `origin/main` SHA.

## The principles, restated

The five SOLID principles, as concrete
constraints in this codebase:

- **Single responsibility**: every package
  under `internal/` answers one question.
  `internal/lint` answers "does this file
  violate a rule?"; `internal/fix` answers
  "what edits make it stop violating?". Do
  not collapse them because a function feels
  shared.
- **Open/closed**: new rules and fixes are
  added by registering a new package under
  `internal/rules/<id>-<name>/`, not by
  editing the engine. New conventions extend
  via deep-merge config layers, not new
  switches in `internal/engine`.
- **Liskov substitution**: every `rule.Rule`
  implementation must work in every engine
  call site without special-casing. A rule
  that needs the engine to know its name is
  a Liskov violation — widen the interface
  or push the special case down.
- **Interface segregation**: the `rule`
  package defines small interfaces (`Rule`,
  `Fixer`, `ListMerger`, …) so a rule only
  depends on the capabilities it uses. Do
  not add methods to `Rule` because one rule
  wants them.
- **Dependency inversion**: high-level code
  depends on interfaces, not concretes. The
  engine talks to `rule.Rule`, never to a
  specific rule package. The VS Code
  extension talks to the LSP server over
  the protocol, never to a Go type.

## Project layering

Dependencies flow downward only. A higher
layer may import a lower one; the reverse is
a violation.

Go side:

```text
cmd/mdsmith                (entry, wiring)
  └─> internal/engine      (orchestration)
        └─> internal/{rule, fix, config,
                      output, lint, lsp,
                      discovery, schema, …}
              └─> internal/{mdtext, globpath,
                            yamlutil, log,
                            placeholders}
                  (pure helpers; no
                   cross-deps among siblings)
```

Plus the rule plugins:

```text
internal/rules/<id>-<name>/
  └─> internal/rule (interfaces only)
  └─> internal/mdtext (parse helpers)
  ✗ MUST NOT import internal/engine
  ✗ MUST NOT import another rule package
```

TypeScript side:

```text
editors/vscode/src/extension.ts   (entry)
  └─> wiring.ts                   (compose)
        └─> commands/*            (one each)
              └─> binary.ts       (find +
                                   exec)
              └─> commands/runner.ts
                                  (typed I/O
                                   to binary)
```

Cross-system contracts live at the very edge.
They include the LSP wire protocol, the
`.mdsmith.yml` schema, generated section
markers, the plugin manifest, and shim entry
points. They follow their own versioning
rules. See
[references/cross-system.md](references/cross-system.md).

## Anti-patterns to reject on sight

- A new Go package named `util`, `common`,
  `helpers`, or `lib`.
- A rule package that imports another rule
  package.
- A command (TS or Go) that builds and
  parses Markdown inline instead of
  delegating to `internal/mdtext`.
- A distribution shim (`npm/`, `python/`)
  that reimplements binary logic instead of
  exec'ing it.
- A `.mdsmith.yml` field consumed only
  inside one rule package — promote it to
  that rule's settings or push the consumer
  back to where the data is owned.
- A test that mocks `rule.Rule` instead of
  using a real Markdown fixture; mocks at
  the interface boundary suggest the
  boundary is wrong.
- A TypeScript command that imports another
  command.
- A new public function in `internal/engine`
  whose only caller is one rule — move it to
  `internal/rule` or `internal/mdtext`.

## References

- [Go: SOLID and clean architecture in
  internal/](references/go.md)
- [TypeScript: SOLID in the VS Code
  extension](references/typescript.md)
- [Cross-system contracts (LSP, shims,
  plugins, config)](references/cross-system.md)
- [Audit checklist for sweeping
  origin/main](references/audit-checklist.md)

## Notes

- This skill never edits source code on its
  own. In design and plan modes it advises
  and rewrites prose; in audit mode it
  appends to
  `docs/development/architecture-audit.md`
  and creates plan files for blockers.
- The audit log is Markdown, so
  `mdsmith check .` will run over it; keep
  entries within the project's line-length
  budget (80 chars outside code blocks,
  tables, and URLs).
- When recommending a refactor, name the
  principle being upheld and the layer being
  protected. Vague "this would be cleaner"
  guidance is not actionable.
