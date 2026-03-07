# CLAUDE.md

## Project

mdsmith — a Markdown linter written in Go.

<?include
file: DEVELOPMENT.md
strip-frontmatter: "true"
?>
Build and test reference for mdsmith contributors.

## Build & Test Commands

Requires Go 1.24+.

- `go build ./...` — build all packages
- `go test ./...` — run all tests
- `go test -run TestName ./...` — run a specific test
- `go run ./cmd/mdsmith check .` — lint markdown
- `go run ./cmd/mdsmith fix .` — auto-fix markdown
- `go tool golangci-lint run` — run linter
- `go vet ./...` — run go vet

## Project Layout

Follow the [standard Go project
layout](https://go.dev/doc/modules/layout):

- `cmd/mdsmith/` — main application entry point
- `internal/` — private packages not importable by
  other modules
- `internal/rules/` — rule documentation
  (`internal/rules/<id>-<name>/README.md`)
- `testdata/` — test fixtures (markdown files for
  testing rules)

## Code Style

- Follow standard Go conventions (gofmt, goimports)
- Use golangci-lint for linting
- Keep functions small and focused
- Error messages should be lowercase, no trailing
  punctuation
- Prefer returning errors over panicking

## Test Fixtures

Rule test fixtures live in
`internal/rules/<id>-<name>/`. Each rule has `good/` and
`bad/` examples (or `good.md` / `bad.md`).

Good fixtures must pass **all** rules, not just their
own. When a good fixture uses non-default settings
(e.g. setext headings, tilde fences), add a matching
override in `.mdsmith.yml` so that `mdsmith check .`
also passes.

Bad fixtures are excluded via the `ignore:` section in
`.mdsmith.yml`.

## Merge Conflicts in Generated Sections

`PLAN.md` and `README.md` have auto-generated
sections between `<?name` ...
`?>` and `<?/name?>` markers. Run `mdsmith fix <file>`
after merging — it regenerates these sections. The
`merge-driver` command automates this:

```bash
mdsmith merge-driver install [files...]
```

Run `mdsmith merge-driver install` once per clone.
<?/include?>

## Development Workflow

- Any change follows Red / Green TDD: write a failing
  test (red), make it pass (green), commit
- Keep commits small and focused on one change
- Run `mdsmith check .` before committing to ensure all
  markdown files pass linting

## PR Workflow

Use `gh` for all GitHub PR operations:

```bash
# View PR comments
gh pr view <number> --comments

# List review comments on a PR
gh api "$(gh repo view --json nameWithOwner \
  -q '.nameWithOwner')/pulls/<number>/comments" \
  --paginate

# Push updates after addressing comments
git push origin <branch>
```

These commands are auto-approved in
`.claude/settings.json`.

## Plans

Task plans live in [`plan/`](plan/). See
[`PLAN.md`](PLAN.md) for the current status of all
plans. Use [`plan/proto.md`](plan/proto.md) as a
template when creating new plans.

## Writing Guidelines

When writing descriptions, state the concrete constraint:
what specific data must satisfy what condition. Name the
inputs (front matter fields, glob pattern, heading level)
not just the mechanism. Avoid vague verbs (match, sync,
reflect) without saying what is checked against what.
