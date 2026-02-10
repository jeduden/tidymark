# Self-Lint Setup: Make Tidymark Lint Itself

## Goal

Configure tidymark to lint its own markdown files cleanly, fix all rule
README front-matter and content issues, and document best practices for
config management.

## Prerequisites

- Plan 13 (code-block-awareness) — removes need for TM008 override
  workaround
- Plan 14 (config settings) — needed for `tidymark init` docs
- Plan 15 (CLI subcommands) — needed for `tidymark init` docs in README
- Plan 16 (table column wrapping) — removes need for `max: 400`
  workaround
- Plan 17 (fixture cleanup) — ensures good.md/fixed.md are clean

**Note**: Subtasks 1–4 can start immediately (workarounds). Subtask 5
(docs) waits on plans 15. Subtask 6 (verify clean run) waits on all
prerequisites.

## Tasks

1. **Fix rule README front matter**: In all 19 `rules/TM*/README.md`
   files, remove the blank line between the closing `---` front-matter
   delimiter and the `# TM0XX:` heading. After front-matter stripping,
   the first line must be the heading (required by TM004). Files to
   update:
   - `rules/TM001-line-length/README.md`
   - `rules/TM002-heading-style/README.md`
   - `rules/TM003-heading-increment/README.md`
   - `rules/TM004-first-line-heading/README.md`
   - `rules/TM005-no-duplicate-headings/README.md`
   - `rules/TM006-no-trailing-spaces/README.md`
   - `rules/TM007-no-hard-tabs/README.md`
   - `rules/TM008-no-multiple-blanks/README.md`
   - `rules/TM009-single-trailing-newline/README.md`
   - `rules/TM010-fenced-code-style/README.md`
   - `rules/TM011-fenced-code-language/README.md`
   - `rules/TM012-no-bare-urls/README.md`
   - `rules/TM013-blank-line-around-headings/README.md`
   - `rules/TM014-blank-line-around-lists/README.md`
   - `rules/TM015-blank-line-around-fenced-code/README.md`
   - `rules/TM016-list-indent/README.md`
   - `rules/TM017-no-trailing-punctuation-in-heading/README.md`
   - `rules/TM018-no-emphasis-as-heading/README.md`
   - `rules/TM019-generated-section/README.md`

2. **Fix TM019 README content issues**:
   - Add `text` language tag to the 2 unlabeled fenced code blocks
     (~lines 16 and 211 in `rules/TM019-generated-section/README.md`)
   - Convert `**Parameters** (directive-specific):` bold text to a
     proper heading: `#### Parameters (directive-specific)`
   - Convert `**Built-in keys:**` bold text to a proper heading:
     `#### Built-in keys`

3. **Rewrite `.tidymark.yml`** with proper config (workarounds noted):

   ```yaml
   front-matter: true

   ignore:
     - "internal/**"
     - "plan/**"
     - "rules/*/bad.md"

   overrides:
     - files: ["README.md"]
       rules:
         line-length: false
     - files: ["rules/*/README.md"]
       rules:
         line-length:
           max: 400
     - files: ["rules/TM008-no-multiple-blanks/README.md"]
       rules:
         no-multiple-blanks: false
   ```

   Notes:
   - Only `bad.md` is excluded (good.md/fixed.md should be clean after
     plan 17)
   - `no-multiple-blanks: false` override is a workaround until plan 13
     ships (then remove it)
   - `max: 400` is a workaround until plan 16 ships (then reduce/remove)

4. **Add `.gitignore`**: Exclude the `tidymark` binary from version
   control:

   ```
   tidymark
   ```

5. **README best practices section**: Add a section to `README.md`
   explaining the `tidymark init` command — how it dumps all rule
   defaults into config so that upstream default changes don't
   silently break existing usage. Depends on plan 15 (`init`
   subcommand existing).

6. **Verify clean run**: After all prerequisites are complete:
   - `./tidymark check .` exits 0 (or `./tidymark .` if backwards
     compat is kept)
   - `go test ./...` passes
   - No workaround overrides remain in `.tidymark.yml` (TM008 override
     removed after plan 13, `max: 400` removed after plan 16)

## Acceptance Criteria

- [ ] All 19 rule READMEs have no blank line between `---` and heading
- [ ] TM019 README has language tags on all code blocks
- [ ] TM019 README uses proper headings instead of bold text
- [ ] `.tidymark.yml` ignores only `internal/**`, `plan/**`, `rules/*/bad.md`
- [ ] `.gitignore` excludes `tidymark` binary
- [ ] README documents `tidymark init` best practices (after plan 15)
- [ ] `./tidymark check .` exits 0 (after all plans complete)
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
