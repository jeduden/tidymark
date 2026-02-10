# Fixture Cleanup: good.md and fixed.md Must Pass All Rules

## Goal

Update the integration test framework so `good.md` and `fixed.md` fixture files
must produce zero diagnostics from ALL rules (not just their own rule), then
audit and fix every fixture that violates any rule.

## Prerequisites

- Plan 13 (code-block-awareness) — needed so TM006/TM007/TM008 don't fire
  false positives inside code blocks within fixture files

## Tasks

1. Update `internal/integration/rules_test.go`:
   - In the `good` subtest: replace `filterByRule(r.Check(f), ruleID)`
     with a loop over ALL registered rules (`rule.All()`), collecting
     diagnostics without filtering. Assert zero total diagnostics.
   - In the `fix` subtest: after verifying the fix output matches
     `fixed.md`, parse `fixed.md` through ALL rules and assert zero
     diagnostics.
   - Keep the `bad` subtest unchanged (it intentionally tests only the
     target rule).

2. Run `go test ./internal/integration/...` to identify all fixture
   files that now fail.

3. Audit and fix each `rules/TM*/good.md`:
   - Ensure it has a heading on line 1 (TM004)
   - Ensure no trailing spaces (TM006)
   - Ensure no hard tabs (TM007)
   - Ensure no multiple blank lines (TM008)
   - Ensure single trailing newline (TM009)
   - Ensure headings use ATX style (TM002)
   - Ensure heading increments (TM003)
   - Ensure blank lines around headings (TM013)
   - Ensure blank lines around lists (TM014)
   - Ensure blank lines around fenced code (TM015)
   - Ensure fenced code has language tags (TM011)
   - Ensure no bare URLs (TM012)
   - Ensure no trailing punctuation in headings (TM017)
   - Ensure no emphasis-as-heading (TM018)
   - Ensure lines under 80 chars (TM001)
   - Ensure fenced code uses backtick style (TM010)
   - Ensure list indent is 2 spaces (TM016)

4. Audit and fix each `rules/TM*/fixed.md` with the same checks.

5. Where a good.md needs content that would violate another rule to
   demonstrate what "good" looks like for its own rule, restructure the
   example to avoid the cross-rule violation. If impossible, document
   it as a known limitation.

6. Run the full test suite to verify all fixtures pass.

## Acceptance Criteria

- [ ] `good` subtests run ALL rules, not just the target rule
- [ ] `fix` subtests verify `fixed.md` passes ALL rules
- [ ] Every `rules/TM*/good.md` produces zero diagnostics from all rules
- [ ] Every `rules/TM*/fixed.md` produces zero diagnostics from all rules
- [ ] No `bad.md` test behavior is changed
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
