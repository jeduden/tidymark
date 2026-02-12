package catalog

import (
	"github.com/jeduden/tidymark/internal/archetype/gensection"
	"github.com/jeduden/tidymark/internal/lint"
)

// markerPair is an alias for gensection.MarkerPair, kept for
// backward compatibility with internal tests.
type markerPair = gensection.MarkerPair

// extractContent delegates to gensection.ExtractContent.
func extractContent(f *lint.File, mp markerPair) string {
	return gensection.ExtractContent(f, mp)
}

// replaceContent delegates to gensection.ReplaceContent.
func replaceContent(f *lint.File, mp markerPair, content string) []byte {
	return gensection.ReplaceContent(f, mp, content)
}

// splitLines delegates to gensection.SplitLines.
func splitLines(source []byte) [][]byte {
	return gensection.SplitLines(source)
}
