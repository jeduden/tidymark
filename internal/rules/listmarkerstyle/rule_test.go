package listmarkerstyle

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Check tests ---

func TestCheck_DashStyle_DashItem_NoViolation(t *testing.T) {
	src := []byte("# Title\n\n- item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestCheck_DashStyle_AsteriskItem_OneViolation(t *testing.T) {
	src := []byte("# Title\n\n* item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Equal(t, "MDS045", diags[0].RuleID)
	assert.Equal(t, "list-marker-style", diags[0].RuleName)
	assert.Equal(t, 3, diags[0].Line)
	assert.Equal(t, 1, diags[0].Column)
	assert.Equal(t, "unordered list uses *; configured style is -", diags[0].Message)
}

func TestCheck_DashStyle_PlusItem_OneViolation(t *testing.T) {
	src := []byte("# Title\n\n+ item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Equal(t, "unordered list uses +; configured style is -", diags[0].Message)
}

func TestCheck_AsteriskStyle_AsteriskItem_NoViolation(t *testing.T) {
	src := []byte("# Title\n\n* item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "asterisk"}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestCheck_AsteriskStyle_DashItem_OneViolation(t *testing.T) {
	src := []byte("# Title\n\n- item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "asterisk"}
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Equal(t, "unordered list uses -; configured style is *", diags[0].Message)
}

func TestCheck_PlusStyle_PlusItem_NoViolation(t *testing.T) {
	src := []byte("# Title\n\n+ item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "plus"}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestCheck_OrderedList_NoViolation(t *testing.T) {
	src := []byte("# Title\n\n1. item one\n2. item two\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestCheck_DisabledByDefault(t *testing.T) {
	r := &Rule{}
	assert.False(t, r.EnabledByDefault())
}

func TestCheck_NestedList_WithNestedSetting_CorrectMarkers_NoViolation(t *testing.T) {
	// Top-level with dash, nested with asterisk — matches nested: [dash, asterisk]
	src := []byte("# Title\n\n- item\n  * nested\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash", Nested: []string{"dash", "asterisk"}}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestCheck_NestedList_WithNestedSetting_WrongMarker_OneViolation(t *testing.T) {
	// Top-level with dash, nested also with dash — nested[1] should be asterisk
	src := []byte("# Title\n\n- item\n  - nested\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash", Nested: []string{"dash", "asterisk"}}
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "depth 1")
	assert.Contains(t, diags[0].Message, "uses -")
	assert.Contains(t, diags[0].Message, "expected *")
}

func TestCheck_NestedList_WithoutNestedSetting_SameMarkerRequired(t *testing.T) {
	// When no nested setting, nested list should use same style as top
	src := []byte("# Title\n\n- item\n  * nested\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Equal(t, "unordered list uses *; configured style is -", diags[0].Message)
}

// --- Fix tests ---

func TestFix_AsteriskToDash(t *testing.T) {
	src := []byte("# Title\n\n* item\n* item two\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	result := r.Fix(f)
	assert.Equal(t, "# Title\n\n- item\n- item two\n", string(result))
}

func TestFix_PlusToDash(t *testing.T) {
	src := []byte("+ item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	result := r.Fix(f)
	assert.Equal(t, "- item\n", string(result))
}

func TestFix_DashToAsterisk(t *testing.T) {
	src := []byte("- item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "asterisk"}
	result := r.Fix(f)
	assert.Equal(t, "* item\n", string(result))
}

func TestFix_NoChangeNeeded(t *testing.T) {
	src := []byte("- item\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	result := r.Fix(f)
	assert.Equal(t, string(src), string(result))
}

func TestFix_OrderedListNotTouched(t *testing.T) {
	src := []byte("1. item one\n2. item two\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash"}
	result := r.Fix(f)
	assert.Equal(t, string(src), string(result))
}

func TestFix_NestedWithNestedSetting(t *testing.T) {
	// Top-level dash correct, nested dash should become asterisk
	src := []byte("- item\n  - nested\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: "dash", Nested: []string{"dash", "asterisk"}}
	result := r.Fix(f)
	assert.Equal(t, "- item\n  * nested\n", string(result))
}

// --- ID / Name / Category ---

func TestRule_ID(t *testing.T) {
	r := &Rule{}
	assert.Equal(t, "MDS045", r.ID())
}

func TestRule_Name(t *testing.T) {
	r := &Rule{}
	assert.Equal(t, "list-marker-style", r.Name())
}

func TestRule_Category(t *testing.T) {
	r := &Rule{}
	assert.Equal(t, "list", r.Category())
}

// --- ApplySettings tests ---

func TestApplySettings_Style_Dash(t *testing.T) {
	r := &Rule{}
	require.NoError(t, r.ApplySettings(map[string]any{"style": "dash"}))
	assert.Equal(t, "dash", r.Style)
}

func TestApplySettings_Style_Asterisk(t *testing.T) {
	r := &Rule{}
	require.NoError(t, r.ApplySettings(map[string]any{"style": "asterisk"}))
	assert.Equal(t, "asterisk", r.Style)
}

func TestApplySettings_Style_Plus(t *testing.T) {
	r := &Rule{}
	require.NoError(t, r.ApplySettings(map[string]any{"style": "plus"}))
	assert.Equal(t, "plus", r.Style)
}

func TestApplySettings_Style_Invalid(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"style": "invalid"})
	require.Error(t, err)
}

func TestApplySettings_Style_WrongType(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"style": 42})
	require.Error(t, err)
}

func TestApplySettings_Nested_ValidSlice(t *testing.T) {
	r := &Rule{}
	require.NoError(t, r.ApplySettings(map[string]any{"nested": []any{"dash", "asterisk"}}))
	assert.Equal(t, []string{"dash", "asterisk"}, r.Nested)
}

func TestApplySettings_Nested_EmptySlice(t *testing.T) {
	r := &Rule{}
	require.NoError(t, r.ApplySettings(map[string]any{"nested": []any{}}))
	assert.Empty(t, r.Nested)
}

func TestApplySettings_Nested_InvalidValue(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"nested": "not-a-list"})
	require.Error(t, err)
}

func TestApplySettings_Nested_InvalidMarker(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"nested": []any{"dash", "invalid"}})
	require.Error(t, err)
}

func TestApplySettings_UnknownKey(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"unknown": true})
	require.Error(t, err)
}

func TestDefaultSettings(t *testing.T) {
	r := &Rule{}
	ds := r.DefaultSettings()
	assert.Equal(t, "", ds["style"])
	nested, ok := ds["nested"].([]string)
	require.True(t, ok)
	assert.Empty(t, nested)
}
