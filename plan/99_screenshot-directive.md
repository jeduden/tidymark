---
id: 99
title: Capture directive and extensible builder
status: "🔲"
summary: >-
  A generic `<?capture type:...?>` directive
  keeps a Markdown image link in sync with its
  params via the lint layer. Built-in types
  cover `screenshot` (chromedp) and `vhs`
  (GIF). Users declare custom types in
  `.mdsmith.yml` with a command template and
  param schema; no Go code needed.
---
# Capture directive and extensible builder

## Goal

Make tool-generated files — screenshots, GIFs,
diagrams — declared artifacts of the source
that produces them. Authors state `type`,
inputs, and `output`. `mdsmith fix` keeps the
body in sync. `mdsmith capture` drives the
rebuild.

The built-in types are `screenshot` (headless
browser) and `vhs` (terminal GIF). Users
extend the system in `.mdsmith.yml` with a
command template and a param schema. No Go
code is required for new types.

## Context

[interblah.net/self-updating-screenshots][post]
describes a Rake task that scans Markdown for
`<!-- SCREENSHOT: ... -->` comments and drives
Capybara to capture PNGs. The biggest win is
killing drift: once capture is automatic,
images stop diverging from the UI.

This repo has the same problem twice: browser
screenshots for feature docs, and `demo.gif`
driven by VHS from `demo.tape`. Both captures
are currently manual.

The [generated-section archetype][gensection]
already handles body sync. Two pieces are
missing: a directive with a typed capture
layer, and a separate rebuild command that
users can extend without writing Go code.

[post]: https://interblah.net/self-updating-screenshots
[gensection]: ../docs/background/archetypes/generated-section/README.md

### Why split linting from capture

Capture needs a running target, an external
tool, and time. Folding it into `mdsmith fix`
would make the linter slow and stateful.

- **Lint layer (always on).** MDS039 validates
  params and keeps `![alt](output)` in sync.
  `mdsmith fix` rewrites the body. No network,
  no external tools.
- **Capture layer (opt-in).** `mdsmith capture`
  dispatches to the per-type driver. It is
  never a side effect of `check`/`fix`.

## Design

### Directive syntax

```text
<?capture
type: screenshot
url: /inbox
output: docs/inbox.png
alt: Inbox screenshot
?>
![Inbox screenshot](docs/inbox.png)
<?/capture?>
```

Common parameters (all types):

| Name     | Required | Description                                  |
|----------|----------|----------------------------------------------|
| `type`   | yes      | Registered type name                         |
| `output` | yes      | Artifact path relative to the Markdown file  |
| `alt`    | no       | Alt text. Derived from type+output if absent |

Type-specific parameters are declared by each
type (built-in or user-defined in config).

### Built-in types

**`screenshot`** captures a URL via
[chromedp][chromedp]:

| Param      | Required | Default    | Description                     |
|------------|----------|------------|---------------------------------|
| `url`      | yes      | —          | Target URL                      |
| `selector` | no       | full page  | CSS element to capture          |
| `viewport` | no       | `1280x800` | Browser size (`WIDTHxHEIGHT`)   |
| `wait`     | no       | `0` ms     | Delay after navigation          |
| `click`    | no       | —          | CSS selector to click first     |
| `hide`     | no       | `[]`       | Selectors hidden before capture |

**`vhs`** renders a `.tape` file via the
[VHS][vhs] binary:

| Param   | Required | Description          |
|---------|----------|----------------------|
| `input` | yes      | Path to `.tape` file |

[chromedp]: https://github.com/chromedp/chromedp
[vhs]: https://github.com/charmbracelet/vhs

### Custom types

Users declare new types in `.mdsmith.yml`:

```yaml
capture:
  types:
    mermaid:
      command: "mmdc -i {input} -o {output}"
      params:
        required: [input, output]
        optional: [theme]
```

`{param}` tokens expand to directive parameter
values at capture time. MDS039 reads the schema
and validates that `required` params are present.

### Generated body (lint layer)

The body is exactly one line:

```text
![{alt}]({output})
```

MDS039 compares this to the actual body and
reports a stale-section diagnostic on
divergence. `Fix` rewrites the body. It reuses
`gensection.Engine` unchanged — same plumbing
as `catalog` and `include`.

When `alt` is absent, it is derived as
`"Screenshot of {selector or "page"} at {url}"`
so the body always passes MDS032.

### Rule: MDS039 (capture)

- ID: `MDS039`
- Name: `capture`
- Category: `meta`
- Default: enabled
- Fixable: yes (lint layer only)

Validation:

- `type` is present and resolves to a known
  type (built-in or declared in config)
- `output` is a relative path inside the
  project root, no `..`
- Type-specific required params are present
- `output` extension is in the type's allowed
  set (`{.png,.jpg,.webp}` for `screenshot`,
  `.gif` for `vhs`, unrestricted for custom)

### `mdsmith capture` subcommand

```text
mdsmith capture [paths...] [flags]
```

Flags:

- `--type NAME` — limit to one type
- `--base-url URL` — prefix for path-only URLs
  (`screenshot` type)
- `--dry-run` — list captures; write nothing
- `--timeout DURATION` — per-artifact timeout
  (default `30s`)

Behaviour:

1. Walk files the same way `check`/`fix` do.
2. Collect and validate `<?capture?>` blocks.
3. Dispatch each block to its type driver in
   declaration order.
4. Write artifacts atomically to `output`.
5. Exit non-zero on any failure and print a
   per-file `OK | FAIL` summary.

### Configuration

```yaml
capture:
  base-url: ""       # joined to path-only URLs
  viewport: 1280x800 # default for screenshot
  wait: 0            # default delay (ms)
  types: {}          # custom type declarations
```

Per-directive params override config defaults.
The `--base-url` CLI flag overrides both.

### Interaction with existing rules

- **MDS032**: derived alt text is non-empty.
- **MDS012**: image link is not a bare URL.
- **MDS027**: a missing artifact file fires
  MDS027 and points the user at
  `mdsmith capture`. No special case needed.
- **`merge-driver`**: regenerates the body on
  conflict. Artifact bytes are not regenerated.

## Tasks

1. Define a `Capturer` interface and type
   registry in `internal/capture/`. Built-in
   impls: `screenshot` (chromedp) and `vhs`
   (exec). Custom-type impl reads the command
   template from config and shells out via
   `os/exec`.
2. Create the `<?capture?>` directive in
   `internal/rules/capture/` using
   `gensection.Engine`. Register as MDS039,
   category `meta`. `Generate` renders only the
   image line; it never calls the capturer.
3. Add MDS039 fixtures under
   `internal/rules/MDS039-capture/`:
   `good/`, `bad/`, `fixed/`. Cover built-in
   types and a custom type declared in config.
4. Wire the rule into `cmd/mdsmith/main.go`.
5. Add `mdsmith capture` subcommand. Reuse the
   file walker and AST parser. Wire all flags.
6. Add the `capture:` config block in
   `internal/config/`. Surface defaults and the
   custom-type registry to MDS039 and the
   capture command.
7. Integration test: `<?capture type:screenshot?>`
   against `httptest.Server` writes a non-empty
   PNG. A custom type using `cp` writes a stub
   file. Both skip without their prerequisite.
8. Document MDS039 README and a user guide at
   `docs/guides/directives/capture.md`. Update
   the gensection archetype doc.
9. Add a demo using a static HTML file so it
   runs without a dev server.

## Acceptance Criteria

- [ ] `<?capture type:screenshot url:... output:... ?>`
      regenerates its body to `![alt](output)`
      on `mdsmith fix`
- [ ] MDS039 reports stale-section on `check`
      when the body diverges from params
- [ ] MDS039 rejects unknown type, missing
      `output`, path traversal (`..`), and
      extensions that do not match the type
- [ ] A custom type declared in config with a
      `cp` command writes its output file
- [ ] `mdsmith capture` against `httptest.Server`
      writes a non-empty PNG
- [ ] `mdsmith capture --dry-run` lists every
      capture without running any tool
- [ ] `mdsmith capture` exits non-zero on
      failure with a per-file `OK | FAIL` summary
- [ ] `mdsmith check` does **not** launch a
      browser or external tool for
      `<?capture?>` blocks
- [ ] `capture:` config defaults apply to
      directives that omit them and are
      overridden by per-directive params
- [ ] CI without chromium still passes; capture
      tests are skipped, not failed
- [ ] Merge driver regenerates `<?capture?>`
      bodies on conflict (gensection engine)
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports
      no issues
