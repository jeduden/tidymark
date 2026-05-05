---
summary: >-
  Concrete use cases for query aggregations and the toughest open
  question for mdsmith — whether the index that makes them fast in
  mdbase has a stateless equivalent. Walks the workload shapes, the
  SQLite payoff, and the fzf / ripgrep-style alternative.
status: 🔳
---
# Aggregation use cases and the index question

mdbase's strongest argument for a SQLite cache is
aggregation: grouping, counting, summing, joining
across typed records at vault scale. mdsmith does
not have aggregation today and does not have a
persistent index. This document walks the use
cases, looks at when the SQLite payoff is real
versus marginal, and explores whether mdsmith
could serve the same workloads stateless — the
way `fzf` and `ripgrep` do for their domains.

The point is to have a concrete picture of the
workload before committing to (or against) an
index. Triggers in
[learn-from-mdbase.md](learn-from-mdbase.md)
mention this doc; this is where they get
detailed.

## Aggregation workload shapes

Aggregation is not one thing. Five shapes recur
across teams that adopt mdbase or want to:

| Shape       | Example                                           | Cost driver              |
|-------------|---------------------------------------------------|--------------------------|
| count       | "How many open tasks per assignee?"               | scan all matching FM     |
| sum / avg   | "Total estimated hours per project."              | scan + arithmetic        |
| top-N       | "Five oldest unresolved bugs."                    | scan + sort              |
| join        | "List tasks whose assignee.role = 'reviewer'."    | scan + cross-file lookup |
| time bucket | "Plans completed per week over the last quarter." | scan + date math + group |

mdbase serves all five with a SQL query against
its index. mdsmith would have to scan files per
query today; with an index, it could serve them
the same way. With a stateless-fast approach (see
section 5), four out of five are reachable
without state.

## Worked use cases

### A-1: Sprint dashboard

**Setting.** Engineering team tracks tasks as
Markdown files under `tasks/`. ~800 active tasks,
~5,000 historical. CI runs a "sprint dashboard"
job after every push that produces a status
summary as JSON, fed into a Slack notification.

**Query.** The query shape:

```yaml
types: [task]
where: status == "in-progress"
group_by: assignee
aggregate:
  - count
  - field: priority
    op: avg
order_by: count desc
```

**What it produces.** A two-column table:
assignee, in-progress task count, average
priority. ~10 rows.

**Cost driver.** 5,800 files scanned, 800
matching, 800 grouped. Per-file work is parsing
the FM (~1ms cold). 5.8 seconds cold, 50ms warm.

**With mdbase.** SQLite index makes this a single
`SELECT assignee, COUNT(*), AVG(priority) FROM tasks
WHERE status='in-progress' GROUP BY assignee`.
Sub-millisecond.

**With mdsmith stateless.** Parallel re-read
through every `.md` file in `tasks/`, filter by
FM, count and average in-process. With ripgrep-
class IO and FM parsing in Go, ~600ms cold on
modern hardware. JSON output piped to the next
step.

**Verdict.** Stateless is fine for a CI job
running once per push. The 600ms is hidden in
the test pipeline.

### A-2: Reviewer load report

**Setting.** Documentation team. ~3,000 RFCs.
Weekly report: "Show all RFCs whose author or
reviewer is X, grouped by status, with the
five-most-recent in each group."

**Query.** The query shape:

```yaml
types: [rfc]
where: 'author == "alice" || "alice" in reviewers'
group_by: status
aggregate:
  - count
  - field: created
    op: max
top_per_group: 5
order_by: created desc
```

**Cost driver.** 3,000 files scanned, 200
matching, 200 grouped, top-5 per group. Per-file
work the same ~1ms.

**With mdbase.** Index hit on the `author`
column, secondary scan of the `reviewers` array,
GROUP BY in SQL. Sub-second even cold; the cache
warms across queries.

**With mdsmith stateless.** Same parallel re-read
shape. ~300ms cold for the filter alone; sort and
top-5 per group in process. Total ~400ms.

**Verdict.** Stateless still works but the gap is
narrowing. At 30,000 RFCs the per-query cost
(~3 seconds) starts to feel slow for an interactive
dashboard.

### A-3: Knowledge-graph backlinks at scale

**Setting.** Research lab vault. 12,000 notes
with dense `[[wikilinks]]`. Researcher opens a
paper note and wants the backlinks panel:
"What cites this paper?" In milliseconds.

**Query.** The query shape:

```yaml
backlinks_to: notes/chen-2024.md
group_by: tag
order_by: count desc
limit: 50
```

**Cost driver.** 12,000 files; the link graph is
the load-bearing structure. Each query needs an
edge lookup `(target → sources)` and then a group
by tag.

**With mdbase.** Indexed `(target_path, source_path)`
edge table. The lookup is a single B-tree probe;
the group-by uses the cached tag column.
Sub-millisecond.

**With mdsmith stateless.** Re-read 12,000 files
to rebuild the link graph per query. Even with
parallel IO and fast parsing, ~2 seconds cold.
Unacceptable for an interactive backlinks panel.

**Verdict.** Stateless does not work at this
scale for interactive use. This is the case
where the SQLite-class index pays.

### A-4: Time-bucketed velocity

**Setting.** Plan tracker. ~500 plans across
two years. Quarterly review: "Plans completed
per month, last 12 months."

**Query.** The query shape:

```yaml
types: [plan]
where: status == "✅" && completed_at >= today() - "365d"
group_by: time_bucket(completed_at, "1M")
aggregate: count
order_by: time_bucket asc
```

**Cost driver.** 500 files; light. The time-
bucket logic is the interesting part.

**With mdbase.** Date functions in expressions
plus GROUP BY on the bucket key. Trivial in SQL.

**With mdsmith stateless.** Parallel parse, in-
process bucket, count. ~50ms even cold.

**Verdict.** Stateless wins. The corpus is
small, the aggregation is light. Adding a cache
would only slow things down.

### A-5: Cross-type join

**Setting.** Mixed product wiki. RFC type lists
its `owner: alice`. Task type also has
`assignee: alice`. Query: "List Alice's open RFCs
with at least one in-progress related task."

**Query.** The query shape:

```yaml
types: [rfc]
where: |
  owner == "alice" &&
  status == "open" &&
  any(related_tasks, t -> t.asFile().status == "in-progress")
```

**Cost driver.** ~200 RFCs, ~5,000 tasks. Each
matching RFC triggers per-task lookups on
`related_tasks`, which is a list of paths.

**With mdbase.** The `asFile()` call goes through
the index: link target → cached FM → status
column. ~10 lookups per RFC, all O(log n).
Total: 200 × 10 = 2,000 index hits. Fast.

**With mdsmith stateless.** Re-read all RFCs
(filter), then re-read each related task (200 ×
~3 hops × ~1ms parse = 600ms additional). Total
~1 second cold. Acceptable but not great.

**Verdict.** Marginal. At this scale, stateless
is on the edge. Once joins go more than two
hops or the corpus doubles, the index pays.

### A-6: Real-time editor decoration

**Setting.** Researcher in VS Code wants
inline decoration: every wikilink shows the
target's title in light text after the link.
Updates as files change.

**Query.** Per visible link, fetch
`asFile().title`. Hundreds of times per file
open, called from the LSP server.

**Cost driver.** Latency. 50ms feels slow;
200ms is unacceptable.

**With mdbase.** Index-backed lookup.
Sub-millisecond per resolution.

**With mdsmith stateless.** Per-resolution disk
read. Even with read coalescing, 5–10ms per
target file. With 100 visible links: 500ms-1s of
work per editor open. Bad.

**Verdict.** This is an LSP case. The right
shape is option 2 from
[learn-from-mdbase.md §P-1](learn-from-mdbase.md):
in-memory, process-scoped index in the LSP
server. No disk cache; the editor session keeps
state, the CLI does not.

## What an index actually costs

Before deciding whether the index pays, the
estimate above ("seconds cold without cache,
sub-millisecond with cache") is too coarse. The
reality has three confounders that narrow the
band where a persistent cache wins.

### Sync-check overhead is not free

A persistent index has its own startup cost
before any query runs. At minimum it must
verify the cache is current. Two layers of check
exist (mdbase spec §13):

1. **mtime sweep.** `stat()` every file in the
   collection, compare against the cached
   mtime. A `stat()` is a few microseconds when
   the dirent is in the OS cache, more on a
   cold filesystem. 5,000 files × ~10μs ≈ 50ms.
   50,000 files ≈ 500ms.
2. **Hash on doubt.** When mtime is unreliable
   (network filesystems, build tools that
   `touch`, file-system clock skew across
   machines), the impl falls back to hashing
   file contents. SHA-256 of a 4KB file is
   ~30μs of CPU, plus the read. 5,000 files ≈
   150ms of hashing alone, more if the OS
   cache is cold.

Combined: a clean cache validation pass on a
50,000-file corpus is on the order of a second.
**That is in the same range as a fresh parallel
parse of the same corpus** (see the sketch
under "What a real benchmark plan would
measure" below). The cache wins only when the
saved work — parse time, link-graph build,
aggregation — clearly exceeds the validation
cost.

For a query like A-1 (sprint dashboard, 5,800
files, simple FM filter), the saved work is ~600ms
of parsing and a `WHERE` clause. The validation
cost is ~50ms. The cache wins, but by less than
the naive analysis suggests.

For a query like A-3 (backlinks panel, 12,000
files, link-graph lookup), the saved work is
~2 seconds of full-body parse plus link-edge
construction. Validation is ~120ms. Here the
cache pays clearly.

### OS file cache is the implicit warm cache

Linux's page cache, macOS's UBC, Windows' system
cache — every modern OS keeps recently-read
files in RAM. The first `mdsmith check` reads
files from disk; the second reads them from
memory at memcpy speed.

For a typical project this matters more than
any application-level cache:

- A 50MB vault fits entirely in OS cache on any
  development machine. Once read, repeated reads
  are RAM-speed (≈10GB/s) for as long as the
  pages stay resident.
- The pages stay resident until evicted (LRU
  pressure, typically minutes-to-hours of idle)
  or the file is modified (the dirty page is
  flushed and re-read on next access).
- `stat()` results are cached the same way. A
  warm `stat()` is ~1μs.

The implication: **mdsmith's "stateless re-read"
is, in practice, a re-read from RAM**, not from
disk, for any corpus that fits in available
memory and any session that runs more than once.
A persistent application cache duplicates work
the OS already does for free. Where it wins is
*parsing* work — the conversion from raw bytes
to a structured index — which the OS does not
cache.

So the question shifts: at what corpus size, and
for what workload, does the parsing cost dominate
to the point where caching the parse result pays
for the cache's own validation overhead?

### Background indexing with priority queries

A third option sits between "rebuild on every
run" and "fully persisted index": **build the
index in memory at startup, in the background,
while queries run against the partial state and
get hoisted in priority.**

The pattern is well-trodden in IDEs.
IntelliJ and the TypeScript language server
both work this way: the editor opens
immediately, indexing starts in the background,
and any feature that needs not-yet-indexed data
either jumps the queue (re-prioritizing the
relevant subdirectory) or falls back to a
slower path until the index catches up.

For mdsmith the shape would be:

- **Cold start.** Spawn a background indexer
  goroutine that walks the workspace and parses
  files in priority order: `kind:` matches first,
  then files referenced by an open query, then
  the rest.
- **Query arrives.** The query thread checks
  the in-progress index. Files it needs but
  doesn't find get pushed to the front of the
  indexer's queue. Once parsed, the result is
  returned.
- **Tail.** The indexer keeps building until
  the workspace is fully indexed or the process
  exits.

This works **only when the process is
long-lived** — an LSP session, an
`mdsmith watch` daemon (P-2), or an interactive
REPL. For a one-shot CLI command (`mdsmith query
... && exit`) the process exits before the
background work finishes, so the value is
small. The pattern is therefore not a
replacement for stateless one-shot, but an
extension of the in-memory in-LSP option from
the P-1 alternatives table.

What it buys:

- **Apparent cold-start latency drops.** The
  user sees the first query result fast even on
  a 50,000-file workspace, because the indexer
  does not need to be complete to answer
  point queries.
- **No persistent state.** The whole index lives
  in process memory. Restart and rebuild; no
  staleness, no schema migration, no `.mdbase/`
  on disk.
- **Iteratively useful.** A workspace that's
  10% indexed can already answer 90% of "find
  this kind" queries when that kind is what was
  prioritized.

What it costs:

- Indexer state machine and priority queue
  (modest implementation work, well-understood
  pattern).
- Memory bounded by workspace size; mdsmith's
  existing 512MB GOMEMLIMIT covers most cases.
- More complex than "rebuild fresh per query",
  less complex than a persistent on-disk store.

### What this means for the trigger

The trigger for adding a persistent on-disk
cache (P-1 in
[learn-from-mdbase.md](learn-from-mdbase.md))
is therefore narrower than "cold-start cost
dominates". It needs to survive three filters
in sequence:

1. **OS-cache filter.** Repeated runs on a
   stable corpus are already fast through the
   page cache. A persistent cache helps only
   when the *parsed* index — not the raw bytes
   — is what's expensive to rebuild. Confirm by
   profiling the second invocation, not the
   first.
2. **Sync-check filter.** The savings from
   skipping parse work must clearly exceed the
   cost of validating the cache against the
   filesystem. This is the gap between "naive
   re-parse takes 1s" and "cache validation
   takes 0.5s".
3. **In-memory-with-priority filter.** A
   long-lived process can already get most of
   the latency benefit with background indexing
   in RAM, without the operational cost of an
   on-disk cache. Confirm the workload genuinely
   needs cross-process freshness or restart-time
   warmth.

Surviving all three is a higher bar than "cold
start is slow on a big repo". The wiggle room
is real and worth using before reaching for
persistent state.

### What a real benchmark plan would measure

For the choice to be informed rather than
guessed, the next step is concrete numbers on a
real corpus. A benchmark plan would compare
five configurations on synthetic 1k / 10k / 50k
file workspaces:

| Config                                | Cold start | 2nd run | Notes                             |
|---------------------------------------|------------|---------|-----------------------------------|
| stateless re-read, OS cache cold      | T_cold     | —       | upper bound; rare in practice     |
| stateless re-read, OS cache warm      | T_warm     | T_warm  | the realistic baseline            |
| in-memory index, lazy build, in-LSP   | T_warm     | <1ms    | for long-lived processes          |
| in-memory index with priority queries | <T_warm    | <1ms    | ditto, with apparent latency win  |
| persistent on-disk cache, mtime sync  | T_sync     | <1ms    | T_sync ≈ stat + index lookup      |
| persistent on-disk cache, hash sync   | T_hash     | <1ms    | T_hash > T_sync, paranoid setting |

The decision lands when `T_warm` for the
relevant workload exceeds an interactive
threshold (typically 100ms), the in-memory
options fall short for the access pattern, and
`T_sync` is comfortably below `T_warm`. Until
then, none of the persistent options earn their
operational baggage.

## When the index pays

Reading across the six cases, three signals
predict when an index earns its cost:

1. **Interactive latency.** A query that runs
   inside a UI (LSP decoration, backlinks panel,
   editor hover) needs <100ms response. Stateless
   re-read gets there only at small file counts.
2. **Repeated queries on a stable corpus.** A CI
   job runs once per push and caches don't help.
   A researcher running queries all day in a
   stable vault sees the cache pay every minute.
3. **Cross-file joins beyond one hop.** Single-
   file queries scale linearly with the corpus.
   Joins multiply the cost; an index turns each
   hop from O(n) to O(log n).

The cross-product:

| Use case             | Latency need | Repeated? | Joins?  | Index pays? |
|----------------------|--------------|-----------|---------|-------------|
| A-1 sprint dashboard | CI (seconds) | no        | no      | no          |
| A-2 reviewer load    | weekly job   | no        | one hop | borderline  |
| A-3 backlinks panel  | UI (<100ms)  | yes       | one hop | yes         |
| A-4 velocity         | report (~s)  | no        | no      | no          |
| A-5 cross-type join  | report (~s)  | no        | 2+ hops | borderline  |
| A-6 editor decor     | UI (<50ms)   | yes       | one hop | yes         |

Two of six clearly need an index; two clearly
don't; two sit on the edge. The interesting
design question for mdsmith is whether the two
clear cases (A-3 backlinks panel, A-6 editor
decoration) can be served by an in-memory index
inside the LSP server without ever shipping a
persistent on-disk cache.

## Stateless-fast: the `fzf` / `ripgrep` model

Both `fzf` and `ripgrep` are well known for being
fast without persistent state. They work because
of three properties:

1. **Aggressive parallelism.** Both tools fan out
   across all available cores. ripgrep walks the
   filesystem with concurrent producers and
   consumers; fzf processes input lines in
   parallel.
2. **Tight inner loops.** ripgrep's regex engine
   uses Aho-Corasick and Teddy SIMD matching;
   fzf's scoring is hand-tuned. Per-line cost
   is in the tens of nanoseconds.
3. **No fsync, no parsing of structure.**
   ripgrep matches bytes, not ASTs. fzf doesn't
   know what it's matching. The work that's
   skipped is the work that would need an index.

For an FM-aware tool like mdsmith, the analogy
holds for parts of the workload:

**What mdsmith could ripgrep-class.** Workloads where the stateless model holds:

- **Body full-text search (Q-3).** Already
  ripgrep's domain; mdsmith could shell out, or
  embed `regexp/syntax` and walk files
  in parallel.
- **FM filtering (Q-1, Q-2, A-1, A-4).** Reading
  YAML between `---` delimiters and matching a
  small struct against a CUE struct is cheap if
  parallelized. Estimate: 10,000 files in
  ~500ms cold on a modern laptop.
- **Aggregations on FM only (A-1, A-4).** Group
  and count after the parallel parse. The
  aggregation itself is microseconds; the parse
  dominates.

**What ripgrep-class hits limits on.** Workloads
where statelessness breaks down:

- **Backlinks at scale (A-3).** Building the link
  graph means parsing every link in every body,
  not just FM. ~10× more parse work. At 10,000
  files this is 5+ seconds cold — too slow for an
  interactive panel.
- **Multi-hop joins (A-5).** Each hop multiplies
  the work. Two hops over 10,000 files is
  50+ seconds without a cache.
- **Editor-LSP decoration (A-6).** Per-keystroke
  parsing is hopeless at any corpus size. The
  LSP server must keep state.

**The pattern.** Stateless-fast handles workloads
where each query touches each file exactly once
and the per-file work is small (<1ms). It breaks
down where the same data is touched repeatedly
(LSP decoration), where the access pattern is
graph-shaped (joins, backlinks at scale), or
where work per file is heavy (full-body parse).

## A pragmatic path for mdsmith

These design choices are open. The shape that
falls out of the workload analysis is:

1. **Make the stateless path as fast as possible
   first.** Parallel walk via the existing
   `internal/discovery` package, FM parse with
   `gopkg.in/yaml.v3`, in-process filtering and
   aggregation. Target: 10,000 files / 500ms
   cold. This lifts A-1, A-2, A-4 without any
   new state.
2. **Add an in-memory link graph in the LSP
   server.** Plan 131 already builds toward
   this. Extending the LSP's process-scoped
   state to cover backlinks (L-4) and decoration
   (A-6) costs nothing in the CLI workflow and
   handles the two cases that need <100ms
   response.
3. **Defer persistent on-disk index until a real
   trigger fires.** Real profiling on a real
   corpus showing CLI cold-start dominates. Or
   a feature like Q-5 aggregations whose joins
   make stateless infeasible at the user's
   scale. When it does fire, the choice between
   BoltDB-style and SQLite is a measurable one,
   not a guess.

The summary slogan, if one helps: **stateless by
default; in-memory in the LSP; persistent only
when the workload proves it.**

This is not a commitment. Triggers in
[learn-from-mdbase.md](learn-from-mdbase.md)
might fire in any order, and a real user case can
override any of these choices. The point of
walking the workloads here is to make sure the
choice is informed when it lands.

## Open questions

A real plan to act on any of this would have to
settle:

- **What is the actual cold-start cost today?**
  Benchmarks on a 1k / 10k / 50k file synthetic
  corpus would replace the estimates above.
- **Does parallel FM-only parse hit IO or CPU
  limits first?** Determines whether mmap or
  fanout-readers is the right shape.
- **Can the LSP in-memory graph subsystem be
  cleanly factored so that, if a CLI cache is
  ever added, both share one builder?** Reuse
  matters more than the storage choice.
- **What's the corpus-size threshold where each
  workload shape (A-1..A-6) tips into needing
  the index?** Should be measured, not guessed.
- **Is there a half-step: an "ephemeral"
  in-memory cache mode for one CLI invocation
  that runs many queries (e.g., a script
  block)?** Cheap to add, useful in agent loops.

## Sources

- [`fzf` README on architecture and SIMD scoring][fzf]
- [`ripgrep` "How fast is it" notes][rg]
- mdbase spec §10 (querying), §13 (caching), and
  appendix A.2 (task-management worked example)
- mdsmith codebase 2026-05: `internal/discovery`,
  `internal/lint/file.go`, `internal/query/query.go`

[fzf]: https://github.com/junegunn/fzf
[rg]: https://github.com/BurntSushi/ripgrep/blob/master/FAQ.md
