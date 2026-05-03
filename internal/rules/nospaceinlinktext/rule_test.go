package nospaceinlinktext

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func check(t *testing.T, src string, checkImages bool) []lint.Diagnostic {
	t.Helper()
	f, err := lint.NewFile("test.md", []byte(src))
	require.NoError(t, err)
	r := &Rule{CheckImages: checkImages}
	return r.Check(f)
}

func fix(t *testing.T, src string, checkImages bool) string {
	t.Helper()
	f, err := lint.NewFile("test.md", []byte(src))
	require.NoError(t, err)
	r := &Rule{CheckImages: checkImages}
	return string(r.Fix(f))
}

func TestCleanLink(t *testing.T) {
	diags := check(t, "# T\n\n[text](url)\n", true)
	assert.Empty(t, diags)
}

func TestLeadingSpace(t *testing.T) {
	diags := check(t, "# T\n\n[ text](url)\n", true)
	require.Len(t, diags, 1)
	assert.Equal(t, "link text has leading whitespace", diags[0].Message)
}

func TestTrailingSpace(t *testing.T) {
	diags := check(t, "# T\n\n[text ](url)\n", true)
	require.Len(t, diags, 1)
	assert.Equal(t, "link text has trailing whitespace", diags[0].Message)
}

func TestLeadingAndTrailingSpace(t *testing.T) {
	diags := check(t, "# T\n\n[ text ](url)\n", true)
	require.Len(t, diags, 2)
	msgs := []string{diags[0].Message, diags[1].Message}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestFixLeadingAndTrailing(t *testing.T) {
	result := fix(t, "# T\n\n[ text ](url)\n", true)
	assert.Equal(t, "# T\n\n[text](url)\n", result)
}

func TestImageLeadingSpace(t *testing.T) {
	diags := check(t, "# T\n\n![ alt](img.png)\n", true)
	require.Len(t, diags, 1)
	assert.Equal(t, "image alt text has leading whitespace", diags[0].Message)
}

func TestImageTrailingSpace(t *testing.T) {
	diags := check(t, "# T\n\n![alt ](img.png)\n", true)
	require.Len(t, diags, 1)
	assert.Equal(t, "image alt text has trailing whitespace", diags[0].Message)
}

func TestImageBothSpaces(t *testing.T) {
	diags := check(t, "# T\n\n![ alt ](img.png)\n", true)
	require.Len(t, diags, 2)
	msgs := []string{diags[0].Message, diags[1].Message}
	assert.Contains(t, msgs, "image alt text has leading whitespace")
	assert.Contains(t, msgs, "image alt text has trailing whitespace")
}

func TestImageFixBothSpaces(t *testing.T) {
	result := fix(t, "# T\n\n![ alt ](img.png)\n", true)
	assert.Equal(t, "# T\n\n![alt](img.png)\n", result)
}

func TestCheckImagesDisabled(t *testing.T) {
	diags := check(t, "# T\n\n![ alt ](img.png)\n", false)
	assert.Empty(t, diags)
}

func TestNewlineInLinkTextNotFlagged(t *testing.T) {
	// Newline between words inside brackets must not be flagged.
	diags := check(t, "# T\n\n[long text that\nwraps](url)\n", true)
	assert.Empty(t, diags)
}

func TestNoChange(t *testing.T) {
	src := "# T\n\n[text](url)\n"
	result := fix(t, src, true)
	assert.Equal(t, src, result)
}

func TestApplySettings_CheckImages(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"check-images": false})
	require.NoError(t, err)
	assert.False(t, r.CheckImages)
}

func TestApplySettings_Unknown(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"unknown": true})
	require.Error(t, err)
}

func TestDefaultSettings(t *testing.T) {
	r := &Rule{}
	s := r.DefaultSettings()
	assert.Equal(t, true, s["check-images"])
}

func TestEnabledByDefault(t *testing.T) {
	r := &Rule{}
	assert.False(t, r.EnabledByDefault())
}

func TestCategory(t *testing.T) {
	r := &Rule{}
	assert.Equal(t, "link", r.Category())
}

func TestApplySettings_CheckImagesWrongType(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"check-images": "yes"})
	require.Error(t, err)
}

func TestEmptyLinkText(t *testing.T) {
	// [](url) has no text child; bracketSpan returns (-1,-1) and is skipped.
	diags := check(t, "# T\n\n[](url)\n", true)
	assert.Empty(t, diags)
}

func TestEmptyLinkTextFix(t *testing.T) {
	src := "# T\n\n[](url)\n"
	result := fix(t, src, true)
	assert.Equal(t, src, result)
}

func TestLinkWithImageChild(t *testing.T) {
	// [![img](img.png)](url) — outer link wraps an image with clean alt text.
	// Outer link inner text is "![img](img.png)" — first byte '!', not space.
	// The inner image has no whitespace in alt text.
	diags := check(t, "# T\n\n[![img](img.png)](url)\n", true)
	assert.Empty(t, diags)
}

func TestLinkWithImageChildSpacedAlt(t *testing.T) {
	// [![ alt ](img.png)](url) — inner image has leading/trailing space in alt.
	// Outer link inner text is "![ alt ](img.png)" — first byte '!', not space.
	// The inner image produces two diagnostics.
	diags := check(t, "# T\n\n[![ alt ](img.png)](url)\n", true)
	require.Len(t, diags, 2)
	msgs := []string{diags[0].Message, diags[1].Message}
	assert.Contains(t, msgs, "image alt text has leading whitespace")
	assert.Contains(t, msgs, "image alt text has trailing whitespace")
}

func TestLinkWithImageChildFixSpacedAlt(t *testing.T) {
	result := fix(t, "# T\n\n[![ alt ](img.png)](url)\n", true)
	assert.Equal(t, "# T\n\n[![alt](img.png)](url)\n", result)
}

func TestBracketInLinkText(t *testing.T) {
	// [foo [bar] baz](url) — nested brackets exercise depth tracking.
	diags := check(t, "# T\n\n[foo [bar] baz](url)\n", true)
	assert.Empty(t, diags)
}

func TestEmphasisFirstChildTrailingSpace(t *testing.T) {
	// [*x* ](url) — emphasis is the first child, trailing space follows.
	diags := check(t, "# T\n\n[*x* ](url)\n", true)
	require.Len(t, diags, 1)
	assert.Equal(t, "link text has trailing whitespace", diags[0].Message)
}

func TestEmphasisFirstChildFix(t *testing.T) {
	result := fix(t, "# T\n\n[*x* ](url)\n", true)
	assert.Equal(t, "# T\n\n[*x*](url)\n", result)
}

func TestImageEmphasisFirstChild(t *testing.T) {
	// ![ *alt* ](img.png) — emphasis inside image alt text.
	diags := check(t, "# T\n\n![ *alt* ](img.png)\n", true)
	require.Len(t, diags, 2)
	msgs := []string{diags[0].Message, diags[1].Message}
	assert.Contains(t, msgs, "image alt text has leading whitespace")
	assert.Contains(t, msgs, "image alt text has trailing whitespace")
}

func TestEscapedBracketInLinkText(t *testing.T) {
	// [ text \] ](url) — escaped ] inside link text should not terminate scan.
	// Goldmark parses \] as an escaped bracket; the real closing ] is later.
	diags := check(t, "# T\n\n[ text \\] ](url)\n", true)
	// Leading space and trailing space around the escaped bracket content.
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestLinkWithTextAndNestedImage(t *testing.T) {
	// [ text ![ alt ](img.png) ](url) — outer link has boundary spaces,
	// inner image also has spaces in alt text.
	// Check reports all four diagnostics.
	diags := check(t, "# T\n\n[ text ![ alt ](img.png) ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
	assert.Contains(t, msgs, "image alt text has leading whitespace")
	assert.Contains(t, msgs, "image alt text has trailing whitespace")
}

func TestFixNestedSpanNoOverlapPanic(t *testing.T) {
	// Fix must not panic when outer link and inner image both need trimming.
	// The outer fix is applied; the inner requires a second pass.
	src := "# T\n\n[ text ![ alt ](img.png) ](url)\n"
	result := fix(t, src, true)
	// Outer boundary spaces must be removed.
	assert.Contains(t, string(result), "[text !")
	// Source must not be unchanged (at least outer fix applied).
	assert.NotEqual(t, src, result)
}

func TestLinkWithTextAndNestedImageClean(t *testing.T) {
	// [ text !(img.png) ](url) — outer link has boundary spaces but clean inner image.
	// Only link text diagnostics; inner image has no alt whitespace.
	diags := check(t, "# T\n\n[ text ![clean](img.png) ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestLinkWithImageThenTrailingSpace(t *testing.T) {
	// [ ![img](img.png) tail ](url) — link has image child followed by text;
	// both boundary spaces must be detected.
	diags := check(t, "# T\n\n[ ![img](img.png) tail ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestLinkStartingWithImageTrailingSpace(t *testing.T) {
	// [![img](img.png) tail ](url) — no space between [ and ![, so minStart is
	// inside the image subtree. The backward scan hits the image's [ first and
	// must skip it (imageOpener=true, img=false) to find the outer [.
	diags := check(t, "# T\n\n[![img](img.png) tail ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestLinkWithOnlyImageAndBoundarySpaces(t *testing.T) {
	// [ ![alt](img.png) ](url) — link content is only an image with surrounding spaces.
	// The outer link's inner text starts with space; leading and trailing are detected.
	diags := check(t, "# T\n\n[ ![alt](img.png) ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestFixLinkWithOnlyImageAndBoundarySpaces(t *testing.T) {
	result := fix(t, "# T\n\n[ ![alt](img.png) ](url)\n", true)
	assert.Equal(t, "# T\n\n[![alt](img.png)](url)\n", result)
}

func TestEscapedExclamationBeforeLink(t *testing.T) {
	// \![ text ](url) — escaped '!' before '['; goldmark parses this as a link,
	// not an image. imageOpener checks the backslash count and returns false,
	// so the [ is correctly identified as the link opener.
	diags := check(t, "# T\n\n\\![ text ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestCodeSpanWithCloseBracketNotFlaggedAsClose(t *testing.T) {
	// [ text `]` ](url) — `]` inside code span (forward scan) must not terminate
	// the forward scan early. The trailing space before the real `]` is detected.
	diags := check(t, "# T\n\n[ text `]` ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestCodeSpanWithOpenBracketNotFlaggedAsNested(t *testing.T) {
	// [ `[` text ](url) — `[` inside code span (forward scan) must not increment
	// bracket depth. The leading space is detected.
	diags := check(t, "# T\n\n[ `[` text](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
}

func TestCodeSpanWithOpenBracketBackwardScan(t *testing.T) {
	// [`[code]` text ](url) — no space between [ and the code span, so the backward
	// scan starts after the code span and must skip it via skipCodeSpanBackward.
	// The trailing space is detected.
	diags := check(t, "# T\n\n[`[code]` text ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestDoubleBacktickCodeSpanBackwardScan(t *testing.T) {
	// [``x`y`` text ](url) — double-backtick code span with a single backtick
	// inside. skipCodeSpanBackward must skip the wrong-length single-backtick
	// sequence before finding the double-backtick opener. Trailing space detected.
	diags := check(t, "# T\n\n[``x`y`` text ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestDoubleBacktickCodeSpanForwardScan(t *testing.T) {
	// [ text ``x`y`` ](url) — double-backtick code span (forward scan) with
	// single backtick inside; skipCodeSpan must skip the wrong-length sequence.
	diags := check(t, "# T\n\n[ text ``x`y`` ](url)\n", true)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	assert.Contains(t, msgs, "link text has leading whitespace")
	assert.Contains(t, msgs, "link text has trailing whitespace")
}

func TestSkipCodeSpanBackwardUnit(t *testing.T) {
	// Unit test for skipCodeSpanBackward directly.
	// Source: abc``x`y``def where double-backtick code span ends at position 10.
	src := []byte("abc``x`y``def")
	// Closing `` is at positions 8-9. Call with i=9 (last char of closer).
	result := skipCodeSpanBackward(src, 9)
	// Should return position before the opening ``, which starts at position 3.
	// j=3 at end, return j-1=2.
	assert.Equal(t, 2, result)
}

func TestSkipCodeSpanBackwardWrongLength(t *testing.T) {
	// skipCodeSpanBackward with n=2, encountering single-backtick first.
	// Source: abc`d``ef``gh where double-backtick closer ends at 9.
	// Single backtick at 3 has wrong length; opener is at 5-6.
	src := []byte("abc`d``ef``gh")
	// Closer `` is at positions 9-10. Call with i=10 (last ` of closer).
	result := skipCodeSpanBackward(src, 10)
	// Opener `` is at 5-6. return j-1 = 4.
	assert.Equal(t, 4, result)
}
