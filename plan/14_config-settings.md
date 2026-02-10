# Apply Config Settings to Rule Instances

## Goal

Bridge the gap between YAML config settings and rule structs: when config
specifies `line-length: { max: 120 }`, the runner and fixer must apply `Max:
120` to the `linelength.Rule` instance before calling `Check`/`Fix`. Also
expose each rule's default settings for use by `tidymark init`.

## Prerequisites

None — this plan can start immediately.

## Tasks

1. Add a `Configurable` interface in `internal/rule/rule.go`:

   ```go
   type Configurable interface {
       ApplySettings(settings map[string]any) error
       DefaultSettings() map[string]any
   }
   ```

   Rules that have configurable fields implement this interface.

2. Implement `Configurable` on rules with settings:
   - `linelength.Rule` (TM001): keys `max` (int), `exclude` ([]string)
     Replace the current `Strict bool` field with `Exclude []string`.
     Valid values: `"code-blocks"`, `"tables"`, `"urls"`.
     Default: `["code-blocks", "tables", "urls"]` (all three excluded).
     Empty list `[]` means check everything (equivalent to old
     `strict: true`). Add a deprecation shim in `ApplySettings`: if
     settings contain `strict: true`, translate to `exclude: []`; if
     `strict: false`, translate to the default exclude list.
     Refactor `Check` to use `r.isExcluded("code-blocks")` etc. instead
     of `if !r.Strict`. Add table-line detection (regex `^\s*\|` or AST
     walk for table nodes) gated by `r.isExcluded("tables")`.
     Config example:
     ```yaml
     rules:
       line-length:
         max: 80
         exclude:
           - code-blocks
           - urls
     ```
   - `headingstyle.Rule` (TM002): key `style` (string)
   - `firstlineheading.Rule` (TM004): key `level` (int)
   - `nomultipleblanks.Rule` (TM008): key `max` (int) — add `Max int`
     field with default 1 (currently hardcoded to disallow >1 blank)
   - `fencedcodestyle.Rule` (TM010): key `style` (string)
   - `listindent.Rule` (TM016): key `spaces` (int)

3. Add a `CloneRule(r Rule) Rule` function in `internal/rule/clone.go`
   that deep-copies a rule (via `ApplySettings(DefaultSettings())` on a
   fresh instance, or reflection). This is needed because a single
   registered rule instance is shared across files, but per-file
   overrides require independent copies.

4. Update `internal/engine/runner.go` (`Runner.Run`): after computing
   `effective` config for a file, for each enabled rule, clone it and
   call `ApplySettings(cfg.Settings)` if the rule implements
   `Configurable` and `cfg.Settings` is non-nil. Use the cloned rule
   for `Check`.

5. Update `internal/fix/fix.go` (`Fixer.Fix`): same approach — clone
   and apply settings before `Check`/`Fix` calls. Update
   `fixableRules` to accept effective config and return cloned,
   configured instances.

6. Add `MarshalYAML` to `config.RuleCfg` in `internal/config/config.go`
   so that a `RuleCfg{Enabled: true, Settings: map[string]any{"max": 80}}`
   serializes as `max: 80` (mapping form) and `RuleCfg{Enabled: false}`
   serializes as `false`.

7. Add a `DumpDefaults() *Config` function in `internal/config/load.go`
   that iterates all registered rules, checks for `Configurable`, and
   builds a `Config` with each rule's `DefaultSettings()` populated in
   `RuleCfg.Settings`. This is consumed by `tidymark init` (plan 15).

8. Update `rules/TM001-line-length/README.md`: document the new
   `exclude` setting replacing `strict`, with examples.

9. Add unit tests:
   - `ApplySettings` on each configurable rule: valid settings, type
     errors, unknown keys
   - `DefaultSettings` returns expected defaults
   - `CloneRule` produces independent copies
   - `MarshalYAML` round-trips correctly
   - `DumpDefaults` produces a valid config with all defaults
   - Runner applies settings from config: create a file that's 100 chars
     wide, configure `line-length: { max: 120 }`, verify no TM001 diagnostic
   - TM001 `exclude` settings: `exclude: ["code-blocks"]` skips code
     blocks but checks tables/urls; `exclude: []` checks everything;
     default excludes all three; `strict: true` deprecation shim works
   - TM001 table exclusion: long table rows skipped when `"tables"` in
     exclude list, flagged when not

## Acceptance Criteria

- [ ] `Configurable` interface exists in `internal/rule/rule.go`
- [ ] All six configurable rules implement `Configurable`
- [ ] `CloneRule` produces independent copies
- [ ] Runner applies `Settings` from effective config before `Check`
- [ ] Fixer applies `Settings` from effective config before `Check`/`Fix`
- [ ] `MarshalYAML` on `RuleCfg` round-trips correctly
- [ ] `DumpDefaults()` returns a config with all rule defaults
- [ ] Unit tests for all new functions pass
- [ ] All tests pass: `go test ./...`
- [ ] `golangci-lint run` reports no issues
