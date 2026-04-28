// Package orderedlistnumbering implements MDS046, which pins how
// ordered list items are numbered in the source: literal sequential
// (1. 2. 3.) or all-ones (1. 1. 1.). CommonMark renders both the same,
// but the choice controls what the source text shows authors and
// reviewers.
package orderedlistnumbering

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/jeduden/mdsmith/internal/rules/settings"
	"github.com/yuin/goldmark/ast"
)

// Style values for the rule's `style` setting.
const (
	StyleSequential = "sequential"
	StyleAllOnes    = "all-ones"
)

func init() {
	rule.Register(&Rule{Style: StyleSequential, Start: 1})
}

// Rule pins the numbering style of ordered lists in source.
type Rule struct {
	Style string
	Start int
}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS046" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "ordered-list-numbering" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "list" }

// EnabledByDefault implements rule.Defaultable. The rule is opt-in:
// users pick a project convention and turn the rule on.
func (r *Rule) EnabledByDefault() bool { return false }

// Check implements rule.Rule.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	var diags []lint.Diagnostic
	configStart := r.Start
	if configStart < 0 {
		configStart = 1
	}

	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		list, ok := n.(*ast.List)
		if !ok || !list.IsOrdered() {
			return ast.WalkContinue, nil
		}
		diags = append(diags, r.checkList(f, list, configStart)...)
		return ast.WalkContinue, nil
	})

	return diags
}

// checkList emits diagnostics for one ordered list.
func (r *Rule) checkList(f *lint.File, list *ast.List, configStart int) []lint.Diagnostic {
	firstItem, _ := firstChildListItem(list)
	if firstItem == nil {
		return nil
	}
	firstLine := firstLineOfListItem(f, firstItem)
	if firstLine < 1 {
		return nil
	}

	var diags []lint.Diagnostic
	actualStart := list.Start
	startMismatch := actualStart != configStart
	if startMismatch {
		diags = append(diags, r.diag(f, firstLine, fmt.Sprintf(
			"ordered list starts at %d; configured start is %d",
			actualStart, configStart,
		)))
	}

	i := 0
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		item, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		if d, ok := r.checkItem(f, item, i, actualStart, startMismatch); ok {
			diags = append(diags, d)
		}
		i++
	}
	return diags
}

// checkItem produces a diagnostic for one list item when its literal
// number does not match the expected number under the configured style.
// The first item is suppressed when the list-start mismatch already
// fired there.
func (r *Rule) checkItem(
	f *lint.File, item *ast.ListItem,
	i, baseStart int, startMismatch bool,
) (lint.Diagnostic, bool) {
	line := firstLineOfListItem(f, item)
	if line < 1 || line > len(f.Lines) {
		return lint.Diagnostic{}, false
	}
	literal, _, _, _, parseOK := parseListItemNumber(f.Lines[line-1])
	if !parseOK {
		return lint.Diagnostic{}, false
	}
	expected := expectedNumber(r.Style, baseStart, i)
	if literal == expected {
		return lint.Diagnostic{}, false
	}
	if i == 0 && startMismatch {
		return lint.Diagnostic{}, false
	}
	return r.diag(f, line, fmt.Sprintf(
		"ordered list item %d numbered %d; expected %d",
		i+1, literal, expected,
	)), true
}

func (r *Rule) diag(f *lint.File, line int, msg string) lint.Diagnostic {
	return lint.Diagnostic{
		File:     f.Path,
		Line:     line,
		Column:   1,
		RuleID:   r.ID(),
		RuleName: r.Name(),
		Severity: lint.Warning,
		Message:  msg,
	}
}

// markerEdit replaces the literal number at a marker line.
type markerEdit struct {
	newDigits int
}

// Fix implements rule.FixableRule.
func (r *Rule) Fix(f *lint.File) []byte {
	configStart := r.Start
	if configStart < 0 {
		configStart = 1
	}

	markerEdits := map[int]markerEdit{}
	indentDeltas := map[int]int{}

	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		list, ok := n.(*ast.List)
		if !ok || !list.IsOrdered() {
			return ast.WalkContinue, nil
		}
		r.collectListEdits(f, list, configStart, markerEdits, indentDeltas)
		return ast.WalkContinue, nil
	})

	if len(markerEdits) == 0 && len(indentDeltas) == 0 {
		out := make([]byte, len(f.Source))
		copy(out, f.Source)
		return out
	}

	resultLines := make([][]byte, len(f.Lines))
	for i, line := range f.Lines {
		lineNum := i + 1
		newLine := append([]byte(nil), line...)
		newLine = applyIndentShift(newLine, indentDeltas[lineNum])
		if e, ok := markerEdits[lineNum]; ok {
			newLine = replaceLeadingDigits(newLine, e.newDigits)
		}
		resultLines[i] = newLine
	}

	return bytes.Join(resultLines, []byte("\n"))
}

// collectListEdits records marker rewrites and continuation-indent
// shifts for one ordered list.
func (r *Rule) collectListEdits(
	f *lint.File, list *ast.List, configStart int,
	markerEdits map[int]markerEdit, indentDeltas map[int]int,
) {
	i := 0
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		item, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		r.collectItemEdits(f, item, i, configStart, markerEdits, indentDeltas)
		i++
	}
}

// collectItemEdits records the marker rewrite and the indent delta on
// continuation lines for one list item.
func (r *Rule) collectItemEdits(
	f *lint.File, item *ast.ListItem, i, configStart int,
	markerEdits map[int]markerEdit, indentDeltas map[int]int,
) {
	line := firstLineOfListItem(f, item)
	if line < 1 || line > len(f.Lines) {
		return
	}
	literal, _, _, _, parseOK := parseListItemNumber(f.Lines[line-1])
	if !parseOK {
		return
	}
	expected := expectedNumber(r.Style, configStart, i)
	if literal != expected {
		markerEdits[line] = markerEdit{newDigits: expected}
	}
	delta := digitWidth(expected) - digitWidth(literal)
	if delta == 0 {
		return
	}
	lastLine := lastLineOfNode(f, item)
	if lastLine < line {
		lastLine = line
	}
	for ln := line + 1; ln <= lastLine; ln++ {
		indentDeltas[ln] += delta
	}
}

// applyIndentShift adjusts the leading-whitespace width of a line by
// shift bytes. Negative shifts that exceed the existing leading
// whitespace are ignored to avoid eating non-space content.
func applyIndentShift(line []byte, shift int) []byte {
	if shift == 0 {
		return line
	}
	if shift > 0 {
		pad := bytes.Repeat([]byte(" "), shift)
		return append(pad, line...)
	}
	n := -shift
	if countLeadingSpaces(line) < n {
		return line
	}
	return line[n:]
}

// replaceLeadingDigits replaces a run of digits at the start of a line
// (after any leading whitespace) with the decimal form of n.
func replaceLeadingDigits(line []byte, n int) []byte {
	leading := countLeadingSpaces(line)
	digitStart := leading
	digitEnd := leading
	for digitEnd < len(line) && isDigit(line[digitEnd]) {
		digitEnd++
	}
	if digitEnd == digitStart {
		return line
	}
	out := make([]byte, 0, len(line)+8)
	out = append(out, line[:digitStart]...)
	out = append(out, []byte(strconv.Itoa(n))...)
	out = append(out, line[digitEnd:]...)
	return out
}

// expectedNumber returns the number an item at index i (0-based) should
// have, given the rule's style and the list's start value.
func expectedNumber(style string, start, i int) int {
	if style == StyleAllOnes {
		return start
	}
	return start + i
}

// parseListItemNumber finds the literal number on a list-item line.
// Returns the number, the byte indices of the digits within the line,
// and the marker character ('.' or ')'). ok is false when the line
// does not begin with an ordered-list marker.
func parseListItemNumber(line []byte) (number int, digitStart, digitEnd int, markerChar byte, ok bool) {
	i := 0
	for i < len(line) && line[i] == ' ' {
		i++
	}
	digitStart = i
	for i < len(line) && isDigit(line[i]) {
		number = number*10 + int(line[i]-'0')
		i++
	}
	digitEnd = i
	if digitEnd == digitStart {
		return 0, 0, 0, 0, false
	}
	if i >= len(line) {
		return 0, 0, 0, 0, false
	}
	if line[i] != '.' && line[i] != ')' {
		return 0, 0, 0, 0, false
	}
	markerChar = line[i]
	return number, digitStart, digitEnd, markerChar, true
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func digitWidth(n int) int {
	if n == 0 {
		return 1
	}
	w := 0
	if n < 0 {
		w = 1
		n = -n
	}
	for n > 0 {
		n /= 10
		w++
	}
	return w
}

func countLeadingSpaces(line []byte) int {
	n := 0
	for _, b := range line {
		if b == ' ' {
			n++
			continue
		}
		break
	}
	return n
}

func firstChildListItem(list *ast.List) (*ast.ListItem, int) {
	idx := 0
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		if li, ok := c.(*ast.ListItem); ok {
			return li, idx
		}
		idx++
	}
	return nil, 0
}

func firstLineOfListItem(f *lint.File, li *ast.ListItem) int {
	if li.Lines().Len() > 0 {
		return f.LineOfOffset(li.Lines().At(0).Start)
	}
	if li.HasChildren() {
		for c := li.FirstChild(); c != nil; c = c.NextSibling() {
			line := lineOfNode(f, c)
			if line > 0 {
				return line
			}
		}
	}
	return 0
}

func lineOfNode(f *lint.File, n ast.Node) int {
	if t, ok := n.(*ast.Text); ok {
		return f.LineOfOffset(t.Segment.Start)
	}
	if isInlineNode(n) {
		if n.HasChildren() {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if l := lineOfNode(f, c); l > 0 {
					return l
				}
			}
		}
		return 0
	}
	if n.Lines().Len() > 0 {
		return f.LineOfOffset(n.Lines().At(0).Start)
	}
	if n.HasChildren() {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if l := lineOfNode(f, c); l > 0 {
				return l
			}
		}
	}
	return 0
}

func lastLineOfNode(f *lint.File, n ast.Node) int {
	if t, ok := n.(*ast.Text); ok {
		stop := t.Segment.Stop
		if stop > 0 {
			stop--
		}
		return f.LineOfOffset(stop)
	}
	if isInlineNode(n) {
		if n.HasChildren() {
			for c := n.LastChild(); c != nil; c = c.PreviousSibling() {
				if l := lastLineOfNode(f, c); l > 0 {
					return l
				}
			}
		}
		return 0
	}
	last := 0
	if n.Lines().Len() > 0 {
		seg := n.Lines().At(n.Lines().Len() - 1)
		last = f.LineOfOffset(seg.Start)
	}
	if n.HasChildren() {
		for c := n.LastChild(); c != nil; c = c.PreviousSibling() {
			if l := lastLineOfNode(f, c); l > last {
				last = l
			}
		}
	}
	return last
}

func isInlineNode(n ast.Node) bool {
	switch n.(type) {
	case *ast.Text, *ast.String, *ast.CodeSpan, *ast.Emphasis,
		*ast.Link, *ast.Image, *ast.AutoLink, *ast.RawHTML:
		return true
	}
	return false
}

// ApplySettings implements rule.Configurable.
func (r *Rule) ApplySettings(s map[string]any) error {
	for k, v := range s {
		switch k {
		case "style":
			str, ok := v.(string)
			if !ok {
				return fmt.Errorf("ordered-list-numbering: style must be a string, got %T", v)
			}
			if str != StyleSequential && str != StyleAllOnes {
				return fmt.Errorf("ordered-list-numbering: invalid style %q (valid: sequential, all-ones)", str)
			}
			r.Style = str
		case "start":
			n, ok := settings.ToInt(v)
			if !ok {
				return fmt.Errorf("ordered-list-numbering: start must be an integer, got %T", v)
			}
			if n < 0 {
				return fmt.Errorf("ordered-list-numbering: start must be non-negative, got %d", n)
			}
			r.Start = n
		default:
			return fmt.Errorf("ordered-list-numbering: unknown setting %q", k)
		}
	}
	return nil
}

// DefaultSettings implements rule.Configurable.
func (r *Rule) DefaultSettings() map[string]any {
	return map[string]any{
		"style": StyleSequential,
		"start": 1,
	}
}

var (
	_ rule.Configurable = (*Rule)(nil)
	_ rule.FixableRule  = (*Rule)(nil)
	_ rule.Defaultable  = (*Rule)(nil)
)
