# Paragraph Structure Limits

## Goal

Implement TM024 `paragraph-structure` that enforces structural
limits on markdown paragraphs: maximum sentences per paragraph
and maximum words per sentence, encouraging concise writing.

## Tasks

### A. Sentence detection

1. Create `internal/rules/paragraphstructure/sentences.go`
   with a sentence splitter:

  - Split on `.`, `!`, `?` followed by whitespace or
     end of text
  - Handle abbreviations (e.g., `e.g.`, `i.e.`, `Dr.`,
     `Mr.`, `vs.`) -- do not split on these
  - Handle ellipsis (`...`) -- single sentence boundary
  - Handle decimal numbers (`3.14`) -- not a boundary

2. Create `countWords(sentence string) int` that counts
   whitespace-separated tokens after stripping markdown
   inline syntax.

### B. Rule implementation

3. Create `internal/rules/paragraphstructure/rule.go`
   implementing `rule.Rule` and `rule.Configurable`.
   Settings:

   | Setting | Type | Default | Description |
   |---------|------|---------|-------------|
   | `max-sentences` | int | 6 | Max sentences per paragraph |
   | `max-words` | int | 40 | Max words per sentence |

4. Check logic: walk AST for `*ast.Paragraph` nodes.
   For each paragraph:

  - Extract plain text (reuse or share the helper from
     TM023 if available)
  - Split into sentences
  - If sentence count exceeds `max-sentences`:
     `paragraph has too many sentences (8 > 6)`
  - For each sentence exceeding `max-words`:
     `sentence too long (45 > 40 words)`
  - Diagnostics on the paragraph's first line

5. Register rule, add blank import, set category to
   `"meta"`.

### C. Documentation

6. Write `rules/TM024-paragraph-structure/README.md`.

7. Create test fixtures:

  - `bad.md`: paragraph with 8 sentences, sentence
     with 50 words
  - `good.md`: well-structured paragraphs

### D. Tests

8. Unit tests for sentence splitter:

  - Simple sentences with `.`, `!`, `?`
  - Abbreviations not split
  - Ellipsis handled
  - Decimal numbers not split
  - Empty input

9. Unit tests for rule Check:

  - Paragraph over `max-sentences`: diagnostic
  - Paragraph under limit: no diagnostic
  - Sentence over `max-words`: diagnostic
  - Short paragraph: no diagnostic
  - Custom settings respected
  - Both limits exceeded: two diagnostics

10. Run `go test ./...` and `go tool golangci-lint run`.

## Acceptance Criteria

- [ ] Sentence splitter handles abbreviations and decimals
- [ ] `max-sentences` enforced per paragraph, default 6
- [ ] `max-words` enforced per sentence, default 40
- [ ] Diagnostics include counts
- [ ] Rule README with examples
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
