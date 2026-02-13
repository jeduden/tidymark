// Package discovery finds Markdown files by expanding glob patterns from config.
package discovery

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/jeduden/tidymark/internal/lint"
)

// Options controls how file discovery behaves.
type Options struct {
	// Patterns is the list of glob patterns to match files against.
	// An empty or nil list means no files are discovered.
	Patterns []string

	// BaseDir is the directory to walk from. Defaults to "." if empty.
	BaseDir string

	// UseGitignore enables filtering by .gitignore rules.
	UseGitignore bool
}

// Discover walks BaseDir and returns files matching any of the configured
// glob patterns. Results are deduplicated and sorted.
func Discover(opts Options) ([]string, error) {
	if len(opts.Patterns) == 0 {
		return nil, nil
	}

	baseDir := opts.BaseDir
	if baseDir == "" {
		baseDir = "."
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}

	validPatterns := validatePatterns(opts.Patterns)
	if len(validPatterns) == 0 {
		return nil, nil
	}

	var gitMatcher *lint.GitignoreMatcher
	if opts.UseGitignore {
		gitMatcher = lint.NewGitignoreMatcher(baseDir)
	}

	w := &walker{
		absBase:  absBase,
		patterns: validPatterns,
		git:      gitMatcher,
		seen:     make(map[string]bool),
	}

	if err := filepath.Walk(absBase, w.visit); err != nil {
		return nil, err
	}

	sort.Strings(w.result)
	return w.result, nil
}

// validatePatterns returns patterns that are syntactically valid.
func validatePatterns(patterns []string) []string {
	valid := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if doublestar.ValidatePattern(p) {
			valid = append(valid, p)
		}
	}
	return valid
}

// walker holds state for the directory walk.
type walker struct {
	absBase  string
	patterns []string
	git      *lint.GitignoreMatcher
	seen     map[string]bool
	result   []string
}

// visit is the filepath.WalkFunc callback.
func (w *walker) visit(path string, info os.FileInfo, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}

	rel, err := filepath.Rel(w.absBase, path)
	if err != nil || rel == "." {
		return nil
	}
	rel = filepath.ToSlash(rel)

	if w.isGitignored(path, info) {
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}

	if info.IsDir() {
		return nil
	}

	if w.matchesAny(rel) {
		w.addFile(path)
	}
	return nil
}

// isGitignored returns true if the path should be skipped by .gitignore rules.
func (w *walker) isGitignored(path string, info os.FileInfo) bool {
	if w.git == nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return w.git.IsIgnored(absPath, info.IsDir())
}

// matchesAny returns true if rel matches any of the configured patterns.
func (w *walker) matchesAny(rel string) bool {
	for _, p := range w.patterns {
		matched, err := doublestar.Match(p, rel)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// addFile adds a file to the result set if not already seen.
func (w *walker) addFile(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	if !w.seen[absPath] {
		w.seen[absPath] = true
		w.result = append(w.result, path)
	}
}
