---
name: solid-architecture
description: >-
  Apply SOLID principles and clean architecture to
  Go and TypeScript codebases. Use when generating
  or revising a multi-step plan so the plan respects
  package boundaries; when designing a new package,
  splitting an existing one, or wiring a new
  cross-system surface (LSP, CLI, distribution
  shim, plugin); and when auditing a branch for
  accumulated boundary violations that should be
  scheduled as fixes. Trigger phrases include
  "design a package", "is this clean architecture",
  "refactor the boundary", "architecture audit",
  "plan a feature", "check main for arch debt",
  "review architecture", "solid principles".
user-invocable: true
argument-hint: "[mode: design | plan | audit]"
allowed-tools: >-
  Bash(git fetch:*), Bash(git log:*),
  Bash(git diff:*), Bash(git branch:*),
  LSP
---

# SOLID and clean architecture

This skill enforces SOLID and clean
architecture for Go and TypeScript code. It
runs in three workflows — design, plan, and
audit — and pulls language-level depth from
the sibling reference files.

Read [Go patterns][skill-go] and
[TypeScript patterns][skill-ts] for the
language-level rules. Read the
[cross-system contracts][skill-cross] page
for the patterns that govern external
surfaces. Use the [audit checklist][skill-audit]
to walk a branch in audit mode. If the
project also ships architecture docs under
`docs/development/architecture/` or similar,
read those for the project's actual layering
and named packages — they instantiate the
patterns this skill describes.

## Prerequisites

- Run each fenced code block as its own
  Bash call. Do not chain with `&&`. Do not
  prefix with an inline variable assignment
  (`VAR=x cmd`) or wrap the leading token in
  `$(...)`. The skill's `allowed-tools` list
  matches on the command prefix, so the
  allowed token (e.g. `git log`) must be the
  first thing the shell sees.
- Where a step needs a value from a prior
  command (a SHA, a path), paste the literal
  value into the next command rather than
  storing it in a shell variable.

## Modes

Pass the mode as the skill argument
(`/solid-architecture design`, `… plan`, or
`… audit`). If no argument is supplied, infer
from the conversation: editing a plan or RFC
document → plan; writing new Go or TypeScript
→ design; sweeping the main branch → audit.

### Design mode

Use design mode for code-shape decisions:

- A new package or module.
- Splitting an existing package whose
  responsibilities have drifted.
- A new feature added at a layer boundary.
- A new external surface — LSP capability,
  CLI subcommand, distribution shim, plugin
  entry point.

Workflow:

1. Identify which layer the change lives in.
   Cross-check against the project's layering
   map (if it ships one) or against the
   layering rules in [Go patterns][skill-go]
   or [TypeScript patterns][skill-ts].
2. List the interface(s) the change crosses
   on a layer boundary. Confirm the
   dependency direction matches.
3. Walk the principle checklist in
   [Go patterns][skill-go] or
   [TypeScript patterns][skill-ts].
4. Reject package or module names that mix
   responsibilities (`util`, `common`,
   `helpers`, `lib`, `misc`). Each package
   name must answer one question.
5. Write the failing test first (Red), then
   the smallest passing implementation
   (Green).

### Plan mode

Use when generating or revising a multi-step
plan, RFC, or task list. Plans lock in
architectural decisions; getting the layer
boundaries wrong here ripples into every
commit that follows.

Workflow:

1. For each task in the plan, name the
   package(s) or module(s) it touches and
   the layer(s) it crosses.
2. Mark any task that pushes a dependency
   the wrong direction (e.g. a low-level
   module importing a high-level one).
3. If a plan introduces a new interface,
   add a task for the test that locks the
   contract.
4. Annotate acceptance criteria with the
   SOLID principle each verifies, when one
   applies.
5. After editing, run the project's plan
   regeneration step (e.g. a catalog
   refresh) so plan indices stay current.

### Audit mode

Use to sweep the main branch for accumulated
violations. Suited to a recurring schedule
(e.g. via `/loop 1d /solid-architecture
audit`).

Workflow:

1. Read the last audit checkpoint from the
   project's architecture audit log (commonly
   under `docs/development/`). If the log
   does not exist, create it from the
   [audit checklist][skill-audit] §"Initial
   file" and start one month back; on a
   younger repo fall back to the root commit.
2. Diff that SHA against `origin/main` and
   walk the [audit checklist][skill-audit]
   over the touched files.
3. Append findings to the audit log with
   severity (`blocker`, `tax`,
   `nice-to-have`), the violating files, and
   the principle violated.
4. Do not fix in this run. Group blockers
   that share the same structural fix into
   one plan file referencing the audit
   entries; only split into more plans when
   a single one would exceed the project's
   max-file-length budget. For tax and
   nice-to-have, leave entries in the audit
   log.
5. Update `audit-from:` to the current
   `origin/main` SHA.

## References

The four sibling reference files live in
this skill directory:

- [Go patterns][skill-go] — SOLID applied to
  Go interfaces, packages, and `cmd/` +
  `internal/` layout.
- [TypeScript patterns][skill-ts] — SOLID
  for TypeScript modules and VS Code
  extension shapes.
- [Cross-system contracts][skill-cross] —
  LSP, CLI flags, config schemas,
  distribution shims, plugin manifests.
- [Audit checklist][skill-audit] — generic
  audit workflow, severity rubric, plan
  grouping rules.

[skill-go]: go.md
[skill-ts]: typescript.md
[skill-cross]: cross-system.md
[skill-audit]: audit-checklist.md

## Notes

- This skill never edits source code on its
  own. In design and plan modes it advises
  and rewrites prose; in audit mode it
  appends to the project's audit log and
  creates plan files for blockers.
- The audit log is Markdown; respect the
  project's lint rules (line-length budget,
  readability caps) when adding entries.
- When recommending a refactor, name the
  principle being upheld and the layer being
  protected. Vague "this would be cleaner"
  guidance is not actionable.
