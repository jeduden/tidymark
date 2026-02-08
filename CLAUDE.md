# CLAUDE.md

## Project

tidymark — a Markdown linter written in Go.

## Build & Test Commands

- `go build ./...` — build all packages
- `go test ./...` — run all tests
- `go test -run TestName ./pkg/...` — run a specific test
- `golangci-lint run` — run linter
- `go vet ./...` — run go vet

## Code Style

- Follow standard Go conventions (gofmt, goimports)
- Use golangci-lint for linting
- Keep functions small and focused
- Error messages should be lowercase, no trailing punctuation
- Prefer returning errors over panicking
