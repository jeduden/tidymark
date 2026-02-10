# Code-Block Awareness for Line-Based Rules

## Goal

Fix TM006 (no-trailing-spaces), TM007 (no-hard-tabs), and TM008
(no-multiple-blanks) so they skip lines inside fenced and indented code blocks,
eliminating false positives on code block content.

## Prerequisites

None — this plan can start immediately.

## Tasks

1. Extract `collectCodeBlockLines()` and its helpers (`addFencedCodeBlockLines`,
   `findFencedOpenLine`, `addBlockLines`) from
   `internal/rules/linelength/rule.go` into a shared helper at
   `internal/lint/codeblocks.go`. The function signature should be
   `CollectCodeBlockLines(f *File) map[int]bool` (exported, on `*lint.File`).

2. Update `internal/rules/linelength/rule.go` to call the new shared
   `lint.CollectCodeBlockLines(f)` instead of its local
   `collectCodeBlockLines(f)`. Remove the now-duplicated local functions.

3. Update `internal/rules/notrailingspaces/rule.go` (TM006): before
   iterating `f.Lines`, call `lint.CollectCodeBlockLines(f)` and skip
   lines whose 1-based line number is in the returned set. Apply the
   same skip in both `Check` and `Fix`.

4. Update `internal/rules/nohardtabs/rule.go` (TM007): same approach —
   skip code block lines in both `Check` and `Fix`.

5. Update `internal/rules/nomultipleblanks/rule.go` (TM008): same
   approach — skip code block lines in both `Check` and `Fix`. This
   fixes the known false positive where blank lines between code blocks
   in a fenced section trigger TM008.

6. Add unit tests in `internal/lint/codeblocks_test.go` verifying the
   helper correctly identifies fenced code block lines (including
   opening/closing fence lines), indented code block lines, and returns
   empty for documents with no code blocks.

7. Add test fixtures for each rule:
   - `rules/TM006-no-trailing-spaces/good.md`: ensure trailing spaces
     inside a fenced code block do NOT fire TM006
   - `rules/TM007-no-hard-tabs/good.md`: ensure tabs inside a fenced
     code block do NOT fire TM007
   - `rules/TM008-no-multiple-blanks/good.md`: ensure consecutive blank
     lines inside a fenced code block do NOT fire TM008
   - Add corresponding `bad.md` cases that confirm violations OUTSIDE
     code blocks still fire

8. Verify TM008 README example (`rules/TM008-no-multiple-blanks/README.md`)
   no longer triggers a false positive on its fenced code block content.

## Acceptance Criteria

- [ ] `lint.CollectCodeBlockLines` exists in `internal/lint/codeblocks.go`
- [ ] TM001 still uses the shared helper (not a local copy)
- [ ] TM006 Check/Fix skip code block lines
- [ ] TM007 Check/Fix skip code block lines
- [ ] TM008 Check/Fix skip code block lines
- [ ] Unit tests for the shared helper pass
- [ ] Fixture tests for code-block-awareness pass
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
