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
	// bracketSpan skips Image subtrees so the outer link returns (-1,-1).
	// The inner image is still visited and has no whitespace in alt text.
	diags := check(t, "# T\n\n[![img](img.png)](url)\n", true)
	assert.Empty(t, diags)
}

func TestLinkWithImageChildSpacedAlt(t *testing.T) {
	// [![ alt ](img.png)](url) — inner image has leading/trailing space in alt.
	// The outer link is skipped; the inner image produces two diagnostics.
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
