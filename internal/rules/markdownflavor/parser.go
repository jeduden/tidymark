package markdownflavor

import (
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

var (
	parserOnce sync.Once
	parserMD   goldmark.Markdown
)

// Parser returns the shared goldmark parser used for dual parsing.
// It enables every built-in goldmark extension MDS034 actually needs
// for AST-based feature detection (table, strikethrough, task list,
// footnote, definition list) and the heading-ID attribute parser.
//
// Linkify is intentionally not enabled here: bare-URL autolinks are
// detected separately in detectBareURLs by scanning Text nodes from
// the main CommonMark parse, so adding Linkify would only duplicate
// work without changing the result.
//
// The parser is detection-only: we never render its output. Storing
// it as a package-level singleton avoids rebuilding the parser on
// every rule clone.
func Parser() goldmark.Markdown {
	parserOnce.Do(func() {
		parserMD = goldmark.New(
			goldmark.WithExtensions(
				extension.Table,
				extension.Strikethrough,
				extension.TaskList,
				extension.Footnote,
				extension.DefinitionList,
			),
			goldmark.WithParserOptions(
				parser.WithAttribute(),
			),
		)
	})
	return parserMD
}
