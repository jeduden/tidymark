//go:build !mdtext_punkt_upstream

package mdtext

import (
	sentlib "github.com/neurosnap/sentences"
	"github.com/neurosnap/sentences/data"
	"github.com/neurosnap/sentences/english"
)

// buildTokenizer assembles the same DefaultSentenceTokenizer that
// `english.NewSentenceTokenizer(nil)` would build — the same trained
// English data, the same word tokenizer, the same supervised
// abbreviations — but replaces the third-pass
// MultiPunctWordAnnotation with fastMultiPunctWordAnnotation. The
// only call-site difference is that the abbreviation classifier
// runs matchAbbrPattern in place of `reAbbr.FindAllString`. See
// `english/main.go:NewSentenceTokenizer` for the upstream original
// and plan 191 for the rationale.
//
// `data.MustAsset` panics if the bundled English Punkt data is
// missing — an invariant we trust at build time and which cannot
// be driven red/green from a test. `sentlib.LoadTraining` returns
// (nil, err) only on malformed JSON; the bundled file is fixed at
// build time, so any error here is also a build-time invariant
// violation. Swallowing it with `_` matches upstream's
// `t, _ := english.NewSentenceTokenizer(nil)` swallow: training
// stays nil, downstream panics on first use — same failure mode.
func buildTokenizer() *sentlib.DefaultSentenceTokenizer {
	raw := data.MustAsset("data/english.json")
	training, _ := sentlib.LoadTraining(raw)

	// Supervised abbreviations applied by english.NewSentenceTokenizer.
	for _, abbr := range []string{"sgt", "gov", "no"} {
		training.AbbrevTypes.Add(abbr)
	}

	lang := sentlib.NewPunctStrings()
	word := english.NewWordTokenizer(lang)

	annotations := sentlib.NewAnnotations(training, lang, word)

	ortho := &sentlib.OrthoContext{
		Storage:      training,
		PunctStrings: lang,
		TokenType:    word,
		TokenFirst:   word,
	}

	fastMulti := &fastMultiPunctWordAnnotation{
		Storage:      training,
		TokenParser:  word,
		TokenGrouper: &sentlib.DefaultTokenGrouper{},
		Ortho:        ortho,
		upstreamWord: word,
	}
	annotations = append(annotations, fastMulti)

	return &sentlib.DefaultSentenceTokenizer{
		Storage:       training,
		PunctStrings:  lang,
		WordTokenizer: word,
		Annotations:   annotations,
	}
}
