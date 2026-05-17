---
command: fix
summary: Auto-fix lint issues in Markdown files in place.
---
# `mdsmith fix`

Auto-fix lint issues in Markdown files in place. Multi-pass
fixing resolves cascading changes in one invocation.

```text
mdsmith fix [flags] [files...]
```

Files can be paths, directories (walked recursively for
`*.md` and `*.markdown`), or glob patterns. Stdin is
rejected — files must be writable. With no file arguments,
files are discovered from `.mdsmith.yml` `files:` patterns.

## Flags

| Flag                | Default | Description                            |
|---------------------|---------|----------------------------------------|
| `-c`, `--config`    | auto    | Override config path (auto-discovers)  |
| `-f`, `--format`    | `text`  | `text` or `json`                       |
| `--max-input-size`  | `2MB`   | Max file size (e.g. `2MB`, `0`=none)   |
| `--no-color`        | false   | Plain output                           |
| `--follow-symlinks` | config  | Follow symlinks; tri-state — see below |
| `--no-gitignore`    | false   | Skip gitignore filtering               |
| `-q`, `--quiet`     | false   | Suppress non-error output              |
| `-v`, `--verbose`   | false   | Show config, files, and rules          |
| `--explain`         | false   | Attach per-leaf rule provenance        |
| `--dry-run`         | false   | Preview fixes without writing to disk  |

`--follow-symlinks` semantics match
[`mdsmith check`](check.md#flags).

## Dry-run preview

`--dry-run` builds the fixed content per file but skips the write.
Per-file output lists which rules would fire and the violation count.
Files with no fixes do not appear.

```bash
mdsmith fix --dry-run docs/
```

Example output:

```text
docs/api.md: would fix 3 violations (MDS001, MDS006)
docs/index.md: would fix 1 violation (MDS002)
stats: checked=12 fixed=0 failures=0 unfixed=4 would-fix=4
```

The summary includes `would-fix=N` and always prints `fixed=0`
(nothing is written).

With `--format json`, each file is an object:

```json
[
  {
    "path": "docs/api.md",
    "would_fix": 3,
    "rules": ["MDS001", "MDS006"],
    "diagnostics": []
  }
]
```

**Exit code:** `0` when every violation is fixable; non-zero when
unfixable violations remain. Matches the real-run exit code.

## Examples

```bash
mdsmith fix README.md            # fix a single file
mdsmith fix docs/                # fix a tree
mdsmith fix --dry-run docs/      # preview without writing
mdsmith fix --explain plan/      # show provenance for unfixed leftovers
```

## Pre-commit

```yaml
# lefthook.yml
pre-commit:
  commands:
    mdsmith:
      glob: "*.{md,markdown}"
      run: mdsmith fix {staged_files}
      stage_fixed: true
```

## Exit codes

| Code | Meaning                        |
|------|--------------------------------|
| 0    | No remaining issues            |
| 1    | Issues remain after fixing     |
| 2    | Runtime or configuration error |

## See also

- [`mdsmith check`](check.md) — read-only sibling
- [`mdsmith merge-driver`](merge-driver.md) — Git merge
  driver that uses `fix` to resolve generated-section
  conflicts
