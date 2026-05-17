package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseDryRunStats parses the stats line from a dry-run stderr output.
// Expected format: stats: checked=N fixed=0 failures=N unfixed=N would-fix=N
func parseDryRunStats(t *testing.T, stderr string) (checked, fixed, failures, unfixed, wouldFix int) {
	t.Helper()
	re := regexp.MustCompile(`stats: checked=(\d+) fixed=(\d+) failures=(\d+) unfixed=(\d+) would-fix=(\d+)`)
	m := re.FindStringSubmatch(stderr)
	require.Len(t, m, 6, "expected dry-run stats line in stderr, got: %s", stderr)

	values := make([]int, 5)
	for i := 0; i < 5; i++ {
		v, err := strconv.Atoi(m[i+1])
		require.NoError(t, err, "parsing stats value %q: %v", m[i+1], err)
		values[i] = v
	}
	return values[0], values[1], values[2], values[3], values[4]
}

// TestE2E_FixDryRun_WritesNothingToDisk verifies that --dry-run does not
// modify any file on disk.
func TestE2E_FixDryRun_WritesNothingToDisk(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)

	content := "# Title\n\nHello   \n"
	path := writeFixture(t, dir, "fixme.md", content)

	_, _, exitCode := runBinaryInDir(t, dir, "", "fix", "--dry-run", "--no-color", "fixme.md")

	// Dry-run should exit 0 (all violations are fixable).
	assert.Equal(t, 0, exitCode, "expected exit code 0 for dry-run with fixable file, got %d", exitCode)

	// File must be byte-identical to original.
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(got), "dry-run must not modify the file on disk")
}

// TestE2E_FixDryRun_ReportsSameCountAsRealRun verifies that the would-fix
// count in dry-run matches the fixed count from a real run on the same input.
func TestE2E_FixDryRun_ReportsSameCountAsRealRun(t *testing.T) {
	// Set up two identical dirs: one for dry-run, one for real run.
	dirDry := t.TempDir()
	isolateDir(t, dirDry)
	dirReal := t.TempDir()
	isolateDir(t, dirReal)

	content := "# Title\n\nHello   \n"
	writeFixture(t, dirDry, "fixme.md", content)
	writeFixture(t, dirReal, "fixme.md", content)

	// Dry run.
	_, stderrDry, _ := runBinaryInDir(t, dirDry, "", "fix", "--dry-run", "--no-color", "fixme.md")
	_, _, _, _, wouldFix := parseDryRunStats(t, stderrDry)

	// Real run.
	_, stderrReal, _ := runBinaryInDir(t, dirReal, "", "fix", "--no-color", "fixme.md")
	_, fixed, _, _ := parseStats(t, stderrReal)

	assert.Equal(t, fixed, wouldFix,
		"dry-run would-fix=%d should match real-run fixed=%d", wouldFix, fixed)
}

// TestE2E_FixDryRun_ExitCodeMatchesRealRun verifies that the exit code from
// --dry-run matches what a real run would return on identical input.
func TestE2E_FixDryRun_ExitCodeMatchesRealRun(t *testing.T) {
	tests := []struct {
		name    string
		content string
		// 0 = all fixable, 1 = unfixable remain
		wantCode int
	}{
		{
			name:     "all_fixable",
			content:  "# Title\n\nHello   \n",
			wantCode: 0,
		},
		{
			name:     "unfixable_remain",
			content:  "# Title!\n\nHello\n", // heading with punctuation is unfixable
			wantCode: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dirDry := t.TempDir()
			isolateDir(t, dirDry)
			dirReal := t.TempDir()
			isolateDir(t, dirReal)

			writeFixture(t, dirDry, "test.md", tc.content)
			writeFixture(t, dirReal, "test.md", tc.content)

			_, _, dryCode := runBinaryInDir(t, dirDry, "", "fix", "--dry-run", "--no-color", "test.md")
			_, _, realCode := runBinaryInDir(t, dirReal, "", "fix", "--no-color", "test.md")

			assert.Equal(t, realCode, dryCode,
				"dry-run exit code %d must match real-run exit code %d", dryCode, realCode)
		})
	}
}

// TestE2E_FixDryRun_SummaryLine verifies the stats line format:
// fixed=0, would-fix=N present.
func TestE2E_FixDryRun_SummaryLine(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)

	writeFixture(t, dir, "fixme.md", "# Title\n\nHello   \n")

	_, stderr, _ := runBinaryInDir(t, dir, "", "fix", "--dry-run", "--no-color", "fixme.md")

	checked, fixed, _, _, wouldFix := parseDryRunStats(t, stderr)
	assert.Equal(t, 1, checked, "expected checked=1")
	assert.Equal(t, 0, fixed, "expected fixed=0 on dry-run (nothing written)")
	assert.Greater(t, wouldFix, 0, "expected would-fix > 0 for a fixable file")
}

// TestE2E_FixDryRun_PerFileOutput verifies that per-file "would fix N violations"
// lines appear in output for files with violations.
func TestE2E_FixDryRun_PerFileOutput(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)

	writeFixture(t, dir, "fixme.md", "# Title\n\nHello   \n")

	_, stderr, _ := runBinaryInDir(t, dir, "", "fix", "--dry-run", "--no-color", "fixme.md")

	assert.Contains(t, stderr, "would fix", "expected 'would fix' in stderr output, got: %s", stderr)
	assert.Contains(t, stderr, "fixme.md", "expected filename in stderr output, got: %s", stderr)
}

// TestE2E_FixDryRun_JSONOutput verifies that --format json exposes would_fix and rules.
func TestE2E_FixDryRun_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)

	path := writeFixture(t, dir, "fixme.md", "# Title\n\nHello   \n")

	_, stderr, exitCode := runBinaryInDir(t, dir, "", "fix", "--dry-run", "--format", "json", path)
	assert.Equal(t, 0, exitCode, "expected exit code 0, got %d; stderr: %s", exitCode, stderr)

	// The JSON output should be parseable.
	var records []map[string]any
	require.NoError(t, json.Unmarshal([]byte(stderr), &records),
		"stderr is not valid JSON: %s", stderr)

	// Find the record for our file.
	var found map[string]any
	for _, r := range records {
		if p, ok := r["path"].(string); ok && filepath.Base(p) == "fixme.md" {
			found = r
			break
		}
	}
	require.NotNil(t, found, "expected a record for fixme.md in JSON output, got: %v", records)

	// Check required fields.
	_, hasWouldFix := found["would_fix"]
	_, hasRules := found["rules"]
	_, hasDiagnostics := found["diagnostics"]
	assert.True(t, hasWouldFix, "JSON record missing 'would_fix' field: %v", found)
	assert.True(t, hasRules, "JSON record missing 'rules' field: %v", found)
	assert.True(t, hasDiagnostics, "JSON record missing 'diagnostics' field: %v", found)

	// would_fix should be > 0 for a fixable file.
	wf, ok := found["would_fix"].(float64)
	assert.True(t, ok, "expected would_fix to be a number, got %T", found["would_fix"])
	assert.Greater(t, wf, float64(0), "expected would_fix > 0 for a fixable file")

	// rules should be a non-empty array.
	rules, ok := found["rules"].([]any)
	assert.True(t, ok, "expected rules to be an array, got %T", found["rules"])
	assert.NotEmpty(t, rules, "expected non-empty rules array for a fixable file")
}

// TestE2E_FixDryRun_CleanFile_NoOutput verifies that files with no
// violations do not appear in the per-file output.
func TestE2E_FixDryRun_CleanFile_NoOutput(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)

	writeFixture(t, dir, "clean.md", "# Title\n\nSome content here.\n")

	_, stderr, exitCode := runBinaryInDir(t, dir, "", "fix", "--dry-run", "--no-color", "clean.md")
	assert.Equal(t, 0, exitCode, "expected exit code 0 for clean file, got %d", exitCode)
	assert.NotContains(t, stderr, "would fix",
		"clean file should not appear in would-fix output, got: %s", stderr)
}

// TestE2E_FixDryRun_Quiet_SuppressesOutput verifies that --quiet suppresses
// the per-file and stats output.
func TestE2E_FixDryRun_Quiet_SuppressesOutput(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)

	writeFixture(t, dir, "fixme.md", "# Title\n\nHello   \n")

	_, stderr, exitCode := runBinaryInDir(t, dir, "", "fix", "--dry-run", "--quiet", "fixme.md")
	assert.Equal(t, 0, exitCode, "expected exit code 0, got %d", exitCode)
	assert.NotContains(t, stderr, "stats:", "quiet mode should suppress stats, got: %s", stderr)
	assert.NotContains(t, stderr, "would fix", "quiet mode should suppress per-file output, got: %s", stderr)
}
