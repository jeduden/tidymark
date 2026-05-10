---
id: 132
title: Package mdsmith as a Claude Code MCP plugin
status: "🔳"
model: sonnet
summary: >-
  Ship an `mdsmith mcp` subcommand that speaks the Model
  Context Protocol over stdio, exposing `check` and `fix`
  as MCP tools so Claude Code can lint and fix Markdown
  files without leaving the editor.
---
# Package mdsmith as a Claude Code MCP Plugin

## Goal

Surface mdsmith diagnostics and auto-fix inside Claude Code
by running `mdsmith mcp` as a local MCP server, so any
Claude Code session that edits Markdown files can call
`mdsmith_check` and `mdsmith_fix` without a separate
terminal window.

## Background

Claude Code supports local MCP (Model Context Protocol)
servers over stdio. A project or user can declare an MCP
server in its Claude Code settings and Claude will
automatically have access to its tools. This is the natural
integration point for mdsmith: ship an `mdsmith mcp`
command, add a project-level `mcpServers` config snippet,
and Claude Code gains lint-and-fix on every Markdown edit.

[Plan 121](121_vscode-integration.md) covers the VS Code
LSP integration. This plan is independent: it targets
Claude Code via MCP, not VS Code via LSP.

## Non-Goals

- New lint rules or new fix logic. MCP wires the existing
  pipeline through the protocol unchanged.
- LSP support (covered by plan 121).
- Remote or HTTP MCP transport — stdio only.

## Design

### Server: `mdsmith mcp`

A new subcommand `mdsmith mcp` runs an MCP server on stdio.
Implementation lives in `internal/mcp/`. The server speaks
JSON-RPC 2.0 line-framed over stdin/stdout (the MCP stdio
transport).

MCP messages handled:

| Message | Behavior |
|---------|----------|
| `initialize` | Return server info and tool list capability |
| `notifications/initialized` | No-op acknowledge |
| `tools/list` | Return the two tool descriptors |
| `tools/call` | Dispatch to `mdsmith_check` or `mdsmith_fix` |

### Tools

#### `mdsmith_check`

Input schema:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | `string` | yes | Markdown text to lint |
| `filename` | `string` | no | Logical filename (for config discovery and rule context) |
| `config` | `string` | no | Path to `.mdsmith.yml` override |

Returns a JSON array of diagnostics in the same shape as
`mdsmith check --format json`:

```json
[
  {
    "file": "README.md",
    "line": 10,
    "column": 81,
    "rule": "MDS001",
    "name": "line-length",
    "severity": "error",
    "message": "line too long (120 > 80)"
  }
]
```

An empty array means no issues. Errors (config parse
failure, etc.) surface as MCP tool error responses.

#### `mdsmith_fix`

Input schema:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | `string` | yes | Markdown text to fix |
| `filename` | `string` | no | Logical filename for config discovery |
| `config` | `string` | no | Path to `.mdsmith.yml` override |

Returns a JSON object:

```json
{
  "content": "<fixed markdown>",
  "changed": true,
  "remaining": []
}
```

`remaining` is the same diagnostic shape as `mdsmith_check`
and lists violations that could not be auto-fixed.

### Configuration discovery

The server resolves `.mdsmith.yml` for each tool call by
walking up from the directory of `filename` (or from the
working directory when `filename` is absent). The optional
`config` parameter overrides discovery, mirroring the
`-c` flag of the CLI.

### `internal/fix` extension

`Fixer.FixSource(path string, source []byte)` applies
the fix pipeline to in-memory bytes and returns the fixed
content plus remaining diagnostics without touching the
disk. This mirrors `engine.Runner.RunSource`.

### Claude Code integration

Users add mdsmith as an MCP server in `.claude/settings.json`
(project-level) or `~/.claude/settings.json` (global):

```json
{
  "mcpServers": {
    "mdsmith": {
      "type": "stdio",
      "command": "mdsmith",
      "args": ["mcp"]
    }
  }
}
```

The guide (`docs/guides/editors/claude-code.md`) documents
this snippet and explains how Claude can call the tools.

## Tasks

1. Add `Fixer.FixSource(path string, source []byte)` to
   [`internal/fix/fix.go`](../internal/fix/fix.go): apply
   the fix pipeline to in-memory bytes without disk I/O,
   returning fixed bytes and remaining diagnostics.
   Unit-test in `fix_source_test.go`.
2. Create `internal/mcp/` package:
   - `server.go` — JSON-RPC 2.0 line-framed reader/writer,
     `initialize`/`tools/list`/`tools/call` dispatch.
   - `tools.go` — `checkTool` and `fixTool` implementations
     that call `engine.Runner.RunSource` and
     `Fixer.FixSource`.
   - `server_test.go` — unit-test the full message round-trip
     with an in-memory pipe.
3. Create [`cmd/mdsmith/mcp.go`](../cmd/mdsmith/mcp.go)
   with `runMCP(args []string) int` that starts the MCP
   server on `os.Stdin`/`os.Stdout`.
4. Register `mcp` in `run()` dispatch and update
   `usageText` in
   [`cmd/mdsmith/main.go`](../cmd/mdsmith/main.go).
5. Add end-to-end test `cmd/mdsmith/mcp_e2e_test.go`:
   pipe `initialize` → `tools/list` → `tools/call
   mdsmith_check` against a file with a known violation,
   assert the diagnostic shape.
6. Create [`docs/reference/cli/mcp.md`](../docs/reference/cli/mcp.md).
7. Update the Commands table in
   [`docs/reference/cli.md`](../docs/reference/cli.md)
   (regenerated by `mdsmith fix`).
8. Create [`docs/guides/editors/claude-code.md`](../docs/guides/editors/claude-code.md)
   covering installation, the `mcpServers` config snippet,
   available tools, and example prompts.
9. Update the `docs/guides/index.md` catalog entry for the
   new guide (regenerated by `mdsmith fix`).
10. Update [`PLAN.md`](../PLAN.md) catalog
    (regenerated by `mdsmith fix`).

## Acceptance Criteria

- [ ] `mdsmith mcp --help` documents the subcommand;
      `usageText` lists `mcp`.
- [ ] `mdsmith mcp` survives a full `initialize` →
      `tools/list` → `tools/call` → `shutdown` round-trip
      in the end-to-end test.
- [ ] `tools/call mdsmith_check` with content containing
      a `MDS001` violation returns a diagnostic with
      `rule: "MDS001"` and the correct line number.
- [ ] `tools/call mdsmith_fix` returns `changed: true`
      and corrected content when given fixable violations;
      `remaining` lists non-fixable issues.
- [ ] `Fixer.FixSource` is unit-tested: input with
      fixable violations produces different output bytes
      and empty `remaining`; clean input is returned
      unchanged.
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports no issues.
- [ ] `mdsmith check .` passes including the new docs
      and the updated `PLAN.md` catalog.

## Open Questions

- **Workspace root.** The MCP server runs in whatever
  working directory the client spawns it from. When
  `filename` is an absolute path the server can walk up
  to find `.mdsmith.yml`; when it is relative or absent
  the server falls back to the working directory. This
  matches how `mdsmith check -` behaves today.
- **Streaming.** MCP 2025-03-26 adds streaming results.
  Out of scope for v1; plain `tools/call` response is
  sufficient and compatible with all current Claude Code
  versions.

## ...

<?allow-empty-section?>
