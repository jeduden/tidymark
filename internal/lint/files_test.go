package lint

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestResolveFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := ResolveFiles([]string{mdFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0] != mdFile {
		t.Errorf("expected %q, got %q", mdFile, files[0])
	}
}

func TestResolveFiles_NonMarkdownFile(t *testing.T) {
	dir := t.TempDir()
	txtFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(txtFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Non-markdown files are still returned when given explicitly as args.
	files, err := ResolveFiles([]string{txtFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestResolveFiles_Directory(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create markdown files at various levels.
	for _, name := range []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "b.markdown"),
		filepath.Join(dir, "c.txt"), // should be excluded
		filepath.Join(subDir, "d.md"),
	} {
		if err := os.WriteFile(name, []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := ResolveFiles([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find a.md, b.markdown, sub/d.md (not c.txt).
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(files), files)
	}

	// Check that all returned files are markdown.
	for _, f := range files {
		ext := filepath.Ext(f)
		if ext != ".md" && ext != ".markdown" {
			t.Errorf("unexpected non-markdown file: %s", f)
		}
	}
}

func TestResolveFiles_GlobPattern(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"a.md", "b.md", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	pattern := filepath.Join(dir, "*.md")
	files, err := ResolveFiles([]string{pattern})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
}

func TestResolveFiles_NonexistentPath(t *testing.T) {
	_, err := ResolveFiles([]string{"/nonexistent/path/file.md"})
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestResolveFiles_EmptyArgs(t *testing.T) {
	files, err := ResolveFiles([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestResolveFiles_NilArgs(t *testing.T) {
	files, err := ResolveFiles(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestResolveFiles_Deduplicated(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pass the same file twice.
	files, err := ResolveFiles([]string{mdFile, mdFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file (deduplicated), got %d", len(files))
	}
}

func TestResolveFiles_Sorted(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"z.md", "a.md", "m.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := ResolveFiles([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sort.StringsAreSorted(files) {
		t.Errorf("expected sorted files, got %v", files)
	}
}

func TestResolveFiles_MarkdownExtension(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "doc.markdown")
	if err := os.WriteFile(mdFile, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := ResolveFiles([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if filepath.Ext(files[0]) != ".markdown" {
		t.Errorf("expected .markdown extension, got %s", filepath.Ext(files[0]))
	}
}

func TestResolveFiles_GlobMatchingDirectory(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "guide.md"), []byte("# Guide"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Glob that matches a directory should recurse into it.
	pattern := filepath.Join(dir, "doc*")
	files, err := ResolveFiles([]string{pattern})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
}

// --- Gitignore-aware walking tests ---

func TestResolveFilesWithOpts_GitignoreSkipsMatchedFiles(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "ignored")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create markdown files.
	for _, name := range []string{
		filepath.Join(dir, "keep.md"),
		filepath.Join(subDir, "skip.md"),
	} {
		if err := os.WriteFile(name, []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create .gitignore that excludes the "ignored" directory.
	gitignore := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("ignored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := ResolveFilesWithOpts([]string{dir}, DefaultResolveOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
	if filepath.Base(files[0]) != "keep.md" {
		t.Errorf("expected keep.md, got %s", files[0])
	}
}

func TestResolveFilesWithOpts_NestedGitignore(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create markdown files.
	for _, name := range []string{
		filepath.Join(dir, "root.md"),
		filepath.Join(subDir, "included.md"),
		filepath.Join(subDir, "draft.md"),
	} {
		if err := os.WriteFile(name, []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Nested .gitignore in sub/ excludes draft.md.
	gitignore := filepath.Join(subDir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("draft.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := ResolveFilesWithOpts([]string{dir}, DefaultResolveOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Base(f) == "draft.md" {
			t.Errorf("draft.md should have been excluded by nested .gitignore")
		}
	}
}

func TestResolveFilesWithOpts_UseGitignoreFalse(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "ignored")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		filepath.Join(dir, "keep.md"),
		filepath.Join(subDir, "skip.md"),
	} {
		if err := os.WriteFile(name, []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create .gitignore that would exclude "ignored/".
	gitignore := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("ignored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// With UseGitignore=false, all files should be included.
	f := false
	opts := ResolveOpts{UseGitignore: &f}
	files, err := ResolveFilesWithOpts([]string{dir}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (gitignore disabled), got %d: %v", len(files), files)
	}
}

func TestResolveFilesWithOpts_NoGitignorePresent(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "b.md"),
	} {
		if err := os.WriteFile(name, []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// No .gitignore file present — all markdown files should be included.
	files, err := ResolveFilesWithOpts([]string{dir}, DefaultResolveOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
}

func TestResolveFilesWithOpts_ExplicitFileNotFilteredByGitignore(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "ignored.md")
	if err := os.WriteFile(mdFile, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create .gitignore that excludes *.md.
	gitignore := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("*.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Explicitly named files are NOT filtered by gitignore.
	files, err := ResolveFilesWithOpts([]string{mdFile}, DefaultResolveOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file (explicit path not filtered), got %d: %v", len(files), files)
	}
	if files[0] != mdFile {
		t.Errorf("expected %q, got %q", mdFile, files[0])
	}
}

func TestResolveFilesWithOpts_GitignoreWildcardPattern(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		filepath.Join(dir, "readme.md"),
		filepath.Join(subDir, "output.md"),
	} {
		if err := os.WriteFile(name, []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// .gitignore uses wildcard to exclude build directory.
	gitignore := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := ResolveFilesWithOpts([]string{dir}, DefaultResolveOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
	if filepath.Base(files[0]) != "readme.md" {
		t.Errorf("expected readme.md, got %s", files[0])
	}
}

func TestResolveFilesWithOpts_GitignoreNegation(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "b.md"),
		filepath.Join(dir, "keep.md"),
	} {
		if err := os.WriteFile(name, []byte("# Test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Ignore all .md files, but negate keep.md.
	gitignore := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("*.md\n!keep.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := ResolveFilesWithOpts([]string{dir}, DefaultResolveOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
	if filepath.Base(files[0]) != "keep.md" {
		t.Errorf("expected keep.md, got %s", files[0])
	}
}

// --- NoFollowSymlinks tests ---

func TestResolveFilesWithOpts_NoFollowSymlinks(t *testing.T) {
	dir := t.TempDir()

	// Create a real markdown file.
	realFile := filepath.Join(dir, "real.md")
	if err := os.WriteFile(realFile, []byte("# Real"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a target file in a subdirectory.
	subDir := filepath.Join(dir, "target")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(subDir, "doc.md")
	if err := os.WriteFile(targetFile, []byte("# Target"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the target file.
	linkFile := filepath.Join(dir, "link.md")
	if err := os.Symlink(targetFile, linkFile); err != nil {
		t.Fatal(err)
	}

	// Without no-follow-symlinks: all files should be found.
	files, err := ResolveFilesWithOpts([]string{dir}, DefaultResolveOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 { // real.md, link.md, target/doc.md
		t.Fatalf("expected 3 files, got %d: %v", len(files), files)
	}

	// With no-follow-symlinks matching all .md: symlink should be skipped.
	noGitignore := false
	opts := ResolveOpts{
		UseGitignore:     &noGitignore,
		NoFollowSymlinks: []string{"*.md"},
	}
	files, err = ResolveFilesWithOpts([]string{dir}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// link.md is a symlink matching *.md, should be skipped.
	// real.md and target/doc.md are regular files, should be included.
	if len(files) != 2 {
		t.Fatalf("expected 2 files (symlink skipped), got %d: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Base(f) == "link.md" {
			t.Error("link.md should have been skipped (symlink)")
		}
	}
}

func TestResolveFilesWithOpts_NoFollowSymlinks_PatternSpecific(t *testing.T) {
	dir := t.TempDir()

	// Create target files.
	targetA := filepath.Join(dir, "a.md")
	targetB := filepath.Join(dir, "b.md")
	if err := os.WriteFile(targetA, []byte("# A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetB, []byte("# B"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create subdirectory with symlinks.
	linkDir := filepath.Join(dir, "links")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetA, filepath.Join(linkDir, "link-a.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetB, filepath.Join(linkDir, "link-b.md")); err != nil {
		t.Fatal(err)
	}

	// Only skip symlinks under links/ directory.
	noGitignore := false
	opts := ResolveOpts{
		UseGitignore:     &noGitignore,
		NoFollowSymlinks: []string{"**/links/*"},
	}
	files, err := ResolveFilesWithOpts([]string{dir}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// a.md and b.md are regular files, included.
	// links/link-a.md and links/link-b.md are symlinks matching "**/links/*", skipped.
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
}
