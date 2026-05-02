package horizontalrulestyle

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRule() *Rule {
	return &Rule{Style: "dash", Length: 3, RequireBlankLines: true}
}

func newFile(t *testing.T, name string, src []byte) *lint.File {
	t.Helper()
	f, err := lint.NewFile(name, src)
	require.NoError(t, err)
	return f
}

func TestCheckValid(t *testing.T) {
	src := []byte("# Title\n\nText.\n\n---\n\nMore.\n")
	diags := newRule().Check(newFile(t, "f.md", src))
	assert.Empty(t, diags)
}

func TestCheckWrongDelimiter(t *testing.T) {
	src := []byte("# Title\n\nText.\n\n***\n\nMore.\n")
	diags := newRule().Check(newFile(t, "f.md", src))
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "horizontal rule uses asterisk")
	assert.Contains(t, diags[0].Message, "configured style is dash")
}

func TestCheckInternalSpaces(t *testing.T) {
	src := []byte("# Title\n\nText.\n\n- - -\n\nMore.\n")
	diags := newRule().Check(newFile(t, "f.md", src))
	require.Len(t, diags, 1)
	assert.Equal(t, "horizontal rule has internal spaces", diags[0].Message)
}

func TestCheckWrongLength(t *testing.T) {
	src := []byte("# Title\n\nText.\n\n-----\n\nMore.\n")
	diags := newRule().Check(newFile(t, "f.md", src))
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "horizontal rule has length 5; configured length is 3")
}

func TestCheckMissingBlankAbove(t *testing.T) {
	src := []byte("# Title\n---\n\nText.\n")
	diags := newRule().Check(newFile(t, "f.md", src))
	require.Len(t, diags, 1)
	assert.Equal(t, "horizontal rule needs a blank line above", diags[0].Message)
}

func TestCheckMissingBlankBelow(t *testing.T) {
	src := []byte("# Title\n\n---\nText.\n")
	diags := newRule().Check(newFile(t, "f.md", src))
	require.Len(t, diags, 1)
	assert.Equal(t, "horizontal rule needs a blank line below", diags[0].Message)
}

func TestCheckSetextHeadingNotFlagged(t *testing.T) {
	src := []byte("Title\n=====\n\nText.\n")
	diags := newRule().Check(newFile(t, "f.md", src))
	assert.Empty(t, diags)
}

func TestCheckDisabledByDefault(t *testing.T) {
	r := &Rule{Style: "dash", Length: 3, RequireBlankLines: true}
	assert.False(t, r.EnabledByDefault())
}

func TestCheckAsteriskStyle(t *testing.T) {
	r := &Rule{Style: "asterisk", Length: 3, RequireBlankLines: true}
	src := []byte("# Title\n\nText.\n\n***\n\nMore.\n")
	diags := r.Check(newFile(t, "f.md", src))
	assert.Empty(t, diags)
}

func TestCheckUnderscoreStyle(t *testing.T) {
	r := &Rule{Style: "underscore", Length: 3, RequireBlankLines: true}
	src := []byte("# Title\n\nText.\n\n___\n\nMore.\n")
	diags := r.Check(newFile(t, "f.md", src))
	assert.Empty(t, diags)
}

func TestCheckCustomLength(t *testing.T) {
	r := &Rule{Style: "dash", Length: 5, RequireBlankLines: true}
	src := []byte("# Title\n\nText.\n\n-----\n\nMore.\n")
	diags := r.Check(newFile(t, "f.md", src))
	assert.Empty(t, diags)
}

func TestFixWrongDelimiter(t *testing.T) {
	src := []byte("# Title\n\nText.\n\n***\n\nMore.\n")
	f := newFile(t, "f.md", src)
	out := newRule().Fix(f)
	assert.Contains(t, string(out), "\n---\n")
	assert.NotContains(t, string(out), "***")
}

func TestFixInsertsBlankLines(t *testing.T) {
	src := []byte("# Title\n---\nText.\n")
	f := newFile(t, "f.md", src)
	out := string(newRule().Fix(f))
	assert.Contains(t, out, "\n\n---\n\n")
}

func TestFixNoDuplicateBlanks(t *testing.T) {
	// Two adjacent thematic breaks — fix must not produce double blank lines.
	src := []byte("# Title\n\n---\n---\n\nText.\n")
	f := newFile(t, "f.md", src)
	out := string(newRule().Fix(f))
	assert.NotContains(t, out, "\n\n\n")
}

func TestParseHR(t *testing.T) {
	tests := []struct {
		token     string
		wantDelim rune
		wantCount int
		wantSpace bool
	}{
		{"---", '-', 3, false},
		{"***", '*', 3, false},
		{"___", '_', 3, false},
		{"- - -", '-', 3, true},
		{"-----", '-', 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			d, c, s := parseHR(tt.token)
			assert.Equal(t, tt.wantDelim, d)
			assert.Equal(t, tt.wantCount, c)
			assert.Equal(t, tt.wantSpace, s)
		})
	}
}

func TestApplySettingsValid(t *testing.T) {
	r := newRule()
	require.NoError(t, r.ApplySettings(map[string]any{
		"style":               "asterisk",
		"length":              5,
		"require-blank-lines": false,
	}))
	assert.Equal(t, "asterisk", r.Style)
	assert.Equal(t, 5, r.Length)
	assert.False(t, r.RequireBlankLines)
}

func TestApplySettingsLengthInt64(t *testing.T) {
	r := newRule()
	require.NoError(t, r.ApplySettings(map[string]any{"length": int64(4)}))
	assert.Equal(t, 4, r.Length)
}

func TestApplySettingsInvalidStyle(t *testing.T) {
	r := newRule()
	assert.Error(t, r.ApplySettings(map[string]any{"style": "invalid"}))
}

func TestApplySettingsLengthTooSmall(t *testing.T) {
	r := newRule()
	assert.Error(t, r.ApplySettings(map[string]any{"length": 2}))
}

func TestCategory(t *testing.T) {
	assert.Equal(t, "whitespace", newRule().Category())
}

func TestDefaultSettings(t *testing.T) {
	ds := newRule().DefaultSettings()
	assert.Equal(t, "dash", ds["style"])
	assert.Equal(t, 3, ds["length"])
	assert.Equal(t, true, ds["require-blank-lines"])
}

func TestCheckNoBlankLinesRequired(t *testing.T) {
	r := &Rule{Style: "dash", Length: 3, RequireBlankLines: false}
	src := []byte("# Title\n---\nText.\n")
	diags := r.Check(newFile(t, "f.md", src))
	assert.Empty(t, diags)
}

func TestFixNoChangesNeeded(t *testing.T) {
	src := []byte("# Title\n\n---\n\nText.\n")
	f := newFile(t, "f.md", src)
	out := newRule().Fix(f)
	// Should return the original source unchanged.
	assert.Equal(t, string(f.Source), string(out))
}

func TestSplitHRLineNoDelimiter(t *testing.T) {
	prefix, token := splitHRLine("just some text")
	assert.Equal(t, "just some text", prefix)
	assert.Equal(t, "", token)
}

func TestSplitHRLineWithPrefix(t *testing.T) {
	prefix, token := splitHRLine("> ---")
	assert.Equal(t, "> ", prefix)
	assert.Equal(t, "---", token)
}

func TestStyleNameUnderscore(t *testing.T) {
	assert.Equal(t, "underscore", styleName('_'))
}

func TestStyleNameDash(t *testing.T) {
	assert.Equal(t, "dash", styleName('-'))
}

func TestIsBlankLineOutOfBounds(t *testing.T) {
	lines := [][]byte{[]byte("hello\n")}
	assert.True(t, isBlankLine(lines, -1))
	assert.True(t, isBlankLine(lines, 5))
}

func TestApplySettingsNonStringStyle(t *testing.T) {
	r := newRule()
	assert.Error(t, r.ApplySettings(map[string]any{"style": 42}))
}

func TestApplySettingsNonBoolRequireBlankLines(t *testing.T) {
	r := newRule()
	assert.Error(t, r.ApplySettings(map[string]any{"require-blank-lines": "yes"}))
}

func TestApplySettingsLengthFloat64(t *testing.T) {
	r := newRule()
	require.NoError(t, r.ApplySettings(map[string]any{"length": float64(4)}))
	assert.Equal(t, 4, r.Length)
}

func TestApplySettingsLengthNotInt(t *testing.T) {
	r := newRule()
	assert.Error(t, r.ApplySettings(map[string]any{"length": "three"}))
}

