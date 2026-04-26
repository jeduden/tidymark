package engine

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoHardcodedKindNamesInCore enforces that the mdsmith core (config,
// engine, lint, rule packages) contains no `if kind == "..."` branches
// and no string literals that hardcode specific kind names. Kinds are an
// extension point: the linter must not special-case any kind by name.
//
// The test scans Go source files (excluding _test.go) under known core
// directories and:
//   - rejects any line matching `kind\s*==\s*"..."` (a literal-named
//     kind comparison)
//   - rejects any line matching `Kind\s*==\s*"..."` for the same reason
//
// To remain useful as the codebase evolves, the test errors when a
// hardcoded kind comparison is reintroduced. Documentation strings (for
// example, in `mdsmith help kinds` content) are intentionally placed in
// _test.go-excluded files when needed; if a real reason to embed a
// concrete kind name in core code arises, the test failure forces a
// design discussion before the regression is allowed.
func TestNoHardcodedKindNamesInCore(t *testing.T) {
	repoRoot := findRepoRoot(t)

	coreDirs := []string{
		filepath.Join(repoRoot, "internal", "config"),
		filepath.Join(repoRoot, "internal", "engine"),
		filepath.Join(repoRoot, "internal", "lint"),
		filepath.Join(repoRoot, "internal", "rule"),
	}

	// Patterns that indicate a hardcoded kind comparison.
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bkind\s*==\s*"[^"]+"`),
		regexp.MustCompile(`(?i)"[^"]+"\s*==\s*\bkind\b`),
	}

	var offences []string
	for _, dir := range coreDirs {
		require.NoError(t, filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for lineNum, line := range strings.Split(string(data), "\n") {
				for _, p := range patterns {
					if p.MatchString(line) {
						offences = append(offences,
							path+":"+itoa(lineNum+1)+": "+strings.TrimSpace(line))
					}
				}
			}
			return nil
		}))
	}

	assert.Empty(t, offences,
		"hardcoded kind-name comparisons found in core; kinds must remain a configuration extension point:\n  %s",
		strings.Join(offences, "\n  "))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf []byte
	for i > 0 {
		buf = append([]byte{byte('0' + i%10)}, buf...)
		i /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// findRepoRoot walks up from the test working directory until it finds
// a go.mod file, returning that directory. The test fails if no go.mod
// is found before reaching the filesystem root.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (go.mod) above test working dir")
		}
		dir = parent
	}
}
