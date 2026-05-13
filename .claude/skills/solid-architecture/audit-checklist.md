---
title: Audit checklist
summary: >-
  Generic step-by-step audit workflow for
  sweeping a branch for SOLID and boundary
  violations. Project-agnostic.
---
# Audit checklist

Walk this list during an audit run. The
audit is a survey; do not edit offending
code in the same run.

For each finding capture:

- File(s) involved.
- Principle violated (one of: SRP, OCP,
  LSP, ISP, DIP) or layering rule
  ("dependency direction", "boundary
  contract", etc.).
- Severity: `blocker`, `tax`, or
  `nice-to-have`.
- Suggested fix — a short sentence, or a
  bullet list if a single sentence would
  trip the project's readability rules
  on the audit log.

The audit log lives in the project's
docs section (commonly under
`docs/development/`). The project's own
docs say where exactly and what lint
budget applies.

## Initial file

If the audit log does not exist yet,
create it with this shape:

```markdown
---
title: Architecture audit log
summary: >-
  Running log of SOLID and clean-
  architecture findings on the main
  branch. Appended in audit mode;
  blockers are also filed as plans.
audit-from: <commit SHA one month ago>
---
# Architecture audit log

Maintained by the solid-architecture
skill in audit mode.

## Audit YYYY-MM-DD (range:
<from-sha>..<to-sha>)

### blockers

### tax

### nice-to-have
```

Pick a starting SHA one month back
unless the user supplies a different
baseline.

If `git log --before="1 month ago"`
returns empty on a young repo, fall back
to the root commit:

```bash
git rev-list --max-parents=0 origin/main
```

The first SHA in that output is the
baseline.

## Step 1: refresh the checkpoint

Read the last audit SHA from the audit
log's front matter (`audit-from:`). Call
it `<from-sha>` below; paste the literal
SHA into each command rather than
capturing it into a shell variable (the
allowed-tools allowlist matches on the
command's leading token, so a
`VAR=… cmd` prefix or `$(…)` capture
would block the call).

Fetch:

```bash
git fetch origin main
```

List the files touched since the
checkpoint — replace `<from-sha>` with
the literal SHA before running:

```bash
git log --name-only --pretty=format: <from-sha>..origin/main
```

Keep the file list as the "touched set"
for the rest of this audit; do not
assign it to a shell variable.

When auditing in a worktree, first
verify the worktree has the project
architecture docs (if any) by listing
the docs directory. Concurrent worktree
creation can race; catch the mismatch
before walking.

## Step 2: language-level layering checks

For each source file in the touched set,
walk the checklist appropriate to its
language. Use [Go patterns](go.md) for
`.go` files and [TypeScript patterns](typescript.md)
for `.ts` / `.tsx` files. The common
checks across both languages:

- Does the file import only from lower
  layers? Cross-check against the
  project's layering map.
- Is its package or module still
  answering one question? If it has
  grown three new files in unrelated
  areas, flag for split (SRP).
- Are new implementations of a port
  interface using only the port and not
  reaching into the orchestrator? (DIP)
- Does any new function in the
  orchestrator only serve one
  implementation? Flag for relocation.
  (SRP)
- Are new interfaces minimal? An
  interface with one implementor is a
  smell unless it sits on a boundary.
  (ISP)
- Are there new packages or modules
  named `util`, `common`, `helpers`,
  `lib`, `misc`? Always a blocker (SRP).

## Step 3: cross-system contract checks

Use [cross-system contracts](cross-system.md)
as the reference for each surface.

- Did the config schema gain a field?
  If yes, is it documented? Is it
  backward compatible? If not, blocker.
- Did CLI flags change? Run the CLI
  help command and diff against the
  reference docs. Any renamed or
  removed flags are blockers without a
  deprecation window.
- Did the wire-protocol server gain a
  capability? Is there a fixture test?
  Did shipped clients opt into it (if
  expected to)?
- Did distribution shims or plugin
  manifests gain logic that should live
  in the binary? Flag.
- Did a generated marker syntax change
  in a way that breaks parsing of older
  markers? Blocker.

## Step 4: severity rubric

- **blocker** — violates dependency
  direction, breaks a published
  contract, or makes a future refactor
  materially harder. Schedule a fix
  before the next release. File a plan
  with a back-link to the audit entry.
- **tax** — works today but adds
  friction (a function in the wrong
  package, a one-off setting). Schedule
  inside the next plan cycle; leave the
  entry in the audit log until picked
  up.
- **nice-to-have** — cosmetic (naming,
  small duplication, doc tightening).
  Batch into a cleanup commit when
  convenient.

## Step 5: record findings

Append to the audit log under a new
`## Audit YYYY-MM-DD (range: …)` section.
For each severity, list bullets:

```markdown
### blockers

- `path/to/file`: imports the wrong
  layer. Violates DIP — orchestrator
  must not depend on variant packages.
  Fix by routing through the port
  interface.
```

Update the `audit-from:` field to the
current `origin/main` SHA. Then run the
project's lint/fix from the workspace
root to refresh any catalogs that list
the audit log (e.g. project index files,
contributor catalogs).

## Step 6: schedule

Group blockers by the structural fix
they need. One plan covers one fix,
even if it resolves several audit
entries; the plan lists each entry it
closes out.

For each group:

- Create a new plan file in the
  project's plan directory using the
  next available numeric prefix.
- Title format: `arch-fix-<short-slug>`.
- Reference every audit entry the plan
  closes in the plan's "Context" section.
- Set the plan `status` to the project's
  "not started" sentinel.
- If the plan would exceed the project's
  lint-enforced max file length, split
  it along natural seams (per-package,
  per-surface) and link the splits to
  each other in their "Context" sections.

For each tax and nice-to-have:

- Leave in the audit log. Mention in the
  next routine cleanup plan.

## Step 7: tell the user

Summarize: number of blockers, number of
tax, number of nice-to-have. List the
new plan filenames. Do not modify
offending code; the audit run is
complete.

## Common skip cases

- Test fixtures (typically under
  `testdata/` or alongside variant
  packages). Their architecture is by
  design.
- Generated content between directive
  markers — review the directive
  parameters, not the body.
- Comments-only changes do not require
  an audit entry unless they document a
  contract that has changed.
- Vendored or generated code (`*_gen.go`,
  `dist/`, etc.) is excluded.
