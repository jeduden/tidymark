# Respect .gitignore When Resolving Files

## Goal

Make tidymark skip files matched by `.gitignore` during directory walking, so
users don't need to duplicate gitignore patterns in `.tidymark.yml`. Add a
`--no-gitignore` flag to disable this behavior.

## Prerequisites

None — this plan can start immediately.

## Tasks

1. Add a gitignore-aware file walker. Use an existing Go library (e.g.,
   `github.com/go-git/go-git/v5/plumbing/format/gitignore` or
   `github.com/sabhiram/go-gitignore`) to parse `.gitignore` files and
   match paths. Alternatively, shell out to `git ls-files` or
   `git check-ignore` if a git repo is detected — simpler but requires
   git to be installed.

   Recommended approach: use a pure-Go gitignore library so tidymark
   works outside git repos and without git installed.

2. Update `internal/lint/files.go`:
   - Add a `ResolveOpts` struct with a `UseGitignore bool` field
     (default `true`)
   - Add `ResolveFilesWithOpts(args []string, opts ResolveOpts)` that
     wraps the current logic
   - Keep `ResolveFiles(args []string)` as a convenience wrapper that
     calls with defaults (`UseGitignore: true`)
   - In `walkDir`, when `UseGitignore` is true, locate `.gitignore`
     files up the directory tree (and nested `.gitignore` files in
     subdirectories) and skip matched paths
   - Also respect the global gitignore (`core.excludesFile` from git
     config, typically `~/.config/git/ignore`)

3. Add `--no-gitignore` flag to the CLI:
   - In `cmd/tidymark/main.go`, add a `--no-gitignore` bool flag
     (default `false`)
   - Pass the flag value through to `ResolveFilesWithOpts`
   - If plan 15 (subcommands) is implemented, add the flag to `check`
     and `fix` subcommands

4. Add unit tests in `internal/lint/files_test.go`:
   - Directory with `.gitignore` excluding `*.md` in a subdirectory —
     those files are skipped
   - Nested `.gitignore` in a subdirectory is respected
   - `UseGitignore: false` includes all files
   - No `.gitignore` present — all files included (no error)
   - Files passed explicitly by path are NOT filtered by gitignore
     (only directory walking respects it)

5. Add an E2E test:
   - Create a temp dir with a `.gitignore` that excludes `ignored/`
   - Place a markdown file with violations in `ignored/`
   - Run tidymark on the directory — expect exit 0 (ignored file skipped)
   - Run with `--no-gitignore` — expect exit 1 (violations found)

6. Document the behavior in `README.md`: tidymark respects `.gitignore`
   by default when walking directories; use `--no-gitignore` to disable.

## Acceptance Criteria

- [ ] Directory walking skips files matched by `.gitignore`
- [ ] Nested `.gitignore` files in subdirectories are respected
- [ ] Explicitly named file paths are NOT filtered by gitignore
- [ ] `--no-gitignore` flag disables gitignore filtering
- [ ] Works without git installed (pure-Go implementation)
- [ ] Works outside git repositories (no `.gitignore` = no filtering)
- [ ] Unit tests for gitignore-aware walking pass
- [ ] E2E test for `--no-gitignore` passes
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
