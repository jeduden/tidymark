package nospaceinlinktext

import (
	"fmt"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/yuin/goldmark/ast"
)

func init() {
	rule.Register(&Rule{CheckImages: true})
}

// Rule implements MDS049, flagging Markdown links and images whose visible
// text has leading or trailing whitespace inside the brackets.
type Rule struct {
	CheckImages bool
}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS049" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "no-space-in-link-text" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "link" }

// EnabledByDefault implements rule.Defaultable. MDS049 is opt-in.
func (r *Rule) EnabledByDefault() bool { return false }

// ApplySettings implements rule.Configurable.
func (r *Rule) ApplySettings(settings map[string]any) error {
	for k, v := range settings {
		switch k {
		case "check-images":
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("no-space-in-link-text: check-images must be a bool, got %T", v)
			}
			r.CheckImages = b
		default:
			return fmt.Errorf("no-space-in-link-text: unknown setting %q", k)
		}
	}
	return nil
}

// DefaultSettings implements rule.Configurable.
func (r *Rule) DefaultSettings() map[string]any {
	return map[string]any{
		"check-images": true,
	}
}

// bracketSpan returns the byte range of the opening `[` and its matching `]`
// for a link or image node. Returns (-1, -1) if the span cannot be determined.
//
// It walks all descendant Text nodes to find the earliest source position, then
// scans backward for `[` (handling emphasis, code, and other inline markers that
// may precede the first text). The forward scan uses depth tracking and skips
// backslash-escaped brackets to find the correct closing `]`.
func bracketSpan(n ast.Node, source []byte) (open, close int) {
	// Find the earliest byte position among all descendant Text nodes, but
	// skip Image subtrees. Walking all descendants (not just direct children)
	// handles cases where the first inline node is emphasis or a code span.
	// Skipping Image subtrees prevents a link wrapping an image (e.g.
	// [![alt](img)](url)) from picking up text inside the image's alt span
	// and scanning to the wrong `[`.
	minStart := -1
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if child == n {
			return ast.WalkContinue, nil
		}
		if _, ok := child.(*ast.Image); ok {
			return ast.WalkSkipChildren, nil
		}
		t, ok := child.(*ast.Text)
		if !ok {
			return ast.WalkContinue, nil
		}
		if minStart == -1 || t.Segment.Start < minStart {
			minStart = t.Segment.Start
		}
		return ast.WalkContinue, nil
	})

	if minStart == -1 {
		return -1, -1
	}

	// Scan backward from the first content byte to find the opening `[`.
	// Any inline formatting bytes (*, _, `, etc.) between `[` and the first
	// text are passed over. A `[` preceded by an odd number of backslashes is
	// an escaped bracket (\[) and is skipped so we land on the real opener.
	openBracket := -1
	for i := minStart - 1; i >= 0; i-- {
		if source[i] != '[' {
			continue
		}
		// Count backslashes immediately before this `[`.
		backslashes := 0
		for j := i - 1; j >= 0 && source[j] == '\\'; j-- {
			backslashes++
		}
		if backslashes%2 == 1 {
			// Odd count → this `[` is escaped; skip.
			continue
		}
		openBracket = i
		break
	}
	if openBracket == -1 {
		return -1, -1
	}

	// Scan forward from openBracket+1 to find the matching `]`, tracking
	// bracket depth. Backslash-escaped brackets are skipped so that `\]`
	// inside link text does not terminate the scan prematurely.
	depth := 1
	i := openBracket + 1
	for i < len(source) && depth > 0 {
		if source[i] == '\\' && i+1 < len(source) {
			i += 2
			continue
		}
		switch source[i] {
		case '[':
			depth++
		case ']':
			depth--
		}
		i++
	}
	if depth != 0 {
		return -1, -1
	}
	// i now points one past the `]`.
	closeBracket := i - 1
	return openBracket, closeBracket
}

// isImage reports whether n is an *ast.Image.
func isImage(n ast.Node) bool {
	_, ok := n.(*ast.Image)
	return ok
}

type span struct {
	open  int
	close int
	img   bool
}

// collectSpans walks the AST and returns all bracket spans needing inspection.
func (r *Rule) collectSpans(f *lint.File) []span {
	var spans []span
	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *ast.Link, *ast.Image:
		default:
			return ast.WalkContinue, nil
		}
		img := isImage(n)
		if img && !r.CheckImages {
			return ast.WalkContinue, nil
		}
		open, close := bracketSpan(n, f.Source)
		if open == -1 {
			return ast.WalkContinue, nil
		}
		spans = append(spans, span{open: open, close: close, img: img})
		return ast.WalkContinue, nil
	})
	return spans
}

// Check implements rule.Rule.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	var diags []lint.Diagnostic
	for _, s := range r.collectSpans(f) {
		// Text inside brackets is source[s.open+1 : s.close].
		inner := f.Source[s.open+1 : s.close]
		if len(inner) == 0 {
			continue
		}
		role := "link text"
		if s.img {
			role = "image alt text"
		}
		first := inner[0]
		last := inner[len(inner)-1]
		// Only flag space/tab, not newlines.
		if first == ' ' || first == '\t' {
			diags = append(diags, lint.Diagnostic{
				File:     f.Path,
				Line:     f.LineOfOffset(s.open),
				Column:   f.ColumnOfOffset(s.open),
				RuleID:   r.ID(),
				RuleName: r.Name(),
				Severity: lint.Warning,
				Message:  role + " has leading whitespace",
			})
		}
		if last == ' ' || last == '\t' {
			diags = append(diags, lint.Diagnostic{
				File:     f.Path,
				Line:     f.LineOfOffset(s.close - 1),
				Column:   f.ColumnOfOffset(s.close - 1),
				RuleID:   r.ID(),
				RuleName: r.Name(),
				Severity: lint.Warning,
				Message:  role + " has trailing whitespace",
			})
		}
	}
	return diags
}

// Fix implements rule.FixableRule. Trims leading/trailing space/tab inside
// each bracket pair while leaving the surrounding markdown structure intact.
func (r *Rule) Fix(f *lint.File) []byte {
	type replacement struct {
		open  int
		close int
		text  []byte
	}

	var reps []replacement
	for _, s := range r.collectSpans(f) {
		inner := f.Source[s.open+1 : s.close]
		if len(inner) == 0 {
			continue
		}
		trimmed := trimSpaceTab(inner)
		if len(trimmed) == len(inner) {
			continue
		}
		reps = append(reps, replacement{
			open:  s.open + 1,
			close: s.close,
			text:  trimmed,
		})
	}

	if len(reps) == 0 {
		result := make([]byte, len(f.Source))
		copy(result, f.Source)
		return result
	}

	var result []byte
	prev := 0
	for _, rep := range reps {
		if rep.open < prev {
			// This span is nested inside a previously fixed span.
			// Skip it — a subsequent fix pass will address it.
			continue
		}
		result = append(result, f.Source[prev:rep.open]...)
		result = append(result, rep.text...)
		prev = rep.close
	}
	result = append(result, f.Source[prev:]...)
	return result
}

// trimSpaceTab trims leading and trailing space/tab bytes (not newlines).
func trimSpaceTab(b []byte) []byte {
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\t') {
		start++
	}
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}

var (
	_ rule.Configurable = (*Rule)(nil)
	_ rule.Defaultable  = (*Rule)(nil)
	_ rule.FixableRule  = (*Rule)(nil)
)
