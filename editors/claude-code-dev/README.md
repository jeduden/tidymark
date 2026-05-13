---
title: Claude Code dev plugin
summary: >-
  Contributor-only Claude Code plugin that wires
  Go (gopls) and TypeScript LSP servers so any
  agent working on the mdsmith codebase gets
  code intelligence across cmd/, internal/, and
  editors/vscode/.
---
# Claude Code dev plugin

This directory ships the `mdsmith-dev-lsp`
plugin. It is contributor-facing: it declares
language servers for the languages used inside
mdsmith itself, not the Markdown LSP that
end users get from the published
[`mdsmith-lsp` plugin](../claude-code/README.md).

Project settings auto-enable both
plugins. They come from the local
marketplace. The `LSP` tool then sees
these files in this repo:

- Go under `cmd/` and `internal/` via
  `gopls`.
- TypeScript under `editors/vscode/`
  via `typescript-language-server` over
  `npx`.
- Markdown via the sibling `mdsmith-lsp`
  plugin, served by the `@mdsmith/cli`
  npm package.

See `.claude/settings.json` and
`.claude-plugin/marketplace.json` for the
wiring.

## Prerequisites

- `gopls` on `$PATH`. Install with
  `go install golang.org/x/tools/gopls@latest`.
- A working `npx` (any recent Node.js). The
  TypeScript LSP is fetched lazily on first
  use; no global install needed.

## Why a separate plugin

The `mdsmith-lsp` plugin under
`editors/claude-code/` ships for end
users. They want Markdown intelligence.
Bundling `gopls` and
`typescript-language-server` into that
plugin would spawn extra processes for
every end user. Keeping the dev wiring
in its own plugin scopes the contributor
cost to contributors.
