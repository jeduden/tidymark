---
id: 86
title: 'GitHub Actions merge queue'
status: "🔲"
summary: >-
  Bors-style merge queue using only GitHub Actions:
  label state machine, batch merging, binary bisection
  on failure, and escalation triggers.
---
# GitHub Actions merge queue

## Goal

Build a merge queue that serializes PR merges to
`main` using only GitHub Actions. No external server.
No GitHub native merge queue. Supports batch testing
and binary bisection on CI failure, modeled after Gas
Town's Refinery.

## Background

GitHub's native merge queue needs a Team or Enterprise
plan for private repos. It offers limited control over
batching and failure handling. Gas Town's Refinery
implements Bors-style queuing (batch, test tip, bisect
on failure) but is coupled to the Gas Town daemon.

This plan builds the same algorithm as a standalone
GitHub Actions workflow. Labels drive the state
machine. Branches serve as the batch mechanism.

## Design overview

### State machine

Labels on PRs encode queue state:

| Label           | Meaning                       |
|-----------------|-------------------------------|
| `queue`         | Author requests merge         |
| `queue:pending` | Waiting in line               |
| `queue:active`  | Currently in a batch under CI |
| `queue:failed`  | Batch CI failed, ejected      |

A single concurrency group (`merge-queue`) ensures
only one queue run executes at a time.

### Branch naming

Batch branches live under `merge-queue/`:

- `merge-queue/batch-<run_id>` — combined PRs
- `merge-queue/bisect-<run_id>-<half>` — bisection

All batch branches are ephemeral. They are deleted
after merge or failure.

### Workflow files

#### ci-reusable.yml

Extract the existing CI jobs (`lint`, `test`,
`mdsmith`, `demo`) into a reusable workflow that
accepts a `ref` input. Each job checks out
`inputs.ref` instead of the default ref.

```yaml
on:
  workflow_call:
    inputs:
      ref:
        required: true
        type: string
```

#### ci.yml (updated)

Thin wrapper calling `ci-reusable.yml` with the
default SHA. Keeps `push`, `pull_request`, and
`merge_group` triggers unchanged.

#### merge-queue.yml

Primary orchestrator. Triggers on `pull_request:
[labeled]` and `workflow_dispatch` (for self-
re-trigger and bisection inputs).

```yaml
concurrency:
  group: merge-queue
  cancel-in-progress: false
```

Jobs:

1. **prepare** — Collect queued PRs from labels or
   dispatch input. Output JSON PR list.
2. **batch** — Create batch branch, merge PRs.
   Eject conflicts.
3. **verify** — Call `ci-reusable.yml` against the
   batch branch.
4. **merge-or-bisect** — Fast-forward main on pass.
   Trigger bisection dispatch on fail.
5. **notify** — Comment on each PR with result.
6. **cleanup** — Delete branches. Re-trigger if
   more PRs are queued.

### Authentication

The workflow needs `contents: write` and
`pull-requests: write`. Use `GITHUB_TOKEN` if
branch protection allows it. Otherwise store a
fine-grained PAT as a repo secret.

## Phases

All three phases use the same `merge-queue.yml`.
They differ in batch size and the fail path in the
merge-or-bisect job.

### Phase 1 — Serial queue (MVP)

Batch size fixed to 1. No bisection needed.

**prepare:** `gh pr list --label queue` sorted by
number. Pick the oldest PR. Relabel it
`queue:active`. Output PR number and head branch.

**batch:** Create `merge-queue/batch-<run_id>` from
main. Run `git merge --no-ff origin/<pr-branch>`.
On conflict: label `queue:failed`, comment, exit.

**verify:** Call `ci-reusable.yml` with the batch
branch ref.

**merge-or-bisect:** Pass → fast-forward main,
close PR. Fail → label `queue:failed`, comment
with CI run link.

**cleanup:** Delete batch branch. If more PRs
carry the `queue` label, fire `gh workflow run`
to self-trigger.

### Phase 2 — Batch merging

**prepare:** Take up to N PRs (default 5) instead
of 1. Relabel all `queue:active`. Output JSON
array of PR numbers and branches.

**batch:** Merge each PR sequentially. If one
conflicts, eject it alone and continue with the
rest. Output the final merged-PR list.

**merge-or-bisect:** Pass → fast-forward main,
close all. Fail with batch > 1 → proceed to
phase 3.

### Phase 3 — Binary bisection

**merge-or-bisect (fail, batch > 1):** Split PR
list in half. Fire `gh workflow run` with
`-f batch_prs='[1,2]' -f bisect=true`.

The new run's **prepare** skips label scan. It
uses the `batch_prs` input directly. Each
recursion is a separate dispatch serialized by
the concurrency group:

- Left passes → merge those PRs to main.
  Dispatch right half with `bisect=true`.
- Left fails, size > 1 → dispatch left half
  split again.
- Left fails, size = 1 → that PR is the culprit.
  Label `queue:failed`. Relabel right half as
  `queue:pending`.

Worst case: `ceil(log2(N)) + 1` CI runs. A batch
of 8 needs at most 4 runs to isolate the failure.

## Escalation: when you need a Bors server

A GitHub Actions queue has inherent limits. Migrate
to a dedicated service (Bors-NG, Mergify, Kodiak,
or Gas Town Refinery) when any trigger fires.

### Throughput triggers

- **Queue depth > 10 PRs regularly.** One batch at
  a time. CI at 5 min means 30+ min to drain 10 PRs
  with bisection.
- **CI > 15 minutes.** Long CI multiplied by bisect
  rounds creates unacceptable latency. A server can
  run parallel speculative CI.
- **More than 20 PRs merged per day.** The label
  state machine becomes a bottleneck.

### Feature triggers

- **Priority queues.** Labels have no ordering. A
  server can implement priority lanes.
- **Cross-repo coordination.** Merging repo A
  depends on repo B. Actions queues are per-repo.
- **Dependent PR chains.** Stacked PRs need
  dependency tracking labels cannot express.
- **Rollback automation.** Auto-revert on post-merge
  failure requires persistent state.

### Reliability triggers

- **Label race conditions.** Concurrent label events
  can cause duplicate merges. A server has
  transactional state.
- **Actions outages.** A self-hosted server with its
  own runners is more resilient.
- **Audit trail needed.** Actions logs are ephemeral.
  A server provides persistent history and metrics.

### Cost triggers

- **Minutes budget exceeded.** Bisection multiplies
  CI runs. A batch of 8 that fails costs up to 4x
  the normal CI bill.

### Decision matrix

| Scenario          | Actions queue | Bors server |
|-------------------|---------------|-------------|
| Solo / small team | sufficient    | overkill    |
| < 10 PRs/day      | sufficient    | optional    |
| 10-20 PRs/day     | monitor       | recommended |
| > 20 PRs/day      | too slow      | required    |
| CI < 5 min        | sufficient    | optional    |
| CI 5-15 min       | monitor       | recommended |
| CI > 15 min       | bisect lag    | required    |
| Priority merges   | no support    | required    |
| Cross-repo deps   | no support    | required    |
| Stacked PRs       | no support    | required    |

## Testing

Extract bisect splitting and PR sorting into
`internal/mergequeue/` as pure Go functions on
JSON arrays. These are unit-testable with
`go test` — no GitHub API needed.

Add a `dry_run` boolean input to the workflow.
In dry-run mode every mutating step (push, merge,
label, comment) logs intent but skips execution.

End-to-end scenarios (on this repo or a test repo
with fast CI):

1. Single green PR → merged to main
2. Single red PR → `queue:failed` + comment
3. Merge conflict → ejected before CI
4. Batch of 3 green → all merged in one run
5. Batch of 3 (1 red) → bisect isolates failure
6. Serial re-trigger → second PR after first

## Tasks

1. Create labels (`queue`, `queue:pending`,
   `queue:active`, `queue:failed`) via `gh label`
2. Extract `ci-reusable.yml` from `ci.yml` with
   `ref` input parameter
3. Update `ci.yml` to call `ci-reusable.yml`
4. Extract bisect/prepare logic into
   `internal/mergequeue/` with unit tests
5. Implement phase 1 serial queue workflow with
   `dry_run` input
6. E2E test phase 1: green PR merges, red PR
   gets `queue:failed`, conflict ejected
7. Implement phase 2 batch merging
8. Implement phase 3 binary bisection via
   `workflow_dispatch` self-invocation
9. E2E test phase 3: batch of 4 (1 red), verify
   bisection isolates the failure
10. Add `/queue` comment trigger via
    `issue_comment` event
11. Document usage in `docs/development/`

## Acceptance Criteria

- [ ] Serial queue: PR labeled `queue` merges to
      `main` after CI passes on batch branch
- [ ] Failed CI: PR ejected with `queue:failed`
      label and explanatory comment
- [ ] Batch merge: up to N queued PRs merged in
      one CI run when all pass
- [ ] Bisection: batch failure triggers binary
      bisect; failing PR isolated, passing PRs merge
- [ ] Merge conflicts: PR that cannot merge into
      batch branch is ejected immediately
- [ ] Re-trigger: workflow checks for new queued
      PRs after each run and self-triggers
- [ ] Dry-run mode: workflow logs intent without
      mutating when `dry_run` is true
- [ ] Bisect/prepare logic has Go unit tests in
      `internal/mergequeue/`
- [ ] Existing CI unchanged: `push`/`pull_request`
      triggers work as before
- [ ] Escalation triggers documented
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports no issues
