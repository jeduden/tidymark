---
id: 86
title: Markdown flavor validation
status: "🔲"
summary: >-
  New rule MDS034 that validates Markdown files against
  a declared flavor (CommonMark, GFM, Goldmark, etc.)
  and reports unsupported syntax with auto-fix where
  possible.
---
# Markdown flavor validation

## Goal

Let users declare a target Markdown flavor so mdsmith
can flag syntax that the target renderer will not
understand, preventing silent rendering failures when
files move between tools.

## Background

### Common Markdown flavors

Different renderers support different supersets of
Markdown. The most common flavors, ordered roughly by
adoption:

| Flavor | Base | Key extensions | Used by |
|---|---|---|---|
| CommonMark | --- | strict baseline spec | many renderers |
| GFM | CommonMark | tables, task lists, strikethrough, autolinks | GitHub, VS Code |
| Goldmark (default) | CommonMark | tables, strikethrough, task lists, linkify | Hugo, mdsmith |
| PHP Markdown Extra | original | footnotes, abbreviations, definition lists, fenced code, tables, heading IDs | WordPress plugins |
| Pandoc | CommonMark | footnotes, definition lists, math, citations, superscript, subscript, heading IDs | academic, Pandoc |
| MyST | CommonMark | directives, roles, cross-references, math | Sphinx, Jupyter Book |
| MultiMarkdown | original | metadata, footnotes, citations, math, tables, cross-references | MultiMarkdown app |
| GitLab | GFM | math blocks, diagrams (Mermaid), footnotes, inline diff | GitLab |
| R Markdown | Pandoc | code chunks, inline R, output formats | RStudio, knitr |

### Syntax differences that matter

These are the concrete features that differ across
flavors and can be detected at the AST or text level:

| Feature | CommonMark | GFM | Goldmark | Pandoc | PHP Extra |
|---|---|---|---|---|---|
| Tables | no | yes | yes | yes | yes |
| Task lists | no | yes | yes | no | no |
| Strikethrough `~~x~~` | no | yes | yes | yes | no |
| Autolinks (bare URLs) | no | yes | yes | no | no |
| Footnotes `[^1]` | no | no | ext | yes | yes |
| Definition lists | no | no | no | yes | yes |
| Math `$x$` / `$$x$$` | no | partial | ext | yes | no |
| Heading IDs `{#id}` | no | no | ext | yes | yes |
| Abbreviations | no | no | no | no | yes |
| Superscript `^x^` | no | no | no | yes | no |
| Subscript `~x~` | no | no | no | yes | no |
| Pipe tables only | --- | yes | yes | yes | yes |
| Grid tables | no | no | no | yes | no |

"ext" means available as a Goldmark extension but not
enabled by default.

### Detection approach

mdsmith already parses with Goldmark. The strategy is:

1. **AST-based detection**: Parse with all extensions
   enabled. Walk the AST and flag node types that the
   target flavor does not support (tables, task lists,
   strikethrough).
2. **Regex-based detection**: Some features (footnote
   references, definition lists, math delimiters,
   heading IDs, abbreviations) may not produce AST
   nodes if the extension is not loaded. Detect these
   via line-level regex patterns on the raw source.
3. **Combined**: Use AST when the extension is loaded,
   fall back to regex for extensions Goldmark does not
   support.

### Scoping: which flavors to support initially

Start with the three most common targets:

- **commonmark** -- strict CommonMark spec
- **gfm** -- GitHub Flavored Markdown
- **goldmark** -- Goldmark with default extensions

These cover the vast majority of use cases. Additional
flavors (Pandoc, PHP Extra, MyST) can be added later
by extending the feature table.

## Design

### New rule: MDS034 `markdown-flavor`

Category: `meta`. Enabled by default: no (opt-in).

#### Configuration

```yaml
rules:
  markdown-flavor:
    flavor: gfm  # commonmark | gfm | goldmark
    # Future: allow per-feature overrides
    # allow:
    #   - footnotes
    # deny:
    #   - autolinks
```

#### Flavor feature registry

Internal data structure mapping each flavor to its
supported feature set:

```go
// internal/rules/markdownflavor/features.go

type Feature string

const (
    Tables        Feature = "tables"
    TaskLists     Feature = "task-lists"
    Strikethrough Feature = "strikethrough"
    Autolinks     Feature = "autolinks"
    Footnotes     Feature = "footnotes"
    DefinitionLists Feature = "definition-lists"
    MathInline    Feature = "math-inline"
    MathBlock     Feature = "math-block"
    HeadingIDs    Feature = "heading-ids"
    Abbreviations Feature = "abbreviations"
    Superscript   Feature = "superscript"
    Subscript     Feature = "subscript"
)

type Flavor struct {
    Name     string
    Features map[Feature]bool
}
```

Each flavor is a set of booleans. Checking is: walk
the document, identify which features are used, report
any feature where `flavor.Features[f] == false`.

#### Detection methods

Each feature needs a detector. Two kinds:

1. **AST detector**: Walks parsed AST nodes.
   - Tables: `ast.KindTable` (from goldmark extension)
   - Task lists: check `ast.ListItem` with
     `HasChildren` of type checkbox
   - Strikethrough: `extension.KindStrikethrough`

2. **Regex detector**: Scans raw source lines.
   - Footnotes: `^\[\^[^\]]+\]:` (definition) and
     `\[\^[^\]]+\]` (reference)
   - Definition lists: line starting with `:` after
     a term line
   - Math: `\$[^$]+\$` (inline), `^\$\$$` (block)
   - Heading IDs: `\{#[\w-]+\}` at end of heading
   - Abbreviations: `^\*\[[^\]]+\]:`
   - Superscript: `\^[^^]+\^`
   - Subscript: `~[^~]+~` (careful not to conflict
     with strikethrough `~~`)

#### Error messages

Messages follow existing mdsmith conventions: state
the concrete constraint, name the feature.

| Feature | Message |
|---|---|
| Tables | `table syntax is not supported by {flavor}; convert to a list or use a supported renderer` |
| Task lists | `task list (checkbox) syntax is not supported by {flavor}` |
| Strikethrough | `strikethrough (~~text~~) is not supported by {flavor}` |
| Autolinks | `bare URL autolink is not supported by {flavor}; wrap in angle brackets or use [text](url)` |
| Footnotes | `footnote syntax is not supported by {flavor}` |
| Definition lists | `definition list syntax is not supported by {flavor}` |
| Math (inline) | `inline math ($...$) is not supported by {flavor}` |
| Math (block) | `block math ($$...$$) is not supported by {flavor}` |
| Heading IDs | `heading ID attribute ({#id}) is not supported by {flavor}` |
| Abbreviations | `abbreviation definition is not supported by {flavor}` |
| Superscript | `superscript (^text^) is not supported by {flavor}` |
| Subscript | `subscript (~text~) is not supported by {flavor}` |

All messages use severity `warning` (the syntax may
still render as plain text, it just won't render as
intended).

#### Auto-fix

Some features have natural downgrades; others do not.

| Feature | Fixable | Fix strategy |
|---|---|---|
| Autolinks | yes | Wrap bare URL in `<url>` or `[url](url)` |
| Strikethrough | partial | Remove `~~` delimiters, leaving plain text |
| Heading IDs | yes | Remove `{#id}` suffix from heading line |
| Tables | no | No lossless downgrade exists |
| Task lists | partial | Replace `- [ ]` / `- [x]` with `- ` / `- (done) ` |
| Footnotes | no | Requires restructuring (inline the note) -- too complex for auto-fix |
| Definition lists | no | Requires restructuring |
| Math | no | No Markdown-only fallback |
| Abbreviations | no | Requires expanding all uses |
| Superscript | partial | Remove `^` delimiters |
| Subscript | partial | Remove `~` delimiters |

The rule implements `FixableRule` for the fixable
subset. Non-fixable features produce diagnostics only.

### Integration with existing rules

- **MDS012 (no-bare-urls)** already flags bare URLs.
  When `markdown-flavor: { flavor: gfm }` is set,
  autolinks are valid GFM. The flavor rule should not
  conflict: MDS012 is about style preference, MDS034
  is about spec compliance. Document the interaction
  in the rule README.
- **MDS002 (heading-style)** is orthogonal -- both ATX
  and setext are valid in all flavors.

## Tasks

1. Add feature enum and flavor registry in
   `internal/rules/markdownflavor/features.go`
2. Add AST-based detectors for tables, task lists,
   strikethrough, autolinks
3. Add regex-based detectors for footnotes, definition
   lists, math, heading IDs, abbreviations,
   superscript, subscript
4. Implement `rule.go` with `Check()` that runs
   detectors and reports unsupported features
5. Implement `Fix()` for fixable features (autolinks,
   strikethrough, heading IDs, task lists)
6. Add `Configurable` interface: `ApplySettings` for
   `flavor` string, validate against known flavors
7. Register rule as MDS034 `markdown-flavor` in
   category `meta`
8. Add test fixtures:
   `internal/rules/MDS034-markdown-flavor/bad/` with
   files using each unsupported feature per flavor,
   `good/` with files using only supported features,
   `fixed/` with expected auto-fix output
9. Add rule README following
   `internal/rules/proto.md` schema
10. Document the rule in `docs/reference/cli.md` and
    add a note to
    `docs/guides/directives/enforcing-structure.md`
    about flavor enforcement

## Acceptance Criteria

- [ ] `markdown-flavor: { flavor: commonmark }` flags
      tables, task lists, strikethrough, and autolinks
- [ ] `markdown-flavor: { flavor: gfm }` accepts
      tables, task lists, strikethrough, autolinks but
      flags footnotes, definition lists, math
- [ ] `markdown-flavor: { flavor: goldmark }` accepts
      Goldmark default extensions, flags Pandoc-only
      features
- [ ] Error messages name the specific unsupported
      feature and the configured flavor
- [ ] `mdsmith fix` auto-fixes autolinks (wraps in
      angle brackets) and heading IDs (removes `{#id}`)
- [ ] `mdsmith fix` auto-fixes strikethrough and task
      lists with plain-text fallbacks
- [ ] Non-fixable features (tables, footnotes, math)
      produce diagnostics only, no fix attempted
- [ ] Invalid flavor name in config produces a clear
      configuration error
- [ ] Rule is disabled by default (opt-in)
- [ ] Rule does not conflict with MDS012
      (no-bare-urls) when both are enabled
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports no issues
