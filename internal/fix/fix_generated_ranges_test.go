package fix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alwaysFiringRule emits one diagnostic per line in the file. It is
// non-fixable, so any diagnostic the engine doesn't filter survives
// into the post-fix result.
type alwaysFiringRule struct {
	id   string
	name string
}

func (r *alwaysFiringRule) ID() string       { return r.id }
func (r *alwaysFiringRule) Name() string     { return r.name }
func (r *alwaysFiringRule) Category() string { return "test" }

func (r *alwaysFiringRule) Check(f *lint.File) []lint.Diagnostic {
	diags := make([]lint.Diagnostic, 0, len(f.Lines))
	for i := range f.Lines {
		diags = append(diags, lint.Diagnostic{
			File:     f.Path,
			Line:     i + 1,
			Column:   1,
			RuleID:   r.id,
			RuleName: r.name,
			Severity: lint.Warning,
			Message:  "always",
		})
	}
	return diags
}

// TestFix_SuppressesDiagnosticsInsideCatalogBody mirrors
// engine.TestLintOnce_CatalogHost on the fix path: diagnostics whose
// lines fall inside a <?catalog?> generated body must not surface
// from the host's perspective. Without this, `mdsmith fix` and
// `mdsmith check` disagree on the same source bytes — check filters
// the body, fix doesn't — which is what surfaced as a CI-only failure
// in the merge queue when the pre-merge-commit hook ran fix on a
// merge containing a long catalog summary.
func TestFix_SuppressesDiagnosticsInsideCatalogBody(t *testing.T) {
	dir := t.TempDir()

	// Lines:
	// 1: # Catalog Host
	// 2: (empty)
	// 3: <?catalog
	// 4: glob: "*.md"
	// 5: row: "- {filename}"
	// 6: ?>
	// 7: - body-line.md         <-- generated-body content (line 7)
	// 8: <?/catalog?>
	host := "# Catalog Host\n\n" +
		"<?catalog\nglob: \"*.md\"\nrow: \"- {filename}\"\n?>\n" +
		"- body-line.md\n" +
		"<?/catalog?>\n"
	hostPath := filepath.Join(dir, "host.md")
	require.NoError(t, os.WriteFile(hostPath, []byte(host), 0o644))

	const ruleName = "always-fires"
	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			ruleName: {Enabled: true},
		},
	}

	fixer := &Fixer{
		Config: cfg,
		Rules:  []rule.Rule{&alwaysFiringRule{id: "MDS999", name: ruleName}},
	}

	result := fixer.Fix([]string{hostPath})
	require.Empty(t, result.Errors, "unexpected errors: %v", result.Errors)

	// Line 7 is inside the generated catalog body. A diagnostic on
	// that line must be filtered out before reaching the result.
	for _, d := range result.Diagnostics {
		if d.RuleName == ruleName && d.Line == 7 {
			t.Errorf("fix surfaced %s diagnostic from inside <?catalog?> body (line 7): %s",
				ruleName, d.Message)
		}
	}

	// Pre-fix Failures count must also exclude the generated-body
	// line. The host has 9 line entries (8 source lines plus the
	// trailing-newline split) but line 7 is inside the generated
	// body, so fewer than 9 should remain after generated-body
	// filtering. The strict-less assertion stays robust to unrelated
	// trailing-line counting tweaks.
	assert.Less(t, result.Failures, 9,
		"pre-fix Failures should exclude the generated-body line, got %d (no filtering applied)",
		result.Failures)
}

// fieldRecordingRule captures f.MaxInputBytes and f.StripFrontMatter
// on every Check invocation. Rules in the wild (catalog, include,
// duplicatedcontent, requiredstructure, crossfilereferenceintegrity)
// consult these fields when reading secondary files; if the Fixer's
// post-fix lint pass receives a *lint.File where these are zero, the
// post-fix CheckRules call diverges from the pre-fix call and from
// the Runner.
type fieldRecordingRule struct {
	id       string
	name     string
	maxBytes []int64
	stripFM  []bool
}

func (r *fieldRecordingRule) ID() string       { return r.id }
func (r *fieldRecordingRule) Name() string     { return r.name }
func (r *fieldRecordingRule) Category() string { return "test" }
func (r *fieldRecordingRule) Check(f *lint.File) []lint.Diagnostic {
	r.maxBytes = append(r.maxBytes, f.MaxInputBytes)
	r.stripFM = append(r.stripFM, f.StripFrontMatter)
	return nil
}

// TestFix_FinalFileInheritsParseTimeFields verifies that the post-fix
// CheckRules call receives a *lint.File whose MaxInputBytes and
// StripFrontMatter mirror the pre-fix file's values. The Runner
// already establishes these on its single File before linting; the
// Fixer parses a fresh finalFile after applyFixPasses and must
// propagate the same fields, otherwise rules that consult them (e.g.
// catalog/include reading secondary files, duplicatedcontent computing
// cross-file coordinates) silently diverge between fix and check on
// the same source bytes.
func TestFix_FinalFileInheritsParseTimeFields(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("---\ntitle: t\n---\n# Doc\n"), 0o644))

	const ruleName = "field-recorder"
	const wantMaxBytes int64 = 4096
	rec := &fieldRecordingRule{id: "MDS998", name: ruleName}

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			ruleName: {Enabled: true},
		},
	}

	fixer := &Fixer{
		Config:           cfg,
		Rules:            []rule.Rule{rec},
		StripFrontMatter: true,
		MaxInputBytes:    wantMaxBytes,
	}

	result := fixer.Fix([]string{mdPath})
	require.Empty(t, result.Errors, "unexpected errors: %v", result.Errors)

	// Two CheckRules calls per file: one pre-fix on lf, one post-fix
	// on finalFile. Both must see the configured values.
	require.Len(t, rec.maxBytes, 2,
		"expected one Check call pre-fix and one post-fix, got %d", len(rec.maxBytes))
	for i, got := range rec.maxBytes {
		assert.Equal(t, wantMaxBytes, got,
			"call %d: MaxInputBytes not propagated to lint.File", i)
	}
	for i, got := range rec.stripFM {
		assert.True(t, got,
			"call %d: StripFrontMatter not propagated to lint.File", i)
	}
}
