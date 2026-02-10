# CLI Subcommands

## Goal

Restructure the CLI from flat flags to subcommands (`check`, `fix`, `init`) for
clearer UX and to support `tidymark init` which dumps a config file with all
rule defaults.

## Prerequisites

- Plan 14 (config settings) — needed for `DumpDefaults()` used by `init`

## Tasks

1. Restructure `cmd/tidymark/main.go` to use subcommands. The top-level
   command accepts only global flags (`--version`, `--help`). Subcommands:
   - `tidymark check [flags] [files...]` — current lint behavior
   - `tidymark fix [flags] [files...]` — current `--fix` behavior
   - `tidymark init` — generate `.tidymark.yml`

2. Implement `check` subcommand:
   - Flags: `--config`, `--format`, `--no-color`, `--quiet`
   - Accepts files as positional args or stdin (pipe)
   - Extract current lint logic from `run()` into `runCheck()`
   - Same exit codes: 0 = clean, 1 = violations, 2 = error

3. Implement `fix` subcommand:
   - Flags: `--config`, `--format`, `--no-color`, `--quiet`
   - Accepts files as positional args only
   - Rejects stdin with exit code 2 and error message
   - Extract current fix logic from `run()` into `runFix()`

4. Implement `init` subcommand:
   - No flags beyond global
   - Calls `config.DumpDefaults()` (from plan 14) to get a config with
     all rule defaults
   - Sets `front-matter: true` as default
   - Serializes to YAML and writes `.tidymark.yml` in the current
     directory
   - If `.tidymark.yml` already exists, print error and exit 2
   - Print confirmation message to stderr

5. Top-level behavior:
   - `tidymark` (no subcommand, no args) → print usage to stderr, exit 0
   - `tidymark --version` / `-v` → print version, exit 0
   - `tidymark --help` / `-h` → print usage with subcommand list, exit 0
   - Unknown subcommand → print error + usage, exit 2

6. Backwards compatibility (optional): if the first positional arg is a
   file path (not a known subcommand), treat it as `tidymark check
   <args>` with a deprecation warning. This eases migration. Can be
   removed in a future version.

7. Update `cmd/tidymark/e2e_test.go`:
   - Existing tests that use `runBinary(t, "", path)` should be updated
     to use `runBinary(t, "", "check", path)` (or test backwards compat)
   - Add tests for each subcommand:
     - `check` with clean file, dirty file, stdin, JSON format
     - `fix` with fixable file, stdin rejection
     - `init` creates config file, refuses if exists
   - Add test for `--version`, `--help`, no-args usage

## Acceptance Criteria

- [ ] `tidymark check [flags] [files...]` works as the current lint behavior
- [ ] `tidymark fix [flags] [files...]` works as the current `--fix` behavior
- [ ] `tidymark init` creates `.tidymark.yml` with all rule defaults
- [ ] `tidymark` (no args) prints usage and exits 0
- [ ] `tidymark --version` prints version and exits 0
- [ ] Stdin works with `check`, is rejected by `fix`
- [ ] E2E tests updated for subcommand interface
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
