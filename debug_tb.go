package main

import (
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func main() {
	source := []byte(`# Title

Text before.

---

Text after.
`)

	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(source))

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if tb, ok := n.(*ast.ThematicBreak); ok {
				fmt.Printf("ThematicBreak found:\n")
				fmt.Printf("  Lines().Len(): %d\n", tb.Lines().Len())
				fmt.Printf("  Type: %T\n", n)

				// Check if we can find segment info via Dump
				fmt.Printf("  Dumping node:\n")
				n.Dump(source, 2)
			}
		}
		return ast.WalkContinue, nil
	})
}
