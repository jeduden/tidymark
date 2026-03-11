---
id: 'int & >=1'
title: 'string & != ""'
status: '"🔲" | "🔳" | "✅"'
summary: 'string | *""'
---
# ?

<!-- Plan conventions:
  - Work test-driven: write a failing test, make it
    pass, commit.
  - Plan files must pass `mdsmith check plan/`.
  - Use Markdown links for real repo paths in prose.
    Bare backticked paths are allowed in commands,
    code blocks, and placeholders.
  - When a plan adds or updates examples in a rule
    README, require the use of <?include?> directives
    that reference fixture files (good/ and bad/
    directories, or good.md/bad.md). Never inline
    example content directly; include it from the
    tested fixture so documentation stays in sync
    with tests.
-->

## ...

<?allow-empty-section?>

## Goal

One-sentence summary of what this task achieves and why
it matters.

## ...

<?allow-empty-section?>

## Tasks

1. First concrete step
2. Second concrete step
3. ...

## ...

<?allow-empty-section?>

## Acceptance Criteria

- [ ] Criterion described as observable behavior
- [ ] Another criterion
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues

## ...

<?allow-empty-section?>
