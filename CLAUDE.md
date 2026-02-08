# CLAUDE.md

## Project

tidymark — a Markdown linter written in Go.

## Build & Test Commands

- `go build ./...` — build all packages
- `go test ./...` — run all tests
- `go test -run TestName ./pkg/...` — run a specific test
- `golangci-lint run` — run linter
- `go vet ./...` — run go vet

## Project Layout

Follow the [standard Go project layout](https://go.dev/doc/modules/layout):

- `cmd/tidymark/` — main application entry point
- `internal/` — private packages not importable by other modules
- `testdata/` — test fixtures (markdown files for testing rules)

## Development Workflow

- New features are test-driven: write a failing test (red), make it pass (green), commit
- Keep commits small and focused on one change

## Code Style

- Follow standard Go conventions (gofmt, goimports)
- Use golangci-lint for linting
- Keep functions small and focused
- Error messages should be lowercase, no trailing punctuation
- Prefer returning errors over panicking

## CLI Design

### Usage

```
tidymark [flags] [files...]
```

Files are positional arguments. Accepts multiple file paths, directories, and glob patterns.
No file args and no stdin exits 0 (graceful empty invocation for pre-commit hooks).

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config <file>` | `-c` | auto-discover | Override config file path |
| `--fix` | | false | Auto-fix issues in place |
| `--format <fmt>` | `-f` | `text` | Output format: `text`, `json` |
| `--no-color` | | false | Disable ANSI colors |
| `--quiet` | `-q` | false | Suppress non-error output |
| `--version` | `-v` | | Print version and exit |
| `--help` | `-h` | | Show help |

Use `--` to separate flags from filenames starting with `-`.

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No lint issues found |
| 1 | Lint issues found |
| 2 | Runtime or configuration error |

### Output

Lint output goes to **stderr**. Format:

**text** (default):
```
README.md:10:5 TM001 line too long (120 > 80)
docs/guide.md:3:1 TM002 first line should be a heading
```
Pattern: `file:line:col rule message`

**json**:
```json
[
  {
    "file": "README.md",
    "line": 10,
    "column": 5,
    "rule": "TM001",
    "name": "line-length",
    "severity": "error",
    "message": "line too long (120 > 80)"
  }
]
```

### Pre-commit (lefthook)

```yaml
# lefthook.yml
pre-commit:
  commands:
    tidymark:
      glob: "*.{md,markdown}"
      run: tidymark {staged_files}
      # To auto-fix and re-stage:
      # run: tidymark --fix {staged_files}
      # stage_fixed: true
```

## Config File

### Discovery

1. If `--config` is given, use that file
2. Otherwise, look for `.tidymark.yml` in the current directory
3. Walk up parent directories to the filesystem or git repo root
4. No config file found = use defaults (all rules enabled, default settings)

### Format

YAML (`.tidymark.yml`):

```yaml
# Global severity: "error" or "warning" (default: "error")
severity: error

# Line length limit (0 = disabled)
line-length: 80

# Rules configuration
# Each rule can be: true (enable), false (disable), or an object with settings
rules:
  heading-style:
    style: atx          # "atx" (#) or "setext" (underline)
  line-length:
    max: 80
    # Skip lines that contain only a URL or code block
    strict: false
  no-trailing-spaces: true
  no-multiple-blanks: true
  first-line-heading:
    level: 1             # Required heading level for first line
  no-hard-tabs: true
  list-indent:
    spaces: 2
  fenced-code-style:
    style: backtick      # "backtick" or "tilde"

# Files to ignore (glob patterns)
ignore:
  - "vendor/**"
  - "node_modules/**"
  - "CHANGELOG.md"
```

### Rule naming

Rules use kebab-case names (e.g. `line-length`, `no-trailing-spaces`).
Each rule also has a short ID (e.g. `TM001`) used in output.

### Config merging

No directory-level config merging. One config file applies to the entire run.
Use `--config` to explicitly select a different config for different contexts.
