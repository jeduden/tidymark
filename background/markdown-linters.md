# Markdown Linters Comparison

This document compares mdsmith with popular Markdown
linting and formatting tools. It also covers the emerging
use of LLMs as linters.

## Tool Overview

### mdsmith

Go binary. 30 rules (MDS001-MDS030) covering structural
linting, readability metrics, and generated-content
management. Single binary with zero runtime dependencies.

Key differentiators:

- Token-budget rule (MDS028) targeting LLM context windows
- Paragraph readability (ARI grade) and structure limits
- Regenerable sections: catalog, include, required-structure
- Git merge driver for auto-resolving generated sections
- Metrics subsystem (bytes, lines, words, headings, tokens)
- Offline rule docs compiled into the binary

### markdownlint (markdownlint-cli2)

Node.js. ~60 built-in rules (MD001-MD058), 31 auto-fixable.
The most widely adopted Markdown linter. GitHub uses it
internally via markdownlint-github.

- Config: JSONC, YAML, or JS
- Official VS Code extension and GitHub Action
- Custom rules via npm packages
- Mature ecosystem with Prettier compatibility presets
- ~5.9k GitHub stars (library)

### remark-lint

Node.js. ~70 rules distributed as individual npm packages.
Part of the unified/remark AST pipeline ecosystem.

- Architecture: parse to mdast AST, lint, serialize
- Three maintained presets (consistent, recommended, style)
- "Fix" by round-tripping through AST (reformat entire file)
- Powers MDX, Gatsby, and Next.js documentation
- Deep composability with unified plugins
- ~7k stars (remark parent project)

### Prettier

Node.js. Opinionated formatter, not a linter. ~51.6k stars.

- Reformats Markdown with zero config
- Formats embedded code blocks (JS, TS, CSS, JSON, etc.)
- Key option: `proseWrap` (preserve, always, never)
- No diagnostics, no structural checks, no rule toggles
- Best paired with a linter for structural validation

### Vale

Go binary. Prose and style linter, not a structural linter.
Checks writing quality against style guides (Microsoft,
Google, AP, custom). ~5.2k stars.

- 11 extension points (existence, substitution, metric, etc.)
- Styles are YAML rule files; no code required
- Config: `.vale.ini` (INI format)
- Markup-aware: Markdown, HTML, RST, AsciiDoc, DITA, XML
- Used by Grafana, GitLab, DigitalOcean for docs
- Official GitHub Action and VS Code extension

### textlint

Node.js. Zero built-in rules, fully pluggable architecture
modeled after ESLint. ~3k stars.

- 100+ community rules available via npm
- Parser plugins for Markdown, HTML, RST, AsciiDoc, Typst
- Autofix via `--fix` for rules implementing the fixer API
- MCP server support (v15.2+) for AI assistant integration
- Strong Japanese language support

### LLM as Linter

Using language models (GPT-4, Claude, etc.) directly to
check prose quality, conciseness, and style. This is
emerging through dedicated CLI tools and AI review bots.

How it works:

- Send Markdown to an LLM with a style prompt
- LLM returns diagnostics: verbose paragraphs, unclear
  phrasing, jargon, redundancy
- Tools wrap this in CLI or CI workflows

Dedicated tools:

- **VectorLint**: CLI AI prose linter. Rules defined in
  natural language in a `VECTORLINT.md` file. Uses
  error-density scoring and 1-4 rubrics. Supports
  OpenAI, Anthropic, and Google providers.
- **GPTLint**: two-pass LLM linter. A cheap model finds
  candidates, a strong model filters false positives.
  Uses GritQL to pre-filter files. Cost: ~$0.83 for
  351 API calls on a small codebase.

AI review services:

- GitHub Copilot code review (reviews Markdown in PRs)
- CodeRabbit (combines 40+ linters with LLM review)
- Claude Code Action (Anthropic's PR review action)

Hybrid systems:

- Grammarly: rule-based grammar + ML models + LLMs.
  Their CoEdIT model (770M-11B params) beats GPT-3
  at text editing while being 60x smaller.

Strengths:

- Excels at subjective quality: conciseness, clarity
- Catches semantic issues no rule-based tool can detect
- A single natural-language instruction replaces many
  regex patterns (e.g. "flag hedging language")
- Understands context and intent, not just patterns

Weaknesses:

- Non-deterministic: same input may yield different
  output, even at temperature=0
- Costly: API calls per file per run add up
- Slow: seconds per file vs milliseconds for rules
- Requires network (local LLMs work but lower quality)
- Hard to use as a blocking CI gate reliably

## Feature Comparison

<!-- allow-empty-section -->

### Structural Linting

| Capability          | mdsmith   | markdownlint | remark-lint           |
|---------------------|-----------|--------------|-----------------------|
| Heading hierarchy   | MDS003    | MD001        | heading-increment     |
| First-line heading  | MDS004    | MD041        | first-heading-level   |
| Duplicate headings  | MDS005    | MD024        | no-duplicate-headings |
| Blank line spacing  | MDS013-15 | MD022,25,31  | plugins               |
| List indentation    | MDS016    | MD007        | list-item-indent      |
| Code fence style    | MDS010    | MD048        | fenced-code-flag      |
| Code block language | MDS011    | MD040        | fenced-code-flag      |
| Bare URLs           | MDS012    | MD034        | no-literal-urls       |
| Line length         | MDS001    | MD013        | max-line-length       |
| Trailing spaces     | MDS006    | MD009        | no-trailing-spaces    |

All three cover core structural rules. markdownlint has
the broadest rule set (~60 rules). mdsmith and remark-lint
are comparable in structural coverage.

### Prose and Readability

| Capability        | mdsmith          | Vale             | LLM |
|-------------------|------------------|------------------|-----|
| Readability grade | MDS023 (ARI)     | metric ext       | yes |
| Sentence limits   | MDS024           | occurrence ext   | yes |
| Word choice       | no               | substitution ext | yes |
| Passive voice     | no               | existence ext    | yes |
| Jargon detection  | no               | existence ext    | yes |
| Conciseness       | MDS029 (planned) | no               | yes |
| Tone enforcement  | no               | custom styles    | yes |
| Token budget      | MDS028           | no               | no  |
| Deterministic     | yes              | yes              | no  |

mdsmith focuses on measurable readability metrics (ARI
grade, sentence count, token budget). Vale excels at style
guide enforcement. LLMs handle subjective quality best but
lack determinism.

### Formatting and Fixing

| Capability         | mdsmith          | Prettier       | markdownlint |
|--------------------|------------------|----------------|--------------|
| Autofix CLI        | `fix`              | `--write`        | `--fix`        |
| Table alignment    | MDS025           | yes            | no           |
| Prose wrapping     | no               | `proseWrap`      | no           |
| Embedded code fmt  | no               | JS/TS/CSS/JSON | no           |
| Multi-pass fix     | yes              | single pass    | single pass  |
| Generated sections | catalog, include | no             | no           |

Prettier is the strongest pure formatter. mdsmith has
unique autofix for generated content (catalog, include).
markdownlint fixes structural violations.

### Cross-File and Project Features

| Capability           | mdsmith | markdownlint | remark-lint           |
|----------------------|---------|--------------|-----------------------|
| Link integrity       | MDS027  | no           | remark-validate-links |
| Include sections     | MDS021  | no           | no                    |
| Catalog generation   | MDS019  | no           | no                    |
| Required structure   | MDS020  | no           | no                    |
| Git merge driver     | yes     | no           | no                    |
| Metrics ranking      | yes     | no           | no                    |
| Gitignore aware      | yes     | yes          | no                    |
| Front matter support | yes     | via plugin   | via plugin            |

mdsmith has the strongest cross-file and project-level
features. The merge driver and regenerable sections are
unique to mdsmith.

### Runtime and Integration

| Property       | mdsmith    | markdownlint   | remark-lint  | Prettier     | Vale       | textlint     | LLM         |
|----------------|------------|----------------|--------------|--------------|------------|--------------|-------------|
| Language       | Go         | Node.js        | Node.js      | Node.js      | Go         | Node.js      | API         |
| Runtime deps   | none       | Node 20+       | Node 16+     | Node 16+     | none       | Node 20+     | network     |
| Install        | binary     | npm            | npm          | npm          | binary     | npm          | varies      |
| Config format  | YAML       | JSONC/YAML/JS  | JSON/YAML/JS | JSON/YAML/JS | INI+YAML   | JSON/YAML/JS | prompt      |
| Output formats | text, JSON | text, JSON     | text         | none         | text, JSON | text, JSON   | text        |
| VS Code        | no         | yes            | yes          | yes          | yes        | yes          | varies      |
| GitHub Action  | no         | yes            | via npm      | via npm      | yes        | via npm      | custom      |
| Pre-commit     | lefthook   | husky/lefthook | husky        | husky        | hooks      | husky        | impractical |
| Offline        | yes        | yes            | yes          | yes          | yes        | yes          | no          |
| Deterministic  | yes        | yes            | yes          | yes          | yes        | yes          | no          |

Go-based tools (mdsmith, Vale) have zero runtime
dependencies. Node.js tools require a runtime but benefit
from the npm ecosystem. LLM-based linting requires network
access and is non-deterministic.

## When to Use What

**mdsmith** fits best when you need readability limits,
token budgets, or generated content sections. Its single
binary makes CI setup simple.

**markdownlint** is the safe default for teams already in the
Node.js ecosystem. Widest community adoption, most editor
integrations, battle-tested rule set.

**remark-lint** suits projects deep in the unified/remark
ecosystem (MDX, Gatsby, Next.js). Its AST pipeline enables
custom transformations beyond linting.

**Prettier** is a formatter, not a linter. Use it alongside a
linter. Pair with markdownlint (using the Prettier compat
preset) or remark-lint for structural checks.

**Vale** is the right choice for enforcing prose style guides
(Microsoft, Google, AP). It complements structural linters
rather than replacing them.

**textlint** works well for polyglot text linting (especially
Japanese) and teams wanting ESLint-style modularity.

**LLM as linter** is best for subjective quality checks:
conciseness, clarity, tone. Use it in PR review workflows
where latency and cost are acceptable. Pair with a
deterministic linter for structural rules.

## Combining Tools

Most teams benefit from layering tools. Common pairings:

- **Structure + format**: markdownlint + Prettier
- **Structure + prose**: mdsmith + Vale
- **Structure + AI review**: mdsmith + LLM review in CI
- **Full stack**: mdsmith (structure + readability) + Vale
  (style) + Prettier (formatting)

mdsmith's conciseness-scoring rule (MDS029, planned) brings
LLM-grade quality checks into an offline, rule-based tool.
It uses classifier-backed scoring to bridge the gap between
static rules and LLM review.
