package orderedlistnumbering

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleMetadata(t *testing.T) {
	r := &Rule{}
	assert.Equal(t, "MDS046", r.ID())
	assert.Equal(t, "ordered-list-numbering", r.Name())
	assert.Equal(t, "list", r.Category())
	assert.False(t, r.EnabledByDefault(), "rule must be opt-in")
}

func TestCheck_Sequential_GoodList(t *testing.T) {
	src := []byte("1. a\n2. b\n3. c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestCheck_Sequential_AllOnesIsBad(t *testing.T) {
	src := []byte("1. a\n1. b\n1. c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	diags := r.Check(f)
	require.Len(t, diags, 2)
	assert.Equal(t, 2, diags[0].Line)
	assert.Equal(t, 3, diags[1].Line)
	assert.Contains(t, diags[0].Message, "item 2")
	assert.Contains(t, diags[0].Message, "expected 2")
}

func TestCheck_AllOnes_GoodList(t *testing.T) {
	src := []byte("1. a\n1. b\n1. c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleAllOnes, Start: 1}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestCheck_AllOnes_FlagsNonOnes(t *testing.T) {
	src := []byte("1. a\n3. b\n7. c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleAllOnes, Start: 1}
	diags := r.Check(f)
	require.Len(t, diags, 2)
	assert.Equal(t, 2, diags[0].Line)
	assert.Equal(t, 3, diags[1].Line)
	assert.Contains(t, diags[0].Message, "expected 1")
}

func TestCheck_WrongStart(t *testing.T) {
	src := []byte("5. a\n6. b\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	diags := r.Check(f)
	require.Len(t, diags, 1, "wrong start should fire one diag and suppress per-item dups")
	assert.Equal(t, 1, diags[0].Line)
	assert.Contains(t, diags[0].Message, "starts at 5")
	assert.Contains(t, diags[0].Message, "configured start is 1")
}

func TestCheck_WrongStart_SuppressesPerItemDiagsForRest(t *testing.T) {
	// Without suppression the items would also fire ("expected 2",
	// "expected 3") under configured start=1, contradicting the
	// auto-fix output that simply renumbers from 1.
	src := []byte("5. a\n5. b\n5. c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	diags := r.Check(f)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "starts at 5")
}

func TestCheck_UnorderedListNotFlagged(t *testing.T) {
	src := []byte("- a\n- b\n- c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestCheck_NestedOrderedList(t *testing.T) {
	src := []byte("1. parent\n   1. child\n   1. child two\n2. parent two\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	diags := r.Check(f)
	require.Len(t, diags, 1, "outer list ok; inner list flags one item")
	assert.Equal(t, 3, diags[0].Line)
}

func TestFix_SequentialFromAllOnes(t *testing.T) {
	src := []byte("1. a\n1. b\n1. c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	got := r.Fix(f)
	want := "1. a\n2. b\n3. c\n"
	assert.Equal(t, want, string(got))
}

func TestFix_AllOnesFromMixed(t *testing.T) {
	src := []byte("1. a\n3. b\n7. c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleAllOnes, Start: 1}
	got := r.Fix(f)
	want := "1. a\n1. b\n1. c\n"
	assert.Equal(t, want, string(got))
}

func TestFix_DigitWidthGrowth_AdjustsContinuation(t *testing.T) {
	// 12 items, all "1." with all-ones, fixed to sequential.
	// Items 10-12 will grow from "1." to "10." ... "12." (+1 char).
	// Continuation lines under those items must shift right by 1.
	src := []byte("" +
		"1. one\n" +
		"1. two\n" +
		"1. three\n" +
		"1. four\n" +
		"1. five\n" +
		"1. six\n" +
		"1. seven\n" +
		"1. eight\n" +
		"1. nine\n" +
		"1. ten\n" +
		"   continuation of ten\n" +
		"1. eleven\n" +
		"1. twelve\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	got := r.Fix(f)
	want := "" +
		"1. one\n" +
		"2. two\n" +
		"3. three\n" +
		"4. four\n" +
		"5. five\n" +
		"6. six\n" +
		"7. seven\n" +
		"8. eight\n" +
		"9. nine\n" +
		"10. ten\n" +
		"    continuation of ten\n" +
		"11. eleven\n" +
		"12. twelve\n"
	assert.Equal(t, want, string(got))
}

func TestFix_DigitWidthShrink_AdjustsContinuation(t *testing.T) {
	// 12-item sequential list fixed to all-ones; items 10-12 shrink.
	src := []byte("" +
		"1. one\n" +
		"2. two\n" +
		"3. three\n" +
		"4. four\n" +
		"5. five\n" +
		"6. six\n" +
		"7. seven\n" +
		"8. eight\n" +
		"9. nine\n" +
		"10. ten\n" +
		"    continuation of ten\n" +
		"11. eleven\n" +
		"12. twelve\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleAllOnes, Start: 1}
	got := r.Fix(f)
	want := "" +
		"1. one\n" +
		"1. two\n" +
		"1. three\n" +
		"1. four\n" +
		"1. five\n" +
		"1. six\n" +
		"1. seven\n" +
		"1. eight\n" +
		"1. nine\n" +
		"1. ten\n" +
		"   continuation of ten\n" +
		"1. eleven\n" +
		"1. twelve\n"
	assert.Equal(t, want, string(got))
}

func TestFix_NoChangeWhenAlreadyCorrect(t *testing.T) {
	src := []byte("1. a\n2. b\n3. c\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	got := r.Fix(f)
	assert.Equal(t, string(src), string(got))
}

func TestApplySettings_Style(t *testing.T) {
	r := &Rule{Style: StyleSequential, Start: 1}
	require.NoError(t, r.ApplySettings(map[string]any{"style": "all-ones"}))
	assert.Equal(t, StyleAllOnes, r.Style)
}

func TestApplySettings_Start(t *testing.T) {
	r := &Rule{Style: StyleSequential, Start: 1}
	require.NoError(t, r.ApplySettings(map[string]any{"start": 3}))
	assert.Equal(t, 3, r.Start)
}

func TestApplySettings_InvalidStyle(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"style": "wibble"})
	assert.Error(t, err)
}

func TestApplySettings_InvalidStartType(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"start": "one"})
	assert.Error(t, err)
}

func TestApplySettings_NegativeStart(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"start": -1})
	assert.Error(t, err)
}

func TestApplySettings_UnknownKey(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"oops": true})
	assert.Error(t, err)
}

func TestDefaultSettings(t *testing.T) {
	r := &Rule{}
	ds := r.DefaultSettings()
	assert.Equal(t, StyleSequential, ds["style"])
	assert.Equal(t, 1, ds["start"])
}

func TestRegistered(t *testing.T) {
	require.NotNil(t, rule.ByID("MDS046"), "rule must register itself")
}

func TestParseListItemNumber_NoDigits(t *testing.T) {
	_, _, _, _, ok := parseListItemNumber([]byte("hello"))
	assert.False(t, ok)
}

func TestParseListItemNumber_DigitsOnlyNoMarker(t *testing.T) {
	_, _, _, _, ok := parseListItemNumber([]byte("12"))
	assert.False(t, ok, "digits with no marker char must not parse")
}

func TestParseListItemNumber_DigitsWrongFollower(t *testing.T) {
	_, _, _, _, ok := parseListItemNumber([]byte("12x"))
	assert.False(t, ok, "non-marker char after digits must not parse")
}

func TestParseListItemNumber_ParenMarker(t *testing.T) {
	n, _, _, marker, ok := parseListItemNumber([]byte("3) item"))
	require.True(t, ok)
	assert.Equal(t, 3, n)
	assert.Equal(t, byte(')'), marker)
}

func TestApplyIndentShift_Zero(t *testing.T) {
	in := []byte("hello")
	out := applyIndentShift(in, 0)
	assert.Equal(t, "hello", string(out))
}

func TestApplyIndentShift_NegativeExceedsLeading(t *testing.T) {
	// Asking for -3 when only 1 leading space is present must
	// return the line unchanged so we never eat content.
	in := []byte(" hello")
	out := applyIndentShift(in, -3)
	assert.Equal(t, " hello", string(out))
}

func TestReplaceLeadingDigits_NoDigits(t *testing.T) {
	in := []byte("hello")
	out := replaceLeadingDigits(in, 7)
	assert.Equal(t, "hello", string(out))
}

func TestDigitWidth(t *testing.T) {
	assert.Equal(t, 1, digitWidth(0))
	assert.Equal(t, 1, digitWidth(9))
	assert.Equal(t, 2, digitWidth(10))
	assert.Equal(t, 3, digitWidth(123))
}

func TestApplySettings_FloatStart(t *testing.T) {
	r := &Rule{}
	require.NoError(t, r.ApplySettings(map[string]any{"start": 5.0}))
	assert.Equal(t, 5, r.Start)
}

func TestCheck_NestedListItemContinuationLines(t *testing.T) {
	// An item whose content spans multiple block children
	// (paragraph + nested list) exercises lastLineOfListItem's
	// recursion through both children.
	src := []byte("" +
		"1. parent\n" +
		"\n" +
		"   continuation\n" +
		"\n" +
		"   - nested\n" +
		"2. next\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	diags := r.Check(f)
	assert.Empty(t, diags)
}

func TestFix_WidthGrowth_RecursesIntoNestedList(t *testing.T) {
	// Item 10 in the outer list contains a nested unordered
	// list. When the outer marker grows from "1." to "10.",
	// the indent shift must extend through the nested list
	// content lines — exercises blockLastLine's recursion
	// through container blocks.
	src := []byte("" +
		"1. one\n" +
		"1. two\n" +
		"1. three\n" +
		"1. four\n" +
		"1. five\n" +
		"1. six\n" +
		"1. seven\n" +
		"1. eight\n" +
		"1. nine\n" +
		"1. ten\n" +
		"\n" +
		"   - nested under ten\n" +
		"   - nested under ten too\n" +
		"\n" +
		"1. eleven\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)
	r := &Rule{Style: StyleSequential, Start: 1}
	got := r.Fix(f)
	want := "" +
		"1. one\n" +
		"2. two\n" +
		"3. three\n" +
		"4. four\n" +
		"5. five\n" +
		"6. six\n" +
		"7. seven\n" +
		"8. eight\n" +
		"9. nine\n" +
		"10. ten\n" +
		"\n" +
		"    - nested under ten\n" +
		"    - nested under ten too\n" +
		"\n" +
		"11. eleven\n"
	assert.Equal(t, want, string(got))
}
