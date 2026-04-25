package markdownflavor

import (
	"sort"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rules/markdownflavor/ext"
)

// edit describes a single byte-range substitution to apply to f.Source.
// All edits are computed against the original source; applyEdits sorts
// them in reverse and rewrites the buffer once.
type edit struct {
	start, end int
	repl       []byte
}

// fixByteRangeFeatures collects edits for the six byte-range features
// (heading IDs, strikethrough, task lists, superscript, subscript, and
// bare-URL autolinks) and returns the rewritten source. Features that
// the configured flavor accepts are skipped. The function returns the
// original source unchanged when no edit applies.
func (r *Rule) fixByteRangeFeatures(f *lint.File) []byte {
	var edits []edit

	// Heading IDs, strikethrough, task lists, superscript, subscript
	// are detected via the dual parser tree.
	if r.needsAnyDualFix() {
		doc := Parser().Parser().Parse(text.NewReader(f.Source))
		_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			edits = append(edits, r.dualNodeEdits(f, n)...)
			return ast.WalkContinue, nil
		})
	}

	// Bare URLs are detected on the main CommonMark AST so URLs inside
	// code spans / fences are skipped (insideNonBareContext).
	if !r.Flavor.Supports(FeatureBareURLAutolinks) {
		for _, fin := range detectBareURLs(f) {
			edits = append(edits, wrapBareURL(f.Source, fin))
		}
	}

	if len(edits) == 0 {
		return f.Source
	}
	return applyEdits(f.Source, edits)
}

// needsAnyDualFix reports whether any of the dual-parser fixable
// features is unsupported by the configured flavor. Used to skip the
// dual re-parse when there is nothing to do.
func (r *Rule) needsAnyDualFix() bool {
	for _, feat := range []Feature{
		FeatureHeadingIDs, FeatureStrikethrough, FeatureTaskLists,
		FeatureSuperscript, FeatureSubscript,
	} {
		if !r.Flavor.Supports(feat) {
			return true
		}
	}
	return false
}

// dualNodeEdits returns the edits to remove an unsupported feature
// produced from a dual-parser AST node. Returns nil when the node is
// either supported or not a fixable feature.
func (r *Rule) dualNodeEdits(f *lint.File, n ast.Node) []edit {
	switch node := n.(type) {
	case *ast.Heading:
		if r.Flavor.Supports(FeatureHeadingIDs) {
			return nil
		}
		return headingIDEdits(f, node)
	case *extast.Strikethrough:
		if r.Flavor.Supports(FeatureStrikethrough) {
			return nil
		}
		return delimiterPairEdits(f.Source, node, "~~")
	case *extast.TaskCheckBox:
		if r.Flavor.Supports(FeatureTaskLists) {
			return nil
		}
		return taskCheckBoxEdits(f, node)
	case *ext.SuperscriptNode:
		if r.Flavor.Supports(FeatureSuperscript) {
			return nil
		}
		return delimiterPairEdits(f.Source, node, "^")
	case *ext.SubscriptNode:
		if r.Flavor.Supports(FeatureSubscript) {
			return nil
		}
		return delimiterPairEdits(f.Source, node, "~")
	}
	return nil
}

// headingIDEdits returns the edit that drops a "{#id}" attribute block
// plus any whitespace separating it from the heading text. Returns nil
// when the heading carries no id attribute.
func headingIDEdits(f *lint.File, h *ast.Heading) []edit {
	fin, ok := findHeadingID(f, h)
	if !ok {
		return nil
	}
	hx, ok := fin.Extra.(HeadingIDExtra)
	if !ok {
		return nil
	}
	start := hx.AttrStart
	for start > 0 && (f.Source[start-1] == ' ' || f.Source[start-1] == '\t') {
		start--
	}
	return []edit{{start: start, end: hx.AttrEnd, repl: nil}}
}

// delimiterPairEdits returns edits removing the opening and closing
// delimiter runs that wrap an inline node. The function locates each
// delimiter from the inner-content span derived from descendant text
// segments — the dual parser does not record segment offsets on the
// wrapper node itself.
func delimiterPairEdits(source []byte, n ast.Node, marker string) []edit {
	innerStart, innerEnd, ok := innerSpan(n)
	if !ok {
		return nil
	}
	openStart := innerStart - len(marker)
	closeEnd := innerEnd + len(marker)
	if openStart < 0 || closeEnd > len(source) {
		return nil
	}
	if string(source[openStart:innerStart]) != marker {
		return nil
	}
	if string(source[innerEnd:closeEnd]) != marker {
		return nil
	}
	return []edit{
		{start: openStart, end: innerStart, repl: nil},
		{start: innerEnd, end: closeEnd, repl: nil},
	}
}

// innerSpan returns the byte range covered by descendant text segments.
// The bounds are conservative: spans for nested inline nodes are
// gathered by walking, so a span like `~~before *bold* after~~` reports
// the range from the start of "before" to the end of "after". Returns
// false when the node carries no descendant text.
func innerSpan(n ast.Node) (int, int, bool) {
	start := -1
	end := -1
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		s, e, ok := childTextRange(c)
		if !ok {
			continue
		}
		if start == -1 || s < start {
			start = s
		}
		if e > end {
			end = e
		}
	}
	if start == -1 {
		return 0, 0, false
	}
	return start, end, true
}

// childTextRange returns the [start, stop) byte range covered by a
// child inline node. ast.Text exposes its segment directly; container
// inlines (emphasis, strong, code spans inside content, etc.) report
// the union of their descendants' spans.
func childTextRange(n ast.Node) (int, int, bool) {
	if t, ok := n.(*ast.Text); ok {
		return t.Segment.Start, t.Segment.Stop, true
	}
	s, e, ok := innerSpan(n)
	return s, e, ok
}

// taskCheckBoxEdits removes the "[X]" run plus a single trailing
// space when present. Per the plan, the bullet itself is preserved.
func taskCheckBoxEdits(f *lint.File, n *extast.TaskCheckBox) []edit {
	block := nearestBlockAncestor(n)
	if block == nil {
		return nil
	}
	lines := block.Lines()
	if lines == nil || lines.Len() == 0 {
		return nil
	}
	start := lines.At(0).Start
	if start+3 > len(f.Source) {
		return nil
	}
	if f.Source[start] != '[' || f.Source[start+2] != ']' {
		return nil
	}
	end := start + 3
	if end < len(f.Source) && f.Source[end] == ' ' {
		end++
	}
	return []edit{{start: start, end: end, repl: nil}}
}

// wrapBareURL wraps a bare URL in angle brackets so the renderer treats
// it as a CommonMark autolink. The detector reports a precise span via
// fin.Start / fin.End.
func wrapBareURL(source []byte, fin Finding) edit {
	url := source[fin.Start:fin.End]
	repl := make([]byte, 0, len(url)+2)
	repl = append(repl, '<')
	repl = append(repl, url...)
	repl = append(repl, '>')
	return edit{start: fin.Start, end: fin.End, repl: repl}
}

// applyEdits rewrites src by applying every edit. Edits are sorted in
// descending start order so each rewrite leaves earlier offsets valid.
// Overlapping edits are dropped silently (last writer wins): the
// detection layer never produces overlaps for the features we fix.
func applyEdits(src []byte, edits []edit) []byte {
	sort.SliceStable(edits, func(i, j int) bool {
		return edits[i].start > edits[j].start
	})
	out := make([]byte, len(src))
	copy(out, src)
	prevStart := len(src) + 1
	for _, e := range edits {
		if e.end > prevStart {
			continue
		}
		out = append(append(append([]byte(nil), out[:e.start]...), e.repl...), out[e.end:]...)
		prevStart = e.start
	}
	return out
}
