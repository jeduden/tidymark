package lint

import (
	"github.com/yuin/goldmark/util"

	"github.com/jeduden/mdsmith/pkg/markdown"
)

// PIBlockParserPrioritized returns the PI parser with its priority
// for registration, forwarded from pkg/markdown. Kept because
// internal/schema registers it directly; the parser logic itself
// lives in pkg/markdown.
func PIBlockParserPrioritized() util.PrioritizedValue {
	return markdown.PIBlockParserPrioritized()
}
