package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeProfile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	return p
}

// The duplicate-block case the CI hit: unit profile records a hit
// for a line the e2e profile recorded as zero. Summing must keep
// it covered rather than letting the second file win.
func TestMergeCoverage_SumsDuplicateBlocks(t *testing.T) {
	dir := t.TempDir()
	unit := writeProfile(t, dir, "unit.cov",
		"mode: atomic\n"+
			"cmd/mdsmith/extract.go:26.31,30.3 3 5\n"+
			"cmd/mdsmith/extract.go:30.3,32.4 1 0\n")
	e2e := writeProfile(t, dir, "e2e.cov",
		"mode: atomic\n"+
			"cmd/mdsmith/extract.go:26.31,30.3 3 0\n"+
			"cmd/mdsmith/extract.go:30.3,32.4 1 2\n")
	out := filepath.Join(dir, "merged.cov")
	require.NoError(t, MergeCoverage([]string{unit, e2e}, out))

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	assert.Equal(t, "mode: atomic", lines[0])
	assert.Contains(t, lines, "cmd/mdsmith/extract.go:26.31,30.3 3 5")
	assert.Contains(t, lines, "cmd/mdsmith/extract.go:30.3,32.4 1 2")
}

func TestMergeCoverage_SetModeOrs(t *testing.T) {
	dir := t.TempDir()
	a := writeProfile(t, dir, "a.cov", "mode: set\nx.go:1.1,2.2 1 0\n")
	b := writeProfile(t, dir, "b.cov", "mode: set\nx.go:1.1,2.2 1 1\n")
	out := filepath.Join(dir, "m.cov")
	require.NoError(t, MergeCoverage([]string{a, b}, out))
	got, _ := os.ReadFile(out)
	assert.Contains(t, string(got), "x.go:1.1,2.2 1 1")
}

func TestMergeCoverage_ModeMismatch(t *testing.T) {
	dir := t.TempDir()
	a := writeProfile(t, dir, "a.cov", "mode: set\nx.go:1.1,2.2 1 1\n")
	b := writeProfile(t, dir, "b.cov", "mode: atomic\nx.go:1.1,2.2 1 1\n")
	err := MergeCoverage([]string{a, b}, filepath.Join(dir, "m.cov"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mode mismatch")
}

func TestMergeCoverage_Errors(t *testing.T) {
	dir := t.TempDir()
	assert.Error(t, MergeCoverage(nil, filepath.Join(dir, "m.cov")))
	assert.Error(t, MergeCoverage([]string{filepath.Join(dir, "missing.cov")},
		filepath.Join(dir, "m.cov")))

	bad := writeProfile(t, dir, "bad.cov", "mode: atomic\ngarbage line\n")
	assert.Error(t, MergeCoverage([]string{bad}, filepath.Join(dir, "m.cov")))

	noMode := writeProfile(t, dir, "nm.cov", "x.go:1.1,2.2 1 1\n")
	assert.Error(t, MergeCoverage([]string{noMode}, filepath.Join(dir, "m.cov")))
}
