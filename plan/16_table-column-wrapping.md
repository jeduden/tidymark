# Table Column Wrapping for Generated Content

## Goal

Generated content (TM019 catalog tables) should produce lines within the
configured line-length limit. Default: truncate/shorten long column values with
ellipsis. Opt-in: `<br>` wrapping for renderers that support it.

Note: TM001 table exclusion (`exclude: ["tables"]`) is handled by plan 14.

## Prerequisites

- Plan 14 (config settings) — needed to read the effective `line-length.max`
  from config; also adds the `exclude` list to TM001 with `"tables"` support

## Tasks

### A. TM019: Column Truncation (Default)

1. Add per-column width configuration to the catalog directive YAML:

   ```yaml
   <!-- tidymark:catalog glob="rules/TM*-*/README.md"
     columns:
       description:
         max-width: 50
   -->
   ```

   Parse this in `internal/rules/generatedsection/rule.go` directive
   parsing. If no per-column config is given, derive widths from a
   global `max-line-length` parameter (or the effective TM001 max).

6. Implement truncation logic in
   `internal/rules/generatedsection/wrap.go`:
   - `truncateCell(text string, maxWidth int) string` — truncate at
     word boundary and append `...` if the text exceeds maxWidth
   - Preserve markdown links — don't break inside `[...](...)`
   - Preserve inline code — don't break inside `` `...` ``

7. Integrate truncation into `renderTemplate()` in
   `internal/rules/generatedsection/generate.go`:
   - After template expansion for each row, detect table rows
     (lines starting with `|`)
   - Parse column values and apply `truncateCell` to columns that
     have `max-width` constraints
   - Recalculate padding so columns remain aligned

### B. TM019: `<br>` Wrapping (Opt-In)

4. Add a `wrap` option per column in the directive YAML:

   ```yaml
   columns:
     description:
       max-width: 50
       wrap: br
   ```

   Supported values:
   - `truncate` (default) — shorten with ellipsis
   - `br` — split at word boundaries, join with `<br>`

5. Implement `wrapCellBr(text string, maxWidth int) string` in
   `internal/rules/generatedsection/wrap.go`:
   - Break at word boundaries (spaces), join with `<br>`
   - Hard break at character boundary as fallback when a single word
     exceeds maxWidth
   - Preserve markdown links and inline code spans

6. Add unit tests in `internal/rules/generatedsection/wrap_test.go`:
   - `truncateCell`: truncation respecting markdown spans, ellipsis
     appended, short strings unchanged, doesn't cut inside links or
     inline code
   - `wrapCellBr`: soft wrap at word boundaries, hard wrap fallback,
     markdown spans preserved, empty/short strings unchanged
   - Integration: generated table rows respect max line width with
     both modes

### C. Documentation and Verification

7. Update `rules/TM019-generated-section/README.md` to document the
   new `columns` parameter with `max-width` and `wrap` options.

8. Verify that the rule catalog in `README.md` generates table rows
   within the configured `line-length.max` using truncation by default.

## Acceptance Criteria

- [ ] `truncateCell` shortens text with `...`, respecting markdown spans
- [ ] `wrapCellBr` wraps text with `<br>` at word boundaries
- [ ] Per-column `max-width` and `wrap` config is parsed from directive YAML
- [ ] Default wrapping mode is `truncate` (not `br`)
- [ ] Generated table rows respect configured line-length by default
- [ ] Markdown formatting (links, inline code) is preserved across
      truncation and wrapping
- [ ] Unit tests for truncation and `<br>` wrapping pass
- [ ] TM019 README documents the `columns` parameter
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
