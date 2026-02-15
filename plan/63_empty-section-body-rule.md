---
id: 63
title: Empty Section Body Rule
status: 🔲
---
# Empty Section Body Rule

## Goal

Add a rule that reports headings whose section body is effectively empty,
so docs do not ship with placeholder or structure-only sections.

## Tasks

1. Define section-body semantics:
   what counts as meaningful content vs ignorable content between a heading
   and the next heading of same or higher level.
2. Review prior-art implementations in other linters and tools
   (for example `markdownlint`, `remark-lint`, Vale, and textlint):
   detect available rules, behavior differences, and known edge cases to
   avoid reinventing weak heuristics.
3. Define scope and exemptions:
   heading levels to check, and any allowed exceptions
   (for example explicit template placeholders).
4. Decide rule shape:
   new rule ID/name, category, default enablement, and configuration fields.
5. Implement heading-section analysis in the rule package,
   including stable line mapping for diagnostics.
6. Add unit tests for positive and negative cases:
   empty sections, comment-only sections, code/list/table content,
   nested headings, and edge boundaries at EOF.
7. Add rule docs and fixtures under `internal/rules/MDS030-*`,
   including config examples and diagnostic messages.
8. Integrate and verify in CLI/integration tests where needed.

## Acceptance Criteria

- [ ] Rule reports headings that have no meaningful body content before the
      next same/higher-level heading.
- [ ] Rule does not report sections with meaningful list, table, code, or
      paragraph content.
- [ ] Prior-art behavior from other linters is summarized and key trade-offs
      are reflected in rule semantics or documented deviations.
- [ ] Scope and exemptions are documented with examples.
- [ ] Diagnostics include heading text, location, and clear remediation
      guidance.
- [ ] Unit tests and fixtures cover representative edge cases.
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
