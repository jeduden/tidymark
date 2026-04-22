package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- visit tests ---

func TestVisit_WalkError(t *testing.T) {
	w := &walker{
		absBase:  "/tmp/base",
		patterns: []string{"**/*.md"},
		seen:     make(map[string]bool),
	}
	err := w.visit("/tmp/base/foo.md", nil, os.ErrPermission)
	assert.ErrorIs(t, err, os.ErrPermission)
}

// TestVisit_SkipsSymlinkDirByDefault confirms the walker returns
// SkipDir for a symlinked directory entry when FollowSymlinks is
// false (the secure default), without needing a pattern.
func TestVisit_SkipsSymlinkDirByDefault(t *testing.T) {
	dir := t.TempDir()
	absBase := dir

	subDir := filepath.Join(dir, "linked")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	w := &walker{
		absBase:  absBase,
		patterns: []string{"**/*.md"},
		seen:     make(map[string]bool),
	}

	info := fakeFileInfo{name: "linked", mode: os.ModeDir | os.ModeSymlink, isDir: true}
	err := w.visit(filepath.Join(absBase, "linked"), info, nil)
	assert.Equal(t, filepath.SkipDir, err)
}

// TestVisit_FollowsSymlinkDirWhenOptedIn asserts the symlinked
// directory is NOT skipped when the walker has FollowSymlinks=true.
func TestVisit_FollowsSymlinkDirWhenOptedIn(t *testing.T) {
	dir := t.TempDir()
	absBase := dir

	subDir := filepath.Join(dir, "linked")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	w := &walker{
		absBase:        absBase,
		patterns:       []string{"**/*.md"},
		followSymlinks: true,
		seen:           make(map[string]bool),
	}

	info := fakeFileInfo{name: "linked", mode: os.ModeDir | os.ModeSymlink, isDir: true}
	err := w.visit(filepath.Join(absBase, "linked"), info, nil)
	assert.NoError(t, err)
}

func TestVisit_SkipsFileWhenGitignored(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "ignored.md\n")
	writeFile(t, dir, "ignored.md", "# Ignored\n")
	writeFile(t, dir, "kept.md", "# Kept\n")

	files, err := Discover(Options{
		Patterns:     []string{"**/*.md"},
		BaseDir:      dir,
		UseGitignore: true,
	})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "kept.md", filepath.Base(files[0]))
}

func TestVisit_RootDirSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "readme.md", "# Hello\n")

	w := &walker{
		absBase:  dir,
		patterns: []string{"**/*.md"},
		seen:     make(map[string]bool),
	}

	// visit the root itself (rel == ".")
	info, err := os.Stat(dir)
	require.NoError(t, err)
	err = w.visit(dir, info, nil)
	assert.NoError(t, err)
	assert.Empty(t, w.result, "root dir should be skipped")
}

// --- Discover edge cases ---

func TestDiscover_InvalidPatterns(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "readme.md", "# Hello\n")

	// All patterns are invalid, so no files discovered.
	files, err := Discover(Options{
		Patterns: []string{"[invalid"},
		BaseDir:  dir,
	})
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestDiscover_MixedValidAndInvalidPatterns(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "readme.md", "# Hello\n")

	files, err := Discover(Options{
		Patterns: []string{"[invalid", "**/*.md"},
		BaseDir:  dir,
	})
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestDiscover_DefaultBaseDir(t *testing.T) {
	// When BaseDir is empty, it defaults to ".".
	// Just verify it doesn't error out.
	_, err := Discover(Options{
		Patterns: []string{"nonexistent-pattern-xyz-*.md"},
		BaseDir:  "",
	})
	require.NoError(t, err)
}

// TestDiscover_SymlinkToFile_SkippedByDefault asserts the secure
// default: a symlinked file is not discovered.
func TestDiscover_SymlinkToFile_SkippedByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real.md", "# Real\n")

	linkPath := filepath.Join(dir, "link.md")
	realPath := filepath.Join(dir, "real.md")
	require.NoError(t, os.Symlink(realPath, linkPath))

	files, err := Discover(Options{
		Patterns: []string{"**/*.md"},
		BaseDir:  dir,
	})
	require.NoError(t, err)
	require.Len(t, files, 1, "symlink must be skipped by default")
	assert.Equal(t, "real.md", filepath.Base(files[0]))
}

// TestDiscover_FollowSymlinks_OptIn asserts that FollowSymlinks=true
// surfaces the symlink as a distinct discovery result.
func TestDiscover_FollowSymlinks_OptIn(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real.md", "# Real\n")

	linkPath := filepath.Join(dir, "link.md")
	realPath := filepath.Join(dir, "real.md")
	require.NoError(t, os.Symlink(realPath, linkPath))

	files, err := Discover(Options{
		Patterns:       []string{"**/*.md"},
		BaseDir:        dir,
		FollowSymlinks: true,
	})
	require.NoError(t, err)
	require.Len(t, files, 2, "both real and linked entries are discovered")
}

// fakeFileInfo implements os.FileInfo for testing.
type fakeFileInfo struct {
	name  string
	mode  os.FileMode
	isDir bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.isDir }
func (f fakeFileInfo) Sys() any           { return nil }
