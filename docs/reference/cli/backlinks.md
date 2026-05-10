---
command: backlinks
summary: List workspace links that point at a file.
---
# `mdsmith backlinks`

Print every workspace file that links to `<target>`,
one row per incoming link. The link graph is the same
one MDS027 (cross-file-reference-integrity) walks for
broken-link checking, queried in reverse.

```text
mdsmith backlinks [flags] <target>
```

`<target>` is workspace-relative. Append `#anchor` to
restrict matches to links whose anchor resolves to the
named heading.

## Flags

| Flag                | Default | Description                                         |
|---------------------|---------|-----------------------------------------------------|
| `-c`, `--config`    | auto    | Override config path                                |
| `-f`, `--format`    | `text`  | Output format: `text` or `json`                     |
| `--include GLOB`    | none    | Restrict sources to paths matching glob; repeatable |
| `--limit N`         | `0`     | Cap output at N rows (`0` = no cap)                 |
| `--no-gitignore`    | false   | Disable `.gitignore` filtering during walk          |
| `--follow-symlinks` | false   | Follow symlinks to regular files                    |
| `--max-input-size`  | `2MB`   | Max file size (e.g. `2MB`, `0`=none)                |

File discovery follows the `files:` patterns in
`.mdsmith.yml` and the same `ignore:` rules `check` and
`fix` use.

## Output

**text** (default):

```text
docs/index.md:14: [API reference](api.md)
docs/getting-started.md:42: [api docs](./api.md)
plan/045_api-overhaul.md:8: [api](../docs/api.md)
```

Each row carries the source path, the 1-based source
line, the visible link text, and the link target as it
appears in the source. Output is sorted by source path
then line, so `--limit` paginates deterministically.

**json**:

```json
[
  {
    "source": "docs/index.md",
    "line": 14,
    "text": "API reference",
    "target": "api.md"
  },
  {
    "source": "plan/045_api-overhaul.md",
    "line": 8,
    "text": "api",
    "target": "../docs/api.md"
  }
]
```

Keys are stable. Empty results emit `[]`, not `null`.

## Examples

List everything that points at `docs/api.md`:

```bash
mdsmith backlinks docs/api.md
```

Filter to a specific anchor on the target:

```bash
mdsmith backlinks docs/api.md#authentication
```

Limit to the plan directory and the first ten rows:

```bash
mdsmith backlinks --include "plan/**" --limit 10 docs/api.md
```

## Scope

The command resolves the same direct Markdown links
MDS027 sees. The graph covers `[text](path)` and
`[text](path#anchor)`. Reference-style links
(`[text][label]`), wiki-style links (`[[page]]`), and
external URLs are out of scope. They would only join
the result set once the matching parser support lands.

## Exit codes

| Code | Meaning                 |
|------|-------------------------|
| 0    | At least one match      |
| 1    | No incoming links found |
| 2    | Runtime/parse error     |
