// Package globpath provides the shared glob matcher used across all mdsmith
// surfaces: config (ignore:, overrides:, kind-assignment:), the catalog
// directive, and CLI argument expansion.
//
// All three surfaces use the same doublestar matcher so that **, brace
// expansion, and !-prefix exclusion behave identically.
package globpath

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Match reports whether path matches pattern using the doublestar matcher.
// It checks the raw path, the cleaned path, and the base name so that
// patterns without path separators (e.g. "slides.md") match files in any
// directory.
// Invalid patterns return false.
func Match(pattern, path string) bool {
	cleanPath := filepath.Clean(path)
	base := filepath.Base(path)

	for _, candidate := range []string{path, cleanPath, base} {
		if ok, err := doublestar.Match(
			filepath.ToSlash(pattern),
			filepath.ToSlash(candidate),
		); err == nil && ok {
			return true
		}
	}
	return false
}

// MatchAny reports whether path matches any of the given patterns.
// A pattern prefixed with "!" is an exclusion pattern. The path matches
// when at least one non-negated pattern matches and no exclusion pattern
// matches; an exclusion pattern always wins over an inclusion pattern,
// regardless of list order. A list containing only exclusion patterns
// matches nothing.
func MatchAny(patterns []string, path string) bool {
	matchedInclude := false
	for _, pattern := range patterns {
		isExclude := strings.HasPrefix(pattern, "!")
		if isExclude {
			pattern = pattern[1:]
		}
		if !Match(pattern, path) {
			continue
		}
		if isExclude {
			return false
		}
		matchedInclude = true
	}
	return matchedInclude
}

// SplitIncludeExclude separates patterns into include and exclude lists.
// Patterns prefixed with "!" are exclusion patterns (the prefix is stripped).
func SplitIncludeExclude(patterns []string) (include, exclude []string) {
	for _, p := range patterns {
		if strings.HasPrefix(p, "!") {
			exclude = append(exclude, p[1:])
		} else {
			include = append(include, p)
		}
	}
	return include, exclude
}
