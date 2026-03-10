package unclosedcodeblock

import (
	"bytes"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/yuin/goldmark/ast"
)

func init() {
	rule.Register(&Rule{})
}

// Rule detects fenced code blocks that lack a closing fence delimiter.
type Rule struct{}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS031" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "unclosed-code-block" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "code" }

// Check implements rule.Rule.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	var diags []lint.Diagnostic

	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		fcb, ok := n.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		if !hasCLosingFence(f, fcb) {
			openLine := findOpenLine(f, fcb)
			diags = append(diags, lint.Diagnostic{
				File:     f.Path,
				Line:     openLine,
				Column:   1,
				RuleID:   r.ID(),
				RuleName: r.Name(),
				Severity: lint.Error,
				Message:  "unclosed fenced code block",
			})
		}

		return ast.WalkContinue, nil
	})

	return diags
}

// hasCLosingFence checks whether a fenced code block has a proper closing
// fence line after its content.
func hasCLosingFence(f *lint.File, fcb *ast.FencedCodeBlock) bool {
	// Determine the fence character from the opening line.
	openStart := findOpenByteOffset(f, fcb)
	if openStart >= len(f.Source) {
		return true
	}
	fenceChar := fenceCharAt(f.Source, openStart)
	if fenceChar == 0 {
		return true
	}

	// Find the closing fence line position.
	var closingStart int
	segs := fcb.Lines()
	if segs.Len() > 0 {
		lastSeg := segs.At(segs.Len() - 1)
		closingStart = lastSeg.Stop
	} else {
		// Empty code block: closing fence is right after opening fence line.
		closingStart = openStart
		for closingStart < len(f.Source) && f.Source[closingStart] != '\n' {
			closingStart++
		}
		if closingStart < len(f.Source) {
			closingStart++ // skip newline
		}
	}

	// If closing start is at or past EOF, there's no closing fence.
	if closingStart >= len(f.Source) {
		return false
	}

	// Extract the closing line and check if it contains the fence character.
	closingEnd := closingStart
	for closingEnd < len(f.Source) && f.Source[closingEnd] != '\n' {
		closingEnd++
	}
	closingLine := bytes.TrimLeft(f.Source[closingStart:closingEnd], " ")
	minFence := []byte{fenceChar, fenceChar, fenceChar}
	return bytes.HasPrefix(closingLine, minFence)
}

// findOpenByteOffset returns the byte offset of the opening fence line.
func findOpenByteOffset(f *lint.File, fcb *ast.FencedCodeBlock) int {
	if fcb.Info != nil {
		pos := fcb.Info.Segment.Start
		for pos > 0 && f.Source[pos-1] != '\n' {
			pos--
		}
		return pos
	}
	if fcb.Lines().Len() > 0 {
		firstContentStart := fcb.Lines().At(0).Start
		pos := firstContentStart
		if pos > 0 && f.Source[pos-1] == '\n' {
			pos--
		}
		for pos > 0 && f.Source[pos-1] != '\n' {
			pos--
		}
		return pos
	}
	return len(f.Source)
}

// findOpenLine returns the 1-based line number of the opening fence.
func findOpenLine(f *lint.File, fcb *ast.FencedCodeBlock) int {
	offset := findOpenByteOffset(f, fcb)
	if offset >= len(f.Source) {
		return len(f.Lines)
	}
	return f.LineOfOffset(offset)
}

// fenceCharAt returns the fence character at the given position, skipping
// leading spaces.
func fenceCharAt(src []byte, pos int) byte {
	for pos < len(src) && src[pos] == ' ' {
		pos++
	}
	if pos < len(src) && (src[pos] == '`' || src[pos] == '~') {
		return src[pos]
	}
	return 0
}
