package markdownflavor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"

	"github.com/jeduden/mdsmith/internal/lint"
)

// TestLineColClampsNegativeOffset exercises the guard that clamps a
// negative offset to 0 so callers that subtract past the start of
// f.Source still get a valid (1, 1) position.
func TestLineColClampsNegativeOffset(t *testing.T) {
	line, col := lineCol([]byte("hello\nworld\n"), -5)
	assert.Equal(t, 1, line)
	assert.Equal(t, 1, col)
}

// TestLineColClampsOversizedOffset exercises the guard that clamps
// an offset past len(source) back to len(source) so callers that
// look one byte past EOF still get a valid position.
func TestLineColClampsOversizedOffset(t *testing.T) {
	src := []byte("hello\nworld\n")
	line, col := lineCol(src, len(src)+10)
	assert.Equal(t, 3, line)
	assert.Equal(t, 1, col)
}

// TestLineStartOfClampsOversizedOffset mirrors the same clamp for
// lineStartOf. An offset past EOF clamps to len(source); for a file
// ending in a newline that puts us one byte past the last newline,
// which is the start of the (empty) line after the document.
func TestLineStartOfClampsOversizedOffset(t *testing.T) {
	src := []byte("hello\nworld\n")
	assert.Equal(t, len(src), lineStartOf(src, len(src)+10))
}

// TestLineStartOfMidLine returns the first byte of the line
// containing the given offset.
func TestLineStartOfMidLine(t *testing.T) {
	src := []byte("hello\nworld\n")
	// Offset 8 sits inside "world" — line start is 6.
	assert.Equal(t, 6, lineStartOf(src, 8))
}

// TestFirstTextStartReturnsNegativeForEmptySubtree covers the
// sentinel return path when no Text node can be found under n.
func TestFirstTextStartReturnsNegativeForEmptySubtree(t *testing.T) {
	// An empty file has no children.
	f := mkFile(t, "\n")
	root := f.AST
	// A *real* ast.Document has no Text descendants, so
	// firstTextStart returns -1 for it.
	assert.Equal(t, -1, firstTextStart(root))
}

// TestFindHeadingIDIgnoresHeadingWithoutAttribute confirms that a
// heading parsed without an `id` attribute short-circuits
// findHeadingID and produces no finding.
func TestFindHeadingIDIgnoresHeadingWithoutAttribute(t *testing.T) {
	// "# Heading" alone: no attribute block, no finding.
	fs := findings(t, "# Heading\n")
	assert.False(t, hasFeature(fs, FeatureHeadingIDs))
}

// TestFindHeadingIDIgnoresAttributesWithoutID covers the second
// guard: the heading has an attribute block but no `id` key.
func TestFindHeadingIDIgnoresAttributesWithoutID(t *testing.T) {
	// Goldmark's attribute parser accepts class-only attribute
	// blocks like `{.highlight}`. Those set Attributes() != nil but
	// no "id" key, so findHeadingID should return ok=false.
	fs := findings(t, "# Heading {.highlight}\n")
	assert.False(t, hasFeature(fs, FeatureHeadingIDs))
}

// TestTaskCheckBoxFindingOrphan exercises the defensive fallback in
// taskCheckBoxFinding when the node has no block ancestor — which
// only happens if the AST was hand-constructed rather than produced
// by goldmark. The fallback returns (1, 1).
func TestTaskCheckBoxFindingOrphan(t *testing.T) {
	f, err := lint.NewFile("t.md", []byte("body\n"))
	require.NoError(t, err)
	orphan := extast.NewTaskCheckBox(true)
	got := taskCheckBoxFinding(f, orphan)
	assert.Equal(t, FeatureTaskLists, got.Feature)
	assert.Equal(t, 1, got.Line)
	assert.Equal(t, 1, got.Column)
}

// TestInlineExtFindingOrphan is the same test for inlineExtFinding.
func TestInlineExtFindingOrphan(t *testing.T) {
	f, err := lint.NewFile("t.md", []byte("body\n"))
	require.NoError(t, err)
	orphan := extast.NewFootnoteLink(7)
	got := inlineExtFinding(f, orphan, FeatureFootnotes)
	assert.Equal(t, FeatureFootnotes, got.Feature)
	assert.Equal(t, 1, got.Line)
	assert.Equal(t, 1, got.Column)
}

// TestFindingFromBlockNoLines covers the `lines == nil || .Len()==0`
// short-circuit: a freshly-constructed block with no Lines appended
// falls back to (1, 1).
func TestFindingFromBlockNoLines(t *testing.T) {
	f, err := lint.NewFile("t.md", []byte("body\n"))
	require.NoError(t, err)
	block := ast.NewParagraph() // no Lines appended
	got := findingFromBlock(f, block, FeatureTables)
	assert.Equal(t, FeatureTables, got.Feature)
	assert.Equal(t, 1, got.Line)
	assert.Equal(t, 1, got.Column)
}

// TestNodeByteRangeClampsNegativeStart covers the clamp in
// nodeByteRange that floors a negative firstTextStart result to 0.
// A FootnoteLink has no children and no source segment, so
// firstTextStart returns -1 and nodeByteRange must floor that.
func TestNodeByteRangeClampsNegativeStart(t *testing.T) {
	n := extast.NewFootnoteLink(7)
	start, end := nodeByteRange(n)
	assert.Equal(t, 0, start)
	assert.Equal(t, 0, end)
}
