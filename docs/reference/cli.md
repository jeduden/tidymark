---
summary: CLI commands, flags, exit codes, and output format.
---
# CLI Design

## Usage

```text
mdsmith <command> [flags] [files...]
```

## Commands

| Command        | Description                                    |
|----------------|------------------------------------------------|
| `check`        | Lint files                                     |
| `fix`          | Auto-fix issues in place                       |
| `query`        | Select files by CUE expression on front matter |
| `help`         | Show help for rules/topics                     |
| `metrics`      | List and rank shared metrics                   |
| `merge-driver` | Git merge driver for regenerable sections      |
| `archetypes`   | Scaffold, list, show, and locate archetypes    |
| `kinds`        | Inspect declared kinds and per-file resolution |
| `init`         | Generate `.mdsmith.yml`                        |
| `version`      | Print version, exit                            |

The `check` and `fix` commands accept file paths,
directories, and glob patterns as positional arguments.
Pass `-` to read from stdin.

When no file arguments are given, `check` and `fix`
discover files using the `files` glob patterns from config
(default: `["**/*.md", "**/*.markdown"]`). If no files
match, exits 0.

## Subcommand Flags (check, fix)

| Flag                | Default | Description                             |
|---------------------|---------|-----------------------------------------|
| `-c`, `--config`    | auto    | Config path (auto-discovers by default) |
| `-f`, `--format`    | `text`  | `text` or `json`                        |
| `--max-input-size`  | `2MB`   | Max file size (e.g. `2MB`, `0`=none)    |
| `--no-color`        | false   | Plain output                            |
| `--follow-symlinks` | config  | Follow symlinks; tri-state — see below  |
| `--no-gitignore`    | false   | Skip gitignore                          |
| `-q`, `--quiet`     | false   | Quiet mode                              |
| `-v`, `--verbose`   | false   | Verbose output                          |
| `--explain`         | false   | Add config-provenance trailer per diag  |

Symlinks are skipped by default. This blocks a malicious
symlink from redirecting `check` or `fix` to files outside
the project. The rule applies to directory walks, glob
expansion, and explicit file or directory arguments:
`mdsmith check ./linked.md` silently skips a symlink named
on the command line. Opt-in follows only symlinks that
resolve to regular files; directory / FIFO / device / socket
targets are always skipped.

`--follow-symlinks` is tri-state:

- omitted — fall back to `follow-symlinks:` in
  `.mdsmith.yml` (default: skip)
- `--follow-symlinks` or `--follow-symlinks=true` — opt in
  for this run
- `--follow-symlinks=false` — force deny for this run, even
  when the loaded config has `follow-symlinks: true`

The old `no-follow-symlinks:` config key still parses and
emits a deprecation warning on stderr.

## Other Subcommand Flags

`query` accepts `-c`/`--config`, `-v`/`--verbose`,
`-0`/`--null`, and `--max-input-size`.

`metrics list` accepts `-f`/`--format` (`text` or `json`)
and `--scope` (only `file` is supported; defaults to
`file`).

`metrics rank` accepts `-c`/`--config`, `-f`/`--format`,
`--no-gitignore`, `--follow-symlinks`,
`--max-input-size`, plus `--metrics`, `--by`, `--order`,
`--top`.

## `archetypes` Subcommands

| Subcommand    | Description                                  |
|---------------|----------------------------------------------|
| `init [dir]`  | Scaffold archetype dir with example + README |
| `list`        | Print discovered archetypes (name + path)    |
| `show <name>` | Print raw schema source to stdout            |
| `path <name>` | Print resolved filesystem path               |

Archetype roots come from the top-level
`archetypes.roots` key in `.mdsmith.yml`. When absent,
`./archetypes` is used. `init` never mutates the
config file; it prints the snippet to add manually.
`list` exits `1` when no archetypes are discovered;
`show` and `path` exit `2` on unknown names.

## `kinds` Subcommands

| Subcommand                   | Description                              |
|------------------------------|------------------------------------------|
| `list [--json]`              | Declared kinds with merged bodies        |
| `show <name> [--json]`       | One kind's merged body                   |
| `path <name>`                | Kind's `required-structure.schema:` path |
| `resolve <file> [--json]`    | Per-leaf rule provenance for a file      |
| `why <file> <rule> [--json]` | Full merge chain for one rule, one file  |

`show` exits `2` on unknown name. `path` exits `2` when
the kind is unknown or carries no
`required-structure.schema:`. `resolve` and `why`
validate the file's front-matter `kinds:` against
declared kinds.

`mdsmith help kinds-cli` prints a one-screen summary;
`mdsmith help kinds` prints the kinds concept page.

### `--explain` on `check` / `fix`

Adds a one-line provenance trailer to each diagnostic
naming the rule and the source layer that won:

```text
plan/foo.md:11:1 MDS022 file too long (305 > 300)
  └─ max-file-length=default [kinds: plan]
```

The bracketed `[kinds: ...]` lists the file's effective
kind list. With `--format=json` the diagnostic carries
an `explanation` object — schema below.

### JSON schema

`--json` on each `kinds` subcommand emits a stable
structured form. The same provenance shape powers the
`explanation` object on `check --explain --format=json`.

`kinds list --json` and `kinds show --json`:

```json
{ "kinds": [
    { "name": "plan",
      "body": { "rules": {}, "categories": {} } } ] }
```

`kinds resolve <file> --json`:

```json
{
  "file": "doc.md",
  "kinds": [
    { "name": "plan", "sources": ["kind-assignment[0]"] }
  ],
  "rules": {
    "line-length": {
      "final": { "max": 80 },
      "leaves": {
        "max": {
          "final": 80,
          "winning_source": "default",
          "chain": [
            { "layer": "default", "source": "default",
              "value": 80, "touched": true }
          ]
        }
      }
    }
  },
  "categories": { "line": true },
  "explicit": { "line-length": true }
}
```

`kinds why <file> <rule> --json`:

```json
{
  "file": "doc.md", "rule": "line-length",
  "chain": [
    { "layer": "default", "source": "default",
      "value": {"max":80}, "touched": true },
    { "layer": "kind", "source": "kinds.wide",
      "touched": false }
  ],
  "final": { "max": 80 }
}
```

`check --explain --format=json` adds per diagnostic:

```json
{
  "explanation": {
    "rule": "line-length",
    "source": "kinds.wide",
    "kinds": ["wide"],
    "leaf_sources": {
      "enabled": "kinds.wide", "max": "kinds.wide"
    }
  }
}
```

Source labels: `default`, `kinds.<name>`,
`overrides[i]`, `front-matter override`.
Layer labels: `default`, `kind`, `override`,
`front-matter override`.

## `--max-input-size`

Sets the byte-size cap for any input file read by
commands that support the flag (`check`, `fix`, `query`,
`metrics rank`). Behavior on oversize input:

- `check` / `fix`: file is skipped and reported as a
  runtime error on stderr. Exit code follows the usual
  precedence — `1` when lint diagnostics are found (even
  if some files were skipped), `2` when only runtime
  errors occur.
- `query`: file is skipped; the per-file error is only
  printed on stderr when `--verbose` is set. Exit code
  `1` if no files matched, `0` on a match.
- `metrics rank`: the whole run fails with exit code
  `2` on the first oversize file.

An invalid `--max-input-size` value always exits `2`.
Accepts `KB`, `MB`, `GB` suffixes (binary:
1 MB = 1,048,576 bytes), bare integers (bytes), or `0`
to disable the limit. Default: `2MB`. The CLI flag
overrides the `max-input-size` key in `.mdsmith.yml`.

## Global Flags

| Flag     | Short | Description |
|----------|-------|-------------|
| `--help` | `-h`  | Show help   |

Use `--` to separate flags from filenames starting with `-`.

## Exit Codes

| Code | Meaning                        |
|------|--------------------------------|
| 0    | No lint issues found           |
| 1    | Lint issues found              |
| 2    | Runtime or configuration error |

## Output

Lint output goes to **stderr**. Format:

**text** (default):

```text
README.md:10:81 MDS001 line too long (120 > 80)
 8 | Previous line of context.
 9 | Another context line.
10 | This very long line exceeds the configured 80 character limit and keeps going...
·····················································································^
11 | Next line of context.
12 | Another context line.
```

Each diagnostic prints a header line (`file:line:col rule message`).
When source context is available, up to 5 surrounding lines appear
with a dot path (`····^`) pointing to the exact column.

**json**:

```json
[
  {
    "file": "README.md",
    "line": 10,
    "column": 81,
    "rule": "MDS001",
    "name": "line-length",
    "severity": "error",
    "message": "line too long (120 > 80)",
    "source_lines": ["Previous line.", "Another context.", "The long line...", "Next.", "Another."],
    "source_start_line": 8
  }
]
```

The `source_lines` and `source_start_line` fields are omitted when
source context is unavailable (e.g., empty diagnostics).

## Pre-commit (lefthook)

```yaml
# lefthook.yml
pre-commit:
  commands:
    mdsmith:
      glob: "*.{md,markdown}"
      run: mdsmith check {staged_files}
      # To auto-fix and re-stage:
      # run: mdsmith fix {staged_files}
      # stage_fixed: true
```
