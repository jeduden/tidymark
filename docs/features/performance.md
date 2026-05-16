---
title: "Fast on every run"
summary: >-
  A single static Go binary, no runtime to start. The workspace walk
  runs in parallel, embeds are linted once, and `check` is built for
  the hot path — a full check of this repo completes in under 300 ms.
icon: zap
link: "/docs/reference/cli/check/"
weight: 7
---
# Fast on every run

Speed is a feature, not an afterthought.

**No runtime cold-start.** mdsmith ships as one static Go binary.
There is no Node, Python, or JVM process to launch before work
begins. Process start time is measured in single-digit milliseconds.

**Parallel workspace walk.** Files fan out across all available
cores. Files pulled in by `<?include?>` and `<?catalog?>` are
linted once regardless of how many host documents reference them —
a large doc set does not re-scan the same prose repeatedly.

**Hot-path design.** `mdsmith check` shares the rule engine with
the LSP server and the fixer. Editor feedback, `git commit` hooks,
and CI all run the same optimised core — no per-surface overhead.

A full check of this repository — 70-plus Markdown files, the full
rule suite, cross-file link resolution — completes in under 300 ms
on commodity hardware. Sub-second on the first run and every run
after it.

See the [`check`](../reference/cli/check.md) reference for flags
and exit codes.
