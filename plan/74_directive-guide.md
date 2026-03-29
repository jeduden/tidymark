---
id: 74
title: Directive guide
status: "🔲"
summary: >-
  Central guide for all directives and rules
  with examples, fixability table, placement
  rules, and nesting behavior.
---
# Directive guide

Part of the user-model work from
[plan 73](73_unify-template-directives.md).
Addresses
[#70](https://github.com/jeduden/mdsmith/issues/70).

## Goal

One guide a developer reads to understand every
directive and rule without consulting per-rule
READMEs.

## Context

Blind trials (plan 73) showed three gaps that
docs alone can close:

- 4-space indent silently breaks directives
  (confidence 2.6, no diagnostic emitted).
- Nested directives are undefined (confidence
  2.0, nobody could predict behavior).
- Users cannot predict which rules auto-fix
  (fix confidence 2-3 points lower than check).

## Rendering note

Processing instructions (`<?...?>`) are hidden
by GitHub's Markdown renderer (CommonMark
type-3 HTML blocks). Directives stay invisible
in rendered docs. Generated content between
markers is visible.

## Tasks

1. Create `docs/guides/directives.md` with:

  - Quick-reference table: name, purpose,
    closing tag (yes/no), fixable (yes/no),
    parameters
  - Two kinds of markers: closing tag means
    `fix` regenerates body; no closing tag means
    `check` validates a condition
  - Placement rules: max 3-space indent, not
    inside fenced code or HTML blocks
  - Explicit warning: 4 spaces turns directive
    into a code block with no error
  - Each directive section: purpose, all
    parameters, good example, bad example, what
    `check` reports, what `fix` does
  - Nesting: state that directives inside
    generated content are not re-processed
  - Fixability summary table for all 33 rules

2. Add a "see the directive guide" link from
   each rule README that uses a directive
   (MDS019, MDS020, MDS021, MDS030).
3. Run `mdsmith fix docs/guides/directives.md`
   to regenerate any catalog/include sections.
4. Run `mdsmith check docs/guides/` to verify.

## Acceptance Criteria

- [ ] `docs/guides/directives.md` exists
- [ ] Guide covers all four directives with
      examples
- [ ] Guide has fixability table for all 33
      rules
- [ ] Guide documents 4-space indent footgun
- [ ] Guide states nesting is not supported
- [ ] Guide passes `mdsmith check docs/guides/`
- [ ] Rule READMEs link to the guide
- [ ] All tests pass: `go test ./...`
- [ ] `go tool golangci-lint run` reports no
      issues
