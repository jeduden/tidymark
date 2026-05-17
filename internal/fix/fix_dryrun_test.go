package fix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/rule"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDryRun_WritesNothingToDisk verifies that --dry-run leaves files
// byte-identical after the run.
func TestDryRun_WritesNothingToDisk(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	original := []byte("# Hello  \nworld  \n")
	require.NoError(t, os.WriteFile(mdFile, original, 0o644))

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"mock-trailing": {Enabled: true},
		},
	}

	fixer := &Fixer{
		Config: cfg,
		Rules:  []rule.Rule{&mockFixableRule{id: "MDS100", name: "mock-trailing"}},
		DryRun: true,
	}

	result := fixer.Fix([]string{mdFile})
	require.Len(t, result.Errors, 0, "unexpected errors: %v", result.Errors)

	// File must be unchanged.
	got, err := os.ReadFile(mdFile)
	require.NoError(t, err)
	assert.Equal(t, original, got, "dry-run must not modify files on disk")

	// Modified list must be empty.
	assert.Len(t, result.Modified, 0, "dry-run must report no modified files")
}

// TestDryRun_ReportsSameFixCount verifies that the WouldFix count on a
// dry-run equals the number of violations fixed by a real run on the same
// input.
func TestDryRun_ReportsSameFixCount(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	content := []byte("# Hello  \nworld  \n")
	require.NoError(t, os.WriteFile(mdFile, content, 0o644))

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"mock-trailing": {Enabled: true},
		},
	}

	// Dry run.
	dryFile := filepath.Join(dir, "dry.md")
	require.NoError(t, os.WriteFile(dryFile, content, 0o644))
	dryFixer := &Fixer{
		Config: cfg,
		Rules:  []rule.Rule{&mockFixableRule{id: "MDS100", name: "mock-trailing"}},
		DryRun: true,
	}
	dryResult := dryFixer.Fix([]string{dryFile})
	require.Len(t, dryResult.Errors, 0)

	// Real run.
	realFile := filepath.Join(dir, "real.md")
	require.NoError(t, os.WriteFile(realFile, content, 0o644))
	realFixer := &Fixer{
		Config: cfg,
		Rules:  []rule.Rule{&mockFixableRule{id: "MDS100", name: "mock-trailing"}},
	}
	realResult := realFixer.Fix([]string{realFile})
	require.Len(t, realResult.Errors, 0)

	// WouldFix in dry run should equal Failures minus remaining in real run.
	realFixed := realResult.Failures - len(realResult.Diagnostics)
	assert.Equal(t, realFixed, dryResult.WouldFix,
		"dry-run WouldFix should match real-run fixed count")

	// WouldFix should be positive (there were violations to fix).
	assert.Greater(t, dryResult.WouldFix, 0)
}

// TestDryRun_ExitCodeMatchesRealRun verifies that WouldFix > 0 when real
// run modifies, and that unfixed diagnostics are correctly reported.
func TestDryRun_ExitCodeMatchesRealRun(t *testing.T) {
	dir := t.TempDir()
	content := []byte("# Hello  \n")

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"mock-trailing":   {Enabled: true},
			"mock-nonfixable": {Enabled: true},
		},
	}
	rules := []rule.Rule{
		&mockFixableRule{id: "MDS100", name: "mock-trailing"},
		&mockNonFixableRule{id: "MDS999", name: "mock-nonfixable"},
	}

	// Dry run.
	dryFile := filepath.Join(dir, "dry.md")
	require.NoError(t, os.WriteFile(dryFile, content, 0o644))
	dryFixer := &Fixer{Config: cfg, Rules: rules, DryRun: true}
	dryResult := dryFixer.Fix([]string{dryFile})
	require.Len(t, dryResult.Errors, 0)

	// Real run.
	realFile := filepath.Join(dir, "real.md")
	require.NoError(t, os.WriteFile(realFile, content, 0o644))
	realFixer := &Fixer{Config: cfg, Rules: rules}
	realResult := realFixer.Fix([]string{realFile})
	require.Len(t, realResult.Errors, 0)

	// Both runs report the same unfixed diagnostics.
	assert.Len(t, dryResult.Diagnostics, len(realResult.Diagnostics),
		"dry-run and real-run should report same remaining diagnostics count")

	// Dry run has WouldFix > 0 when real run made changes.
	if len(realResult.Modified) > 0 {
		assert.Greater(t, dryResult.WouldFix, 0,
			"dry-run should report WouldFix > 0 when real run would modify files")
	}
}

// TestDryRun_WouldFixByFile verifies that the WouldFixByFile map is populated.
func TestDryRun_WouldFixByFile(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "a.md")
	file2 := filepath.Join(dir, "b.md")
	require.NoError(t, os.WriteFile(file1, []byte("# A  \n"), 0o644))
	require.NoError(t, os.WriteFile(file2, []byte("# B\n"), 0o644)) // already clean

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"mock-trailing": {Enabled: true},
		},
	}

	fixer := &Fixer{
		Config: cfg,
		Rules:  []rule.Rule{&mockFixableRule{id: "MDS100", name: "mock-trailing"}},
		DryRun: true,
	}

	result := fixer.Fix([]string{file1, file2})
	require.Len(t, result.Errors, 0)

	// file1 has violations; file2 is clean.
	entry1, ok1 := result.WouldFixByFile[file1]
	entry2, ok2 := result.WouldFixByFile[file2]

	assert.True(t, ok1, "file1 should appear in WouldFixByFile")
	assert.Greater(t, entry1.Count, 0, "file1 should have WouldFix > 0")
	assert.Contains(t, entry1.Rules, "MDS100", "file1 should list MDS100 rule")

	if ok2 {
		assert.Equal(t, 0, entry2.Count, "file2 should have 0 would-fix if present")
	}
}

// TestDryRun_ZeroWouldFixOnCleanFile verifies that WouldFix is 0 when
// no fixes would be applied.
func TestDryRun_ZeroWouldFixOnCleanFile(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "clean.md")
	require.NoError(t, os.WriteFile(mdFile, []byte("# Clean\n"), 0o644))

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"mock-trailing": {Enabled: true},
		},
	}

	fixer := &Fixer{
		Config: cfg,
		Rules:  []rule.Rule{&mockFixableRule{id: "MDS100", name: "mock-trailing"}},
		DryRun: true,
	}

	result := fixer.Fix([]string{mdFile})
	require.Len(t, result.Errors, 0)
	assert.Equal(t, 0, result.WouldFix, "clean file: WouldFix should be 0")
}
