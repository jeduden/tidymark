---
name: kinds
description: >-
  Inspect declared file kinds and resolve the effective
  rule config for a file by shelling out to
  `mdsmith kinds`. Use to answer "what rules apply to
  this Markdown file?" or to debug why a particular
  rule did or did not fire.
user-invocable: true
allowed-tools: >-
  Bash(mdsmith kinds:*), Bash(mdsmith kinds --:*),
  Bash(npx -y -p @mdsmith/cli mdsmith kinds:*)
argument-hint: "[path]"
---

Show the kinds assigned to a file (or all files) and
the merged rule config that results.

## When to invoke

Invoke when a user asks which kind a file belongs to,
which rules apply to it, or why a rule's settings
differ from the project defaults. The matching CLI
reference is
[`mdsmith kinds`](../../../../docs/reference/cli/kinds.md).

## Steps

### 1. Resolve the target path

If the user passed a path argument, use it. Otherwise
list kinds across the workspace from `.`.

Note the value as `$TARGET`.

### 2. Run mdsmith kinds

```bash
mdsmith kinds -- "$TARGET"
```

If the binary is not on `$PATH`, fall back to:

```bash
npx -y -p @mdsmith/cli mdsmith kinds -- "$TARGET"
```

### 3. Read the output

The command lists each file with its assigned kinds
and (when relevant) the merged rule keys that differ
from the defaults. Surface the relevant block back
to the user verbatim.

Non-zero exit usually means a config error or a
malformed kind-assignment — surface stderr so the
user sees the parse error.

## Notes

- Kind-assignment is order-sensitive; later entries
  layer on earlier ones via deep merge. See
  [file-kinds.md](../../../../docs/guides/file-kinds.md).
