---
id: 148
title: Named field-type shortcuts for inline schemas
status: "✅"
model: sonnet
depends-on: [146]
summary: >-
  Ship a small CUE library of common field
  patterns (`date`, `datetime`, `email`, `url`,
  `filename`) and let inline schemas (plan 146)
  reference them by short name. Schemas in
  `proto.md` files import the same library.
---
# Named field-type shortcuts for inline schemas

## Goal

Make inline schemas (plan 146) shorter and more
readable for the common case. A user writing
`created: date` should not have to re-derive the
ISO-date regex; nor should three projects each
land on subtly different versions of the same
pattern.

## Background

mdbase ships a fixed list of named scalar types
(`string`, `int`, `number`, `bool`, `date`,
`datetime`, `time`, `enum`, `link`). mdsmith uses
CUE, which is more expressive but asks each
project to spell out its own date and email
patterns. The mdbase research records this gap as
**S-2**: not a missing capability, but a missing
ergonomic layer over CUE.

Plan 146 (inline schemas) is the trigger. Before
plan 146 the patterns can live in a shared
`proto.md` and be reused by import; after plan
146 each inline `frontmatter:` block re-derives
them.

## Non-Goals

- Replacing CUE. The shortcuts resolve to CUE
  expressions; everything CUE accepts is still
  accepted.
- Custom user-defined shortcuts. A real project
  can already define one in its own `proto.md`
  schema. If user shortcuts become a real
  request, that is a separate plan.
- Validating against system locale or timezone.
  Date and datetime patterns match strings of the
  expected ISO shape; semantic checks (is the day
  valid for the month) are out of scope.

## Design

### The shortcut library

A package lives at a stable, importable module
path. **Settled paths (on landing):**

- On-disk source: `cue/types/types.cue` in this
  repository, declaring `package types`.
- CUE import path: `github.com/jeduden/mdsmith/types`
  (reserved for future CUE-native consumers; the
  literal `import` statement is not yet wired up).
- Distribution: embedded asset. `cue/types/types.go`
  uses `//go:embed types.cue`; `internal/schema`
  reads the source for the drift test and seeds its
  runtime registry from the same definitions.

The initial vocabulary:

```cue
package types

#date:     =~"^\\d{4}-\\d{2}-\\d{2}$"
#datetime: =~"^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(Z|[+-]\\d{2}:\\d{2})?$"
#time:     =~"^\\d{2}:\\d{2}(:\\d{2})?$"
#email:    =~"^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$"
#url:      =~"^https?://"
#filename: =~"^[A-Za-z0-9._-]+\\.md$"
#nonEmpty: string & != ""
```

The contract a user sees is the import path and the
symbol names. Only one surface is wired up today:
the bare-name shortcut described below. A literal
`import "github.com/jeduden/mdsmith/types"` is
reserved for a future plan.

### File-schema use

A `proto.md` schema imports the package and
references the definitions:

```cue
package proto

import "github.com/jeduden/mdsmith/types"

created:  types.#date
modified: types.#datetime
homepage: types.#url
```

Existing `proto.md` schemas keep working. The
import is opt-in.

### Inline-schema shortcut form

Inline schemas (plan 146) accept the unprefixed
short name as a string value:

```yaml
kinds:
  rfc:
    schema:
      frontmatter:
        created: date
        modified: datetime
        contact: email
        homepage: url
```

The loader splits values into two cases:

- **Bare name** — a YAML scalar that is a single
  identifier (no quotes, no operators, no
  whitespace). Must resolve in the registry, or
  the loader rejects the config with an error
  naming the field and the unknown name.
- **Anything else** — values containing
  operators, quotes, parentheses, or whitespace
  — passes through as raw CUE. So
  `status: '"open" | "done"'` continues to work
  unchanged.

The split keeps intent unambiguous. A typo on a
bare shortcut surfaces early. A real CUE
expression is never silently re-read as a
shortcut.

### Composition with CUE

A field that needs both a shortcut and an extra
constraint uses inline CUE syntax:

```yaml
frontmatter:
  created: 'date & >="2020-01-01"'
```

The shortcut is parsed only when the value is
exactly the bare short name. Anything more
complex stays raw CUE.

### Surface

- A new public package import path for the
  library.
- A new bare-name shortcut in inline-schema
  `frontmatter:` values.
- A documentation page listing the registered
  names and what each resolves to.

## Tasks

1. ✅ Create the CUE source for the shortcut
   library at `cue/types/types.cue`, declaring
   `package types`. The CUE import path
   `github.com/jeduden/mdsmith/types` is documented
   but not yet wired up as a CUE module; the
   user-facing surface is the bare-name shortcut.
2. ✅ Ship the library via `go:embed` in
   `cue/types/types.go`. The runtime registry seeds
   from the same source, so `proto.md` schemas
   resolve shortcuts with no network access — the
   library is in the binary.
3. ✅ Registry of short names → canonical CUE
   expressions lives at
   `internal/schema/shortcuts.go` (kept next to
   `frontmatterExpr`, which is the single
   substitution point shared by inline and
   file-based parsing).
4. ✅ `frontmatterExpr` in
   `internal/schema/parse_inline.go` calls
   `resolveBareName` first. Bare identifiers
   matching `[a-zA-Z_][a-zA-Z0-9_-]*` go through:
   registered shortcuts are substituted, CUE
   built-ins (`string`, `int`, `bool`, …) pass
   through verbatim, and unknown bare names error
   with the field path. Operators, whitespace,
   and quoted forms pass through unchanged.
5. ✅ Documentation page at
   `docs/reference/schema-types.md` lists every
   registered name, its canonical CUE, accept /
   reject examples, and an inline / proto.md usage
   walk-through.
6. ✅ Cross-linked from the
   [file-kinds guide](../docs/guides/file-kinds.md),
   the
   [schemas guide](../docs/guides/schemas.md), and
   the
   [MDS020 README](../internal/rules/MDS020-required-structure/README.md).
7. ✅ Tests in
   `internal/schema/shortcuts_test.go`:

  - each shortcut accepts canonical inputs and
    rejects clear violations
    (`TestShortcutRegistry_CanonicalsCompileAndMatch`);
  - a value containing operators (`"open" | "done"`,
    `date & >="2020-01-01"`) is parsed as raw CUE
    (`TestResolveBareName_IgnoresNonBareCandidates`,
    `TestParseInline_RawCUEPassesThroughUnchanged`);
  - an unknown bare name errors with the field name
    and the unknown identifier
    (`TestParseInline_UnknownShortcutErrorNamesField`);
  - an offline `proto.md` schema resolves through
    the embedded library
    (`TestValidate_File_ShortcutWorksOffline`);
  - the registry stays in sync with the embedded CUE
    (`TestShortcutRegistry_MatchesEmbeddedCUE`).

## Acceptance Criteria

- [x] The shortcut library defines `#date`,
      `#datetime`, `#time`, `#email`, `#url`,
      `#filename`, `#nonEmpty` at the chosen
      import path.
- [x] An external `proto.md` schema resolves the
      shortcut library without network access —
      the library is embedded in the binary
      (`TestValidate_File_ShortcutWorksOffline`
      in `internal/schema/shortcuts_test.go`).
- [x] An inline schema with `created: date`
      validates `2024-05-01` and rejects
      `2024-5-1`
      (`TestValidate_Inline_ShortcutAcceptsAndRejects`).
- [x] An inline-schema value that is not a
      registered bare name (`status: '"open" |
      "done"'`) is parsed as raw CUE without
      lookup
      (`TestParseInline_RawCUEPassesThroughUnchanged`).
- [x] An inline-schema value that looks like a
      shortcut but is unknown (`created:
      iso-date`) produces a config error
      naming the field and the unknown name
      (`TestParseInline_UnknownShortcutErrorNamesField`).
- [x] A new `docs/reference/schema-types.md`
      page lists every registered name with the
      canonical CUE expression and an example.
- [x] All tests pass: `go test ./...`
- [x] `go tool golangci-lint run` reports no
      issues.
