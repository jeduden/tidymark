---
summary: Systematic enumeration of mdbase capabilities mdsmith does not yet have, with a per-gap mini-plan covering goal, design sketch, surface, effort, and open questions.
status: 🔳
---
# What mdsmith can learn from mdbase

This document is a research exercise, not a roadmap.
It enumerates every mdbase capability that mdsmith
either lacks or expresses differently, triages each
into in-scope / out-of-scope / already-in-flight,
and sketches a mini-plan for the in-scope ones. The
goal is to give a future maintainer a list of
discrete, well-framed work items they can promote
into actual `plan/<n>_*.md` files when the time
comes.

## Method

1. Walk every section of [features.md](features.md).
2. For each row where mdbase has a capability and
   mdsmith does not, note it as a gap with a short
   identifier (e.g. `S-1`, `Q-3`).
3. Triage:

  - **In-scope** — fits the existing mdsmith
     mission (lint, fix, generate over Markdown
     files in a repo) and would benefit a real use
     case from [use-cases.md](use-cases.md).
  - **Out-of-scope** — would change what mdsmith
     is. Note why and the principled alternative.
  - **In-flight** — already covered by an open
     plan or PR.

4. For each in-scope gap, write a mini-plan with:

  - **Goal** — one sentence
  - **Sketch** — proposed mechanism, including
     concrete syntax where it helps
  - **Surface** — what new config / CLI / output
     the user sees
  - **Effort** — S / M / L (a day, a week, a
     month)
  - **Depends on** — gaps or existing plans this
     builds on
  - **Open questions** — things a real plan would
     have to settle

## Triage tables

Tables split by area for readability. Twelve
in-scope gaps follow as mini-plans below; the
out-of-scope and in-flight items get a brief
rationale at the end.

### Schema language (S)

| ID  | Gap                                          | Status       | Priority |
|-----|----------------------------------------------|--------------|----------|
| S-1 | Inline schema in `kinds:`                    | in-scope     | high     |
| S-2 | Named field-type taxonomy as CUE shortcuts   | in-scope     | medium   |
| S-3 | Schema inheritance (`extends:`)              | in-scope     | medium   |
| S-4 | Computed fields surfaced in catalog / query  | in-scope     | medium   |
| S-5 | Generated values on write (ULID, timestamps) | out-of-scope | n/a      |
| S-6 | Field deprecation flag                       | in-scope     | low      |

### File-to-kind matching (M)

| ID  | Gap                                      | Status   | Priority |
|-----|------------------------------------------|----------|----------|
| M-1 | Field-presence kind assignment           | in-scope | high     |
| M-2 | Field-value kind assignment (`where:`)   | in-scope | low      |
| M-3 | `path-pattern` (validates and generates) | in-scope | medium   |

### CRUD and rename (C)

| ID  | Gap                            | Status       | Priority |
|-----|--------------------------------|--------------|----------|
| C-1 | Create typed file              | out-of-scope | n/a      |
| C-2 | Update fields from CLI         | out-of-scope | n/a      |
| C-3 | Delete with broken-ref warning | out-of-scope | n/a      |
| C-4 | Atomic rename + ref rewrite    | in-scope     | high     |
| C-5 | Batch ops with `--where`       | out-of-scope | n/a      |
| C-6 | Dry-run mode for fix           | in-scope     | low      |

### Links (L)

| ID  | Gap                                   | Status          | Priority |
|-----|---------------------------------------|-----------------|----------|
| L-1 | Wikilink awareness                    | in-scope        | high     |
| L-2 | ID-field link resolution              | out-of-scope    | n/a      |
| L-3 | Auto-rewrite incoming links on rename | in-scope (=C-4) | high     |
| L-4 | Backlinks subcommand                  | in-scope        | high     |
| L-5 | Multi-hop link traversal in queries   | out-of-scope    | n/a      |

### Query (Q)

| ID  | Gap                                   | Status       | Priority |
|-----|---------------------------------------|--------------|----------|
| Q-1 | Sort (`--order-by`)                   | in-scope     | medium   |
| Q-2 | Pagination (`--limit` / `--offset`)   | in-scope     | low      |
| Q-3 | Body full-text search in query        | in-scope     | medium   |
| Q-4 | Date arithmetic with duration strings | in-scope     | low      |
| Q-5 | Aggregations (formulas, summaries)    | out-of-scope | n/a      |
| Q-6 | Cross-file traversal in queries       | out-of-scope | n/a      |
| Q-7 | Bases-compatible expression syntax    | in-scope     | low      |

### Cache, watch, migrations, conformance, LSP

| ID  | Gap                                    | Status          | Priority |
|-----|----------------------------------------|-----------------|----------|
| P-1 | SQLite-backed cache for large repos    | out-of-scope    | n/a      |
| P-2 | Watch mode beyond LSP per-session      | out-of-scope    | n/a      |
| V-1 | Migration manifests                    | out-of-scope    | n/a      |
| X-1 | Spec-first / multi-impl model          | out-of-scope    | n/a      |
| H-1 | LSP hover for rules / directives       | in-flight (122) | n/a      |
| H-2 | LSP symbol navigation + call hierarchy | in-flight (131) | n/a      |

## Schema language plans

### S-1: Inline schema in the `kinds` map

**Goal.** Let `kinds:` carry a schema directly,
without forcing every kind to point at a separate
`proto.md`.

**Sketch.** Extend the kind config to accept an
optional `schema:` block alongside (or instead of)
`required-structure.schema:`:

```yaml
kinds:
  task:
    rules:
      line-length:
        max: 100
    schema:
      frontmatter:
        id: '=~"^TASK-[0-9]{4}$"'
        status: '"open" | "in-progress" | "done"'
        priority: 'int & >=1 & <=5'
        "due?": "string"  # ISO date
      structure:
        - "# {title}"
        - "## Goal"
        - "## Acceptance"
      require:
        filename: "TASK-[0-9]+.md"
```

The schema engine receives a parsed AST whether
it came from inline YAML or from a referenced
`proto.md`. The current `proto.md` mechanism stays
as the path for schemas that include heading
templates with rich body content; inline schemas
cover the common case.

**Surface.** Three things change for the user:

- New `schema:` key in each kind
- `mdsmith kinds` command shows whether a kind's
  schema is inline or file-referenced
- Existing `required-structure.schema:` still works

**Effort.** M (1–2 weeks). Touches config types,
the schema loader, and `mdsmith kinds` output.
Mostly mechanical.

**Depends on.** Nothing.

**Open questions.** Should inline `structure:`
support the full directive set (`<?include?>` for
fragment composition)? Probably yes, for parity.

### S-2: Named field-type taxonomy as CUE shortcuts

**Goal.** Make CUE schemas more ergonomic by
shipping reusable definitions for the common types
mdbase names directly (date, datetime, enum, link).

**Sketch.** Add a small CUE library
`internal/cue/types.cue` that defines:

```cue
package mdsmith

#date:     =~"^\\d{4}-\\d{2}-\\d{2}$"
#datetime: =~"^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}"
#email:    =~"^[^@]+@[^@]+$"
#url:      =~"^https?://"
#filename: =~"^[A-Za-z0-9_-]+\\.md$"
```

Schemas import the package and reference the
definitions:

```cue
package proto

import "github.com/jeduden/mdsmith/types"

created:  types.#date
modified: types.#datetime
```

Optionally expose a more compact YAML shorthand in
inline schemas (S-1):

```yaml
schema:
  frontmatter:
    created: date     # resolves to types.#date
    homepage: url
    contact: email
```

**Surface.** A new package import path; optional
shorthand in inline schemas.

**Effort.** S (a few days). Defining the patterns
is small; wiring the shorthand is the bulk.

**Depends on.** S-1 makes the YAML shorthand
useful; without it the patterns are still helpful
in `proto.md` schemas.

**Open questions.** What's the canonical list?
Start with the seven mdbase covers (string, int,
number, bool, date, datetime, time, enum, link)
and grow from real use.

### S-3: Schema inheritance via `extends`

**Goal.** Let one kind / schema extend another so
common front-matter fields live in a base schema.

**Sketch.** Two parts:

- **In `kinds:`** — `extends:` lists parent kinds
  whose rule overrides apply before this one (today
  this is implicit deep-merge by declaration order;
  `extends:` makes it explicit).
- **In schemas** — CUE already supports
  unification, so `import` + struct embedding
  covers it. For inline schemas (S-1), extend the
  block:

  ```yaml
  schema:
    extends: [base]
    frontmatter:
      status: '"open" | "done"'
  ```

  The schema engine merges the parent's CUE struct
  with the child's, with child-wins on overlapping
  keys (mdbase semantics).

**Surface.** New `extends:` key in kind / inline
schema; documented in the file-kinds guide.

**Effort.** S–M. CUE already handles unification;
the lift is exposing the merge in the inline DSL.

**Depends on.** S-1.

**Open questions.** Single inheritance (mdbase) or
multiple? Multiple is more powerful but harder to
debug. Recommend single for parity, with `<?include?>`
covering composition.

### S-4: Computed fields in catalog / query

**Goal.** Let users define expressions that derive
a field from other front-matter fields, surface
the result in `<?catalog?>` columns and `mdsmith
query`, and never persist it back to disk.

**Sketch.** Computed fields live in the schema
(inline or proto.md) under a `computed:` block:

```yaml
schema:
  frontmatter:
    due: string       # ISO date
  computed:
    days_overdue:
      type: int
      expr: time.Parse("2006-01-02", due).Before(time.Now())
```

The expression language reuses CUE; an extra small
helper for date arithmetic (Q-4) makes typical
cases readable. Catalog references computed values
as `{computed.days_overdue}`. Query treats them as
read-only fields.

**Surface.** New `computed:` block in schemas;
catalog template extension.

**Effort.** M. CUE evaluation per file is
manageable; the catalog templating change is small.

**Depends on.** S-1 (for inline syntax), Q-4 (for
date arithmetic to be useful).

**Open questions.** Performance — naive evaluation
runs the expression per file per render. Cache
across a single `mdsmith fix` run. Beyond that,
out of scope.

### S-6: Field deprecation flag

**Goal.** Let a schema mark a field deprecated;
mdsmith warns when it appears in front matter.

**Sketch.** In the schema, `deprecated: true` on a
field. MDS020 emits a Warning (not an Error) when
the field is present.

```yaml
schema:
  frontmatter:
    legacy_owner:
      type: string
      deprecated: true
      message: "use 'owner' instead"
```

**Surface.** Schema field-level `deprecated:` and
`message:` keys; new MDS020 sub-diagnostic.

**Effort.** S.

**Depends on.** S-1.

**Open questions.** None worth blocking on.

## File-to-kind matching plans

### M-1: Field-presence kind assignment

**Goal.** Auto-assign a kind when specific
front-matter fields are present, removing the need
for an explicit `kind:` tag.

**Sketch.** Extend `kind-assignment:` with a
`fields-present:` selector:

```yaml
kind-assignment:
  - glob: ["docs/**"]
    kinds: [doc]
  - fields-present: [status, priority, assignee]
    kinds: [task]
```

Selectors combine with AND; multiple entries
combine with OR (matching the existing glob
semantics). A file matching no entry stays
untyped, like today.

**Surface.** New `fields-present:` key in
`kind-assignment:` entries; documented in the
file-kinds guide.

**Effort.** S.

**Depends on.** Nothing.

**Open questions.** Should presence-based matching
fire on null values (present but null)? Mirror
mdbase's "non-null required" semantics.

### M-2: Field-value kind assignment via `where`

**Goal.** Auto-assign a kind based on a CUE
expression over front matter, e.g. "files where
status is 'open'".

**Sketch.** Extend `kind-assignment:` with a
`where:` selector that takes a CUE struct
literal — same syntax as `mdsmith query`:

```yaml
kind-assignment:
  - where: 'kind: "plan", status: "in-progress"'
    kinds: [active-plan]
```

Useful for derived kinds: "active plans get a
stricter token budget than archived plans". A
config-time CUE compile per kind keeps the cost
low.

**Surface.** New `where:` key in
`kind-assignment:` entries.

**Effort.** S–M.

**Depends on.** Nothing.

**Open questions.** Performance at scale; if
`where:` runs per file per kind, n×m cost. A
trie-based prefilter on the keys mentioned in
`where:` mitigates. Defer until the feature is in
real use.

### M-3: `path-pattern` (validates and generates)

**Goal.** Let a kind declare a filename pattern
that validates existing files (does the path
match?) and templates new ones (where should a
new file live?).

**Sketch.** Add `path-pattern:` to a kind:

```yaml
kinds:
  plan:
    rules:
      required-structure:
        schema: plan/proto.md
    path-pattern: "plan/{id:int}_{slug}.md"
```

Validation: a file claiming `kind: plan` whose
path doesn't fit `plan/<int>_<slug>.md` produces
an MDS020 Error. Templating: `mdsmith query` could
emit candidate paths from this pattern (low
priority).

**Surface.** New `path-pattern:` key in kind
config; new MDS020 sub-diagnostic.

**Effort.** S.

**Depends on.** Nothing.

**Open questions.** Field substitution syntax —
braces with type tags as above, or simpler globs?
Match what mdbase does for portability.

## Link plans

### L-1: Wikilink awareness

**Goal.** Make MDS027 understand `[[Page]]`,
`[[Page|alias]]`, `[[Page#anchor]]` so wikilink-
heavy vaults can use mdsmith without flagging
every wikilink as text.

**Sketch.** Add a wikilink parser to the lint
pipeline (alongside the existing Markdown link
extractor in `internal/lint/`). Resolution rules
follow goldmark's link-resolution conventions plus
mdbase spec §8 (filename match in `.md` files;
sandboxed to repo root). Configurable via
`cross-file-reference-integrity.wikilinks:`:

```yaml
rules:
  cross-file-reference-integrity:
    wikilinks: true        # default false today
    wikilink-style: simple # | obsidian
```

Two styles because Obsidian and other tools
disagree on tiebreakers and aliases.

**Surface.** New rule settings; new diagnostic
flavors for broken wikilinks.

**Effort.** M. Parsing is small; resolution rules
need test coverage across Obsidian / Foam /
Logseq variants.

**Depends on.** Nothing.

**Open questions.** Should mdsmith canonicalize
wikilinks (rewrite to Markdown links on `fix`)?
Probably no — that's a refactor, not a lint, and
breaks Obsidian for users who want wikilinks.

### L-3 / C-4: Atomic rename + reference rewrite

**Goal.** A `mdsmith refactor rename <old> <new>`
subcommand that moves a file and rewrites every
incoming link.

**Sketch.** New subcommand:

```bash
mdsmith refactor rename docs/old.md docs/new.md
mdsmith refactor rename --heading "Old name" --to "New name" docs/file.md
```

Two modes:

- **File rename.** Moves `docs/old.md` to
  `docs/new.md`. Rewrites every `[text](docs/old.md)`,
  `[text](docs/old.md#anchor)`, and `[[old]]` in
  the workspace. Anchors normalized via the same
  GitHub-slug rules MDS027 uses.
- **Heading rename.** Updates the heading text in
  the source file and rewrites every cross-file
  anchor link pointing at the old slug.

`--dry-run` prints the proposed edits without
writing. Optimistic-concurrency check uses mtime
(file changed on disk during rename → abort).

**Surface.** New `mdsmith refactor` subcommand
group with `rename` and (later) other refactors.

**Effort.** L. Rename is the new write surface
mdsmith hasn't had before. Test matrix is large
(wikilinks, anchors, edge cases like links in
fenced code blocks).

**Depends on.** L-1 (wikilinks) for full coverage.

**Open questions.** Should rename also update
`<?include?>` and `<?build?>` directive paths?
Yes, for consistency. Should it touch front-matter
fields containing paths (`source: file.md`)?
Probably not without explicit opt-in — they aren't
syntactically links.

### L-4: Backlinks subcommand

**Goal.** A `mdsmith backlinks <file>` command
that lists every workspace file with a link to
the target.

**Sketch.** Reuses the link graph that MDS027
already builds during a check pass:

```bash
mdsmith backlinks docs/api.md
# docs/index.md:14: [API reference](api.md)
# docs/getting-started.md:42: [api docs](./api.md)
# plan/045_api-overhaul.md:8: [api](../docs/api.md)

mdsmith backlinks docs/api.md#authentication
# docs/index.md:18
# docs/security.md:33
```

JSON output for agents (`--format json`).

**Surface.** New `mdsmith backlinks` subcommand;
JSON output spec.

**Effort.** S. Most of the engine is in MDS027
already; surface it as a CLI command.

**Depends on.** L-1 to cover wikilinks (otherwise
the result misses Obsidian-style backlinks).

**Open questions.** Performance at 10k files —
the link graph builds fast but the output may be
unwieldy. `--limit N` plus filtering by glob
covers the common case.

## Query plans

### Q-1, Q-2: Sort and pagination

**Goal.** Support `--order-by FIELD` and
`--limit N` / `--offset N` on `mdsmith query`.

**Sketch.** New flags:

```bash
mdsmith query 'status: "open"' --order-by priority --order desc --limit 20
mdsmith query 'kind: "task"' --order-by due --offset 50 --limit 50
```

Sorts after CUE unification (i.e. on filtered
result set). Stable sort with deterministic
secondary key (path) so output is reproducible.

**Surface.** Three new flags on `query`; JSON
output unchanged.

**Effort.** S (a day or two).

**Depends on.** Nothing.

**Open questions.** Sort by computed fields
(S-4)? Yes, when S-4 lands.

### Q-3: Body full-text search in query

**Goal.** Filter files by body content as well as
front matter.

**Sketch.** New flag `--body-contains REGEX`:

```bash
mdsmith query 'kind: "doc"' --body-contains 'TODO|FIXME'
```

Anchors to body only (not front matter, not
inside fenced code unless `--include-code`). The
regex engine is Go's `regexp` package.

**Surface.** New flag; JSON output includes a
`matches` array per file when `--match-context`
set.

**Effort.** S–M.

**Depends on.** Nothing.

**Open questions.** Should it support multiple
patterns (AND / OR)? One pattern keeps the
surface small; users can pipe through `grep` for
more complex needs.

### Q-4: Date arithmetic with duration strings

**Goal.** Make CUE queries readable for "due in
the next 7 days":

```bash
mdsmith query 'due: <=time.Add(time.Now(), "7d24h")' tasks/
```

**Sketch.** Add a small CUE package
`internal/cue/time.cue` that defines `Add`,
`Sub`, and parsing helpers, exposed in the query
namespace by default. Match the duration-string
grammar mdbase uses (`"7d"`, `"1w"`, `"1M"`,
`"1y"`, `"2h"`, `"30m"`, `"45s"`; capital M for
months, lowercase for minutes — error on
ambiguous unit).

The CUE package wraps `time.Parse` /
`time.ParseDuration` with month / year support
that Go's `time` lacks; clamp month-overflow to
the last day of the target month, matching
mdbase.

**Surface.** New CUE helpers visible in `query`
expressions; documented in
`docs/reference/cli/query.md`.

**Effort.** M.

**Depends on.** Nothing.

**Open questions.** Time zone — query against
local time or UTC? mdbase uses an `mdbase.yaml`-
configurable IANA zone. mdsmith could read a
top-level `timezone:` config key.

### Q-7: Bases-compatible expression syntax

**Goal.** Optionally accept Obsidian Bases-style
expressions in `mdsmith query`, so users coming
from a vault don't have to learn CUE for simple
filters.

**Sketch.** Add a `--lang bases` flag:

```bash
mdsmith query --lang bases 'status == "open" && priority >= 3'
mdsmith query 'status: "open"'   # CUE, current default
```

Translation happens at the boundary: parse Bases
syntax into a CUE struct literal and unify as
today. Only the subset that maps cleanly: `==`,
`!=`, `<`, `>`, `<=`, `>=`, `&&`, `||`, `!`,
plus `.startsWith` / `.contains` / `.matches` on
strings.

**Surface.** New `--lang` flag; new doc page on
the supported subset.

**Effort.** M. Parser is small; the work is the
test matrix.

**Depends on.** Nothing.

**Open questions.** Worth the cost? Argument for:
lower onboarding for Obsidian users. Argument
against: two query languages, two failure modes,
two doc surfaces. **Recommendation: defer until a
real user asks for it.** The exploration here is
primarily evaluative. CUE is more powerful and
already documented; Bases is more familiar to
some. If we do add it, structure it as a
translation pass, not a parallel implementation.

## Operational plan

### C-6: `--dry-run` for fix

**Goal.** Show what `mdsmith fix` would change
without writing.

**Sketch.** New flag `--dry-run`:

```bash
mdsmith fix --dry-run docs/
# docs/api.md: would fix 3 violations (MDS001, MDS006, MDS019)
# docs/index.md: would regenerate <?catalog?> body
```

Emits the same diagnostics shape as `check` plus
a per-file summary of fixed-vs-unfixed counts.
Combine with `--format json` for tooling.

**Surface.** New `--dry-run` flag.

**Effort.** S. The fix engine already produces a
fixed-content buffer before writing; gate the
write.

**Depends on.** Nothing.

**Open questions.** None.

## Out-of-scope items: rationale

For each "out-of-scope" gap, a one-line reason and
the principled alternative.

| ID  | Why not                                                                    | Alternative                                 |
|-----|----------------------------------------------------------------------------|---------------------------------------------|
| S-5 | mdsmith does not author files; agents and humans do                        | Use `mdbase create` for typed scaffolding   |
| C-1 | mdsmith is lint-and-fix, not author-and-edit                               | Editor / agent / mdbase                     |
| C-2 | Same as C-1                                                                | Same                                        |
| C-3 | Same as C-1                                                                | Same                                        |
| C-5 | Batch CRUD is out-of-scope; lint/fix already scopes by glob                | mdbase batch ops, or a shell loop           |
| L-2 | ID-field resolution requires a canonical ID system mdsmith does not impose | Use mdbase if you need ID-keyed links       |
| L-5 | Multi-hop traversal couples query with link graph; large complexity        | mdbase Bases queries with `asFile()`        |
| Q-5 | Aggregations grow into a database surface mdsmith does not own             | mdbase Query+ formulas / summaries          |
| Q-6 | Same coupling as L-5                                                       | Same                                        |
| P-1 | Stateless re-read is a design feature: deterministic, no stale state       | mdbase SQLite cache for vault-scale work    |
| P-2 | LSP per-session covers editor flows; persistent daemon adds risk           | LSP, plus mdbase watch mode for vault flows |
| V-1 | Migrations require write-back to data files; mdsmith does not              | mdbase migrate manifests                    |
| X-1 | mdsmith is the implementation; spec-first costs maintenance                | If forks happen, revisit                    |

The recurring theme: mdsmith owns the
**read-and-fix-source** layer. Anything that wants
to **author or edit data** crosses a category line.
mdbase's CRUD-and-cache layer is the principled
home for those features.

### A closer look at P-1 (SQLite cache)

The one-line rationale above hides a real design
question worth pulling apart, since the cache is
mdbase's biggest runtime difference from mdsmith.

**Why SQLite at all.** mdbase carries an index of
files, parsed front matter, computed-field
results, and link edges. At vault scale (5k–50k
files) re-reading every file on every query is
slow: seconds per query versus milliseconds. The
spec calls SQLite "Recommended" because it gives
ACID writes, queryable indexes, single-file
storage, and ubiquity, with the trade-off that
the cache file is binary and not human-readable
(spec §13). Some impls might choose a JSON line
file or an in-memory map; SQLite is the default
because the queries are relational.

**Staleness check at startup.** A correct cache
must invalidate when source files change.
mdbase's spec §13 mandates two signals:

- **Modification time (mtime).** Cheap. Used as
  the first-pass check. If a file's mtime is
  newer than the cached entry's, re-parse it.
- **Content hash.** Used when mtime is
  unreliable (network filesystems, build systems
  that touch files without changing content). On
  a hash mismatch the entry is rebuilt.

The cache is rebuildable from the files alone, so
"correct on stale" is acceptable: the worst case
is an extra parse pass at startup. The cache is
also deletable without data loss — `.mdbase/` can
be removed at any time.

A read-only command like `mdbase query` walks
every entry and verifies mtime against the cached
record before trusting it. If the cache is large
relative to the workspace this becomes the
bottleneck; impls add a watcher (spec §15) so the
cache stays current between queries.

**What queries actually run on the cache.** Three
shapes dominate in mdbase's CRUD and query
surfaces:

1. **Front-matter filtering.** "All files where
   `status == 'open'` and `priority >= 3`." The
   cache stores parsed FM as columns or a JSON
   blob with a generated indexing layer; SQLite's
   `WHERE` does the rest. Equivalent to
   `SELECT path FROM files WHERE status='open' AND priority>=3`.
2. **Backlink lookup.** "What files link to
   `docs/api.md` (or its `auth` heading)?" Stored
   as a normalized link-edge table:
   `(source_path, target_path, target_anchor, link_kind)`.
   A point query finds incoming edges in one
   index lookup.
3. **ID resolution.** "Resolve `[[chen-2024]]` to
   a file." A unique index on the configured
   `id_field` makes this O(log n) instead of a
   full scan.

Plus the L4 traversal (`link.asFile().property`)
which is recursive and benefits from the
denormalized FM in the cache.

**Alternatives mdsmith could consider.** If
mdsmith ever needed an index — for backlinks
(L-4), wikilinks (L-1), or fast `query` over
large repos — these are the candidate
implementations, ranked by complexity:

1. **No cache, parallel re-read.** What mdsmith
   does today. Reads files in parallel, parses
   each, builds an in-memory link graph for the
   duration of the run. Determinism is high,
   memory is bounded, but cold-start time scales
   linearly with file count. Fine to ~5k files.
2. **In-memory index with mtime invalidation,
   process-scoped.** The LSP server already does
   this for compiled config. Extend to a link
   graph and FM map. Survives one editor session;
   does not survive process exit. Good fit for
   LSP, no fit for one-shot CLI.
3. **Memory-mapped file index** (e.g.,
   FlatBuffers or a Go `gob` blob). Persistent
   across runs, bounded format, no SQL. Requires
   a custom query API; no `WHERE`-clause power.
   Fast load (mmap is constant-time) but every
   query rescans the chunk it cares about. Good
   for backlinks (point lookup) and tag indexes;
   bad for FM filtering with multiple predicates.
4. **Embedded key-value store** (BoltDB,
   Badger). Persistent, transactional, no
   external dependency. Schema is hand-rolled
   like option 3. Easier to evolve than mmap.
   No SQL means complex predicates need
   application-side filtering after a key scan.
5. **SQLite (mdbase's choice).** Persistent,
   ACID, full SQL, mature ecosystem.
   `modernc.org/sqlite` is a pure-Go driver so a
   static binary stays static. Trade-off: schema
   migrations and an extra dependency surface.
6. **Full-text search engine** (Bleve,
   Tantivy via cgo). Useful only if Q-3 (body
   FTS) becomes central. Overkill otherwise.

**Where mdsmith would land.** Probably option 2
for the LSP server (extending what's there) and
option 4 (BoltDB-style) if a persistent on-disk
cache is ever needed. Option 5 (SQLite) carries
the most operational baggage — a binary cache
file that needs schema versioning, a driver
dependency, and a "delete it if it gets weird"
escape hatch — for capability mdsmith does not
yet need. The strongest argument for SQLite is
mdbase's Q-5 aggregations; mdsmith does not plan
to ship those.

The right framing for now: **add an in-memory
link graph to the LSP server (option 2)** when
plan 131 lands, since `documentSymbol` /
`references` / call hierarchy all want one. If a
follow-up plan adds backlinks to the CLI (L-4),
prototype with parallel re-read (option 1) and
only escalate to a persistent cache when real
profiling shows a regression on a real corpus.

## In-flight items

| ID  | Plan / PR          | Notes                                         |
|-----|--------------------|-----------------------------------------------|
| H-1 | plan 122           | Hover for rule and directive docs             |
| H-2 | plan 131 (PR #238) | Symbol navigation, references, call hierarchy |

## Suggested sequencing

If a maintainer wants to ship a few of these, the
ones with the highest payoff per unit of effort
are:

1. **L-1 (wikilink awareness)** — unlocks Obsidian
   vault use; small effort, high impact.
2. **L-4 (backlinks subcommand)** — surfaces
   value already in MDS027; small effort.
3. **M-1 (field-presence kind assignment)** —
   ergonomic win; small effort.
4. **C-6 (dry-run for fix)** — agent-friendly;
   small effort.
5. **S-1 (inline schema)** — ergonomic win for
   teams not wanting a separate proto.md; medium
   effort but unlocks S-3, S-4, S-6.
6. **Q-1 / Q-2 (sort and limit)** — small effort,
   makes `query` actually useful in scripts.
7. **L-3 / C-4 (rename refactor)** — large effort
   but high-leverage; closes the biggest workflow
   gap.

The remaining items are nice-to-haves whose
priority depends on real user demand.

## What this exercise revealed

Three structural observations fall out of working
through the gaps systematically.

1. **mdsmith's stateless single-pass design is a
   feature, not a missing piece.** Most "missing"
   features (cache, watch, CRUD, migrations) trade
   determinism for capability. The right answer
   for vault-scale typed-record work is mdbase,
   not a richer mdsmith.

2. **mdsmith's schema mechanism is closer to
   mdbase's than the surface suggests.** Both use
   Markdown files with declarative front matter as
   schemas. The differences (CUE vs DSL, body
   structure check vs body-as-docs) are real but
   smaller than they look. S-1 + S-2 + S-3 close
   most of the ergonomic gap.

3. **The link graph is mdsmith's biggest
   underused asset.** MDS027 already builds it.
   Backlinks (L-4), wikilinks (L-1), and rename
   refactor (L-3) all flow from making that graph
   first-class instead of a per-rule internal.
