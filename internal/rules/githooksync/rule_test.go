package githooksync

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRule_Check_NotInGitRepo(t *testing.T) {
	dir := t.TempDir()
	// No git init - not a git repository

	r := &Rule{configured: true}
	f := &lint.File{
		Path:          filepath.Join(dir, "README.md"),
		Source:        []byte("# Test\n"),
		MaxInputBytes: 1048576,
	}

	// Should return no diagnostics when not in a git repo
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestRule_Check_NotConfigured(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())

	r := &Rule{configured: false}
	f := &lint.File{
		Path:          filepath.Join(dir, "README.md"),
		Source:        []byte("# Test\n"),
		MaxInputBytes: 1048576,
	}

	// Should return no diagnostics when not configured
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestRule_Check_HooksInSync(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())

	// Create README.md with generated content
	readmeFile := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readmeFile,
		[]byte("# Test\n\n<?catalog?>\n<?/catalog?>\n"), 0o644))

	// Create PLAN.md with generated content
	planFile := filepath.Join(dir, "PLAN.md")
	require.NoError(t, os.WriteFile(planFile,
		[]byte("# Plan\n\n<?include file=\"test.md\"?><?/include?>\n"), 0o644))

	// Create pre-merge-commit hook with BOTH files (matching discovery)
	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0o755))
	hookPath := filepath.Join(hooksDir, "pre-merge-commit")
	hookContent := "#!/bin/sh\n# mdsmith pre-merge-commit hook\n" +
		"mdsmith fix -- 'PLAN.md'\ngit add -- 'PLAN.md'\n" +
		"mdsmith fix -- 'README.md'\ngit add -- 'README.md'\n"
	require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0o755))

	// Change to git repo directory so git commands work
	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	r := &Rule{configured: true}
	f := &lint.File{
		Path:          "README.md",
		Source:        []byte("# Test\n\n<?catalog?>\n<?/catalog?>\n"),
		MaxInputBytes: 1048576,
	}

	diags := r.Check(f)
	assert.Empty(t, diags, "Should have no diagnostics when hooks are in sync")
}

func TestRule_Check_PreMergeCommitOutOfSync(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())

	// Only create one file with generated content
	testFile := filepath.Join(dir, "test.md")
	require.NoError(t, os.WriteFile(testFile,
		[]byte("# Test\n\n<?catalog?>\nlist\n<?/catalog?>\n"), 0o644))

	// Create README.md (with generated content) so we can check hooks from it
	readmeFile := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readmeFile,
		[]byte("# README\n\n<?include file=\"test.md\"?><?/include?>\n"), 0o644))

	// Create pre-merge-commit hook with only ONE file (not all discovered files)
	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0o755))
	hookPath := filepath.Join(hooksDir, "pre-merge-commit")
	hookContent := "#!/bin/sh\n# mdsmith pre-merge-commit hook\n" +
		"mdsmith fix -- 'README.md'\ngit add -- 'README.md'\n"
	require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0o755))

	// Change to git repo directory
	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	r := &Rule{configured: true}
	f := &lint.File{
		Path:          "README.md",
		Source:        []byte("# README\n\n<?include file=\"test.md\"?><?/include?>\n"),
		MaxInputBytes: 1048576,
	}

	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "pre-merge-commit hook is out of sync")
	assert.Contains(t, diags[0].Message, "has: README.md")
	// Discovery should find both README.md and test.md
	assert.Contains(t, diags[0].Message, "should have:")
}

func TestRule_ApplySettings(t *testing.T) {
	r := &Rule{}
	assert.False(t, r.configured)

	err := r.ApplySettings(map[string]any{})
	assert.NoError(t, err)
	assert.True(t, r.configured)
}

func TestRule_ApplySettings_UnknownSetting(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"unknown": "value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown setting")
}

func TestFilesMatch(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"empty lists", []string{}, []string{}, true},
		{"same files same order", []string{"a", "b"}, []string{"a", "b"}, true},
		{"same files different order", []string{"a", "b"}, []string{"b", "a"}, true},
		{"different lengths", []string{"a"}, []string{"a", "b"}, false},
		{"different files", []string{"a", "b"}, []string{"a", "c"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filesMatch(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractHookFiles(t *testing.T) {
	content := `#!/bin/sh
# mdsmith pre-merge-commit hook

if [ -e 'PLAN.md' ]; then
  mdsmith fix -- 'PLAN.md'
  git add -- 'PLAN.md'
fi

if [ -e 'README.md' ]; then
  mdsmith fix -- 'README.md'
  git add -- 'README.md'
fi
`
	files := extractHookFiles(content)
	assert.Equal(t, []string{"PLAN.md", "README.md"}, files)
}

func TestExtractMergeDriverFiles(t *testing.T) {
	gitConfig := `[core]
	repositoryformatversion = 0

[merge "mdsmith-PLAN.md"]
	driver = mdsmith merge-driver -- 'PLAN.md' %O %A %B %P
	name = mdsmith merge driver for PLAN.md

[merge "mdsmith-README.md"]
	driver = mdsmith merge-driver -- 'README.md' %O %A %B %P
	name = mdsmith merge driver for README.md
`
	files := extractMergeDriverFiles(gitConfig)
	assert.Equal(t, []string{"PLAN.md", "README.md"}, files)
}
