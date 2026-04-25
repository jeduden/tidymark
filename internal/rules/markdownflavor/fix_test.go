package markdownflavor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixWith parses src into a *lint.File, applies the configured rule's
// Fix, and returns the result as a string for compact assertion.
func fixWith(t *testing.T, flavor, src string) string {
	t.Helper()
	r := &Rule{}
	require.NoError(t, r.ApplySettings(map[string]any{"flavor": flavor}))
	return string(r.Fix(mkFile(t, src)))
}

// --- heading IDs --------------------------------------------------------

func TestRuleFixHeadingIDRemovesAttributeBlock(t *testing.T) {
	got := fixWith(t, "commonmark", "# Heading {#top}\n\nBody text.\n")
	assert.Equal(t, "# Heading\n\nBody text.\n", got)
}

func TestRuleFixHeadingIDPreservesTrailingNewline(t *testing.T) {
	// A heading at end-of-file with no trailing newline must stay that way.
	got := fixWith(t, "commonmark", "# Heading {#top}")
	assert.Equal(t, "# Heading", got)
}

func TestRuleFixHeadingIDMultiple(t *testing.T) {
	src := "# A {#a}\n\n## B {#b}\n"
	got := fixWith(t, "commonmark", src)
	assert.Equal(t, "# A\n\n## B\n", got)
}

func TestRuleFixHeadingIDGoldmarkAccepts(t *testing.T) {
	// goldmark supports heading IDs, so Fix must not strip them.
	src := "# Heading {#top}\n"
	got := fixWith(t, "goldmark", src)
	assert.Equal(t, src, got)
}

// --- strikethrough ------------------------------------------------------

func TestRuleFixStrikethroughRemovesMarkers(t *testing.T) {
	got := fixWith(t, "commonmark", "Text ~~crossed out~~ here.\n")
	assert.Equal(t, "Text crossed out here.\n", got)
}

func TestRuleFixStrikethroughGFMAccepts(t *testing.T) {
	src := "Text ~~crossed out~~ here.\n"
	got := fixWith(t, "gfm", src)
	assert.Equal(t, src, got)
}

// --- task lists ---------------------------------------------------------

func TestRuleFixTaskListRemovesMarker(t *testing.T) {
	src := "- [x] done\n- [ ] todo\n"
	got := fixWith(t, "commonmark", src)
	assert.Equal(t, "- done\n- todo\n", got)
}

func TestRuleFixTaskListPreservesBullet(t *testing.T) {
	// The plan calls out `*` and `+` bullets explicitly.
	src := "* [X] one\n+ [ ] two\n"
	got := fixWith(t, "commonmark", src)
	assert.Equal(t, "* one\n+ two\n", got)
}

func TestRuleFixTaskListGFMAccepts(t *testing.T) {
	src := "- [x] done\n"
	got := fixWith(t, "gfm", src)
	assert.Equal(t, src, got)
}

// --- superscript --------------------------------------------------------

func TestRuleFixSuperscriptRemovesCarets(t *testing.T) {
	got := fixWith(t, "commonmark", "E = mc^2^ is famous.\n")
	assert.Equal(t, "E = mc2 is famous.\n", got)
}

// --- subscript ----------------------------------------------------------

func TestRuleFixSubscriptRemovesTildes(t *testing.T) {
	got := fixWith(t, "commonmark", "H~2~O is water.\n")
	assert.Equal(t, "H2O is water.\n", got)
}

// --- bare URLs ----------------------------------------------------------

func TestRuleFixBareURLWrapsInAngleBrackets(t *testing.T) {
	got := fixWith(t, "commonmark",
		"Visit https://example.com for details.\n")
	assert.Equal(t,
		"Visit <https://example.com> for details.\n", got)
}

func TestRuleFixBareURLGFMAccepts(t *testing.T) {
	src := "Visit https://example.com for details.\n"
	got := fixWith(t, "gfm", src)
	assert.Equal(t, src, got)
}

// --- combined -----------------------------------------------------------

// TestRuleFixMultipleFeaturesOnOneLine covers reverse-order edit
// application: two byte-range edits on the same line must compose
// without one corrupting the other's offsets.
func TestRuleFixMultipleFeaturesOnOneLine(t *testing.T) {
	got := fixWith(t, "commonmark",
		"# Heading {#top}\n\nText ~~old~~ at https://example.com.\n")
	assert.Equal(t,
		"# Heading\n\nText old at <https://example.com>.\n", got)
}
