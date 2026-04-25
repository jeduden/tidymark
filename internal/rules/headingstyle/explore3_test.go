package headingstyle

import (
	"fmt"
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestExploreHeadingWithManualChildren(t *testing.T) {
	// Try to craft a heading with Lines().Len() == 0 but with text children
	// by creating one manually and calling headingLine

	src := []byte("# Title\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	_ = f

	// Create a heading with no lines but manually attach a text child
	h := ast.NewHeading(1)
	textNode := ast.NewText()
	textNode.Segment = text.NewSegment(2, 7) // "Title" in "# Title\n"
	h.AppendChild(h, textNode)

	fmt.Printf("Manual heading: Lines=%d, FirstChild=%T\n", h.Lines().Len(), h.FirstChild())
	line := headingLine(h, f)
	fmt.Printf("headingLine returned: %d\n", line)
}

func TestExploreHeadingWithNonTextChild(t *testing.T) {
	// Heading with Lines=0 and first child is Emphasis (not Text), wrapping Text
	src := []byte("# **bold**\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	// Create heading manually
	h := ast.NewHeading(1)
	em := ast.NewEmphasis(2)
	textNode := ast.NewText()
	textNode.Segment = text.NewSegment(3, 7) // "bold" in "# **bold**\n" -- rough
	em.AppendChild(em, textNode)
	h.AppendChild(h, em)

	fmt.Printf("Manual heading with emphasis: Lines=%d\n", h.Lines().Len())
	line := headingLine(h, f)
	fmt.Printf("headingLine returned: %d (expected 1 for offset 3)\n", line)
}
