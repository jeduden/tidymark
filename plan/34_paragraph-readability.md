# Paragraph Readability Score

## Goal

Implement TM023 `paragraph-readability` that computes a
readability score (Flesch-Kincaid grade level) for each
paragraph and warns when the score exceeds a configurable
threshold, encouraging clearer writing.

## Tasks

### A. Readability computation

1. Implement Flesch-Kincaid grade level formula:

   ```text
   0.39 * (words / sentences)
     + 11.8 * (syllables / words)
     - 15.59
   ```

   Create `internal/rules/paragraphreadability/readability.go`
   with:

  - `countSyllables(word string) int`
  - `countSentences(text string) int`
  - `fleschKincaidGrade(text string) float64`

2. Handle edge cases: single-word paragraphs, no sentences
   (treat as 1), code spans stripped before scoring,
   links reduced to display text.

### B. Rule implementation

3. Create `internal/rules/paragraphreadability/rule.go`
   implementing `rule.Rule` and `rule.Configurable`.
   Settings:

   | Setting | Type | Default | Description |
   |---------|------|---------|-------------|
   | `max-grade` | float | 12.0 | Max Flesch-Kincaid grade level |
   | `min-words` | int | 20 | Minimum words to trigger check |

   Paragraphs with fewer than `min-words` words are skipped
   (short paragraphs produce unreliable scores).

4. Check logic: walk the AST for `*ast.Paragraph` nodes.
   For each, extract plain text (strip inline markup),
   compute grade level. If it exceeds `max-grade`, emit:

  - `paragraph readability grade too high (14.2 > 12.0)`

   Diagnostic on the paragraph's first line.

5. Register rule, add blank import, set category to
   `"meta"`.

### C. Markdown text extraction

6. Create a helper `extractPlainText(node ast.Node,
   source []byte) string` that strips markdown syntax:

  - Inline code: keep text content
  - Links: keep display text, drop URL
  - Emphasis/strong: keep inner text
  - Images: keep alt text

### D. Documentation

7. Write `rules/TM023-paragraph-readability/README.md`.

8. Create test fixtures with high-grade and low-grade
   paragraphs.

### E. Tests

9. Unit tests for `countSyllables`: common words, edge
   cases (silent e, diphthongs, single-syllable).

10. Unit tests for `fleschKincaidGrade`: known reference
    texts with expected scores.

11. Unit tests for rule Check: paragraph over threshold,
    paragraph under threshold, short paragraph skipped,
    paragraph with inline markup.

12. Run `go test ./...` and `go tool golangci-lint run`.

## Acceptance Criteria

- [ ] Flesch-Kincaid grade computed for each paragraph
- [ ] Paragraphs below `min-words` skipped
- [ ] `max-grade` configurable, default 12.0
- [ ] Inline markup stripped before scoring
- [ ] Rule README with examples
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
