---
title: mdsmith for Claude Code
summary: >-
  Install the mdsmith Claude Code plugin so the agent
  receives Markdown diagnostics and navigation through
  the mdsmith LSP server.
---
# mdsmith for Claude Code

Inline mdsmith diagnostics and Markdown navigation
for Claude Code, wired through `mdsmith lsp`.

## Install

```text
/plugin marketplace add jeduden/mdsmith
/plugin install mdsmith-lsp@mdsmith
/reload-plugins
```

## Prerequisite

The plugin spawns this command:

```bash
npx -y -p @mdsmith/cli mdsmith lsp
```

`npx` ships with Node.js, which Claude Code
already requires. First launch downloads
`@mdsmith/cli` and the platform binary subpackage
from npm; later launches reuse the npm cache.

To pin a specific version, edit the plugin
manifest's `args` to read
`@mdsmith/cli@<ver>` instead of `@mdsmith/cli`.

## Troubleshooting

If the `/plugin` Errors tab shows `Executable not
found in $PATH`, Node.js is missing from the shell
`$PATH` Claude Code sees. Install Node 18 or later
(20 LTS recommended), then run `/reload-plugins`.
