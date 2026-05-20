//go:build mdtext_punkt_upstream

package mdtext

import (
	sentlib "github.com/neurosnap/sentences"
	"github.com/neurosnap/sentences/english"
)

// buildTokenizer returns the upstream `english.NewSentenceTokenizer`
// path, with no override of the abbreviation classifier. This is the
// A/B verification path for plan 191: build with
// `-tags mdtext_punkt_upstream` to fall back to the upstream code and
// confirm that the fast-path output matches.
//
// The upstream constructor's data-load error is swallowed to
// preserve the original `t, _ := english.NewSentenceTokenizer(nil)`
// shape this file is meant to verify against. The default build
// (fastpunct_init.go) panics via mustLoadTraining instead — the
// two paths differ on error handling, but the verification this
// file backs is segmentation behaviour on valid input.
func buildTokenizer() *sentlib.DefaultSentenceTokenizer {
	t, _ := english.NewSentenceTokenizer(nil)
	return t
}
