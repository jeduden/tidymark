package sectionsizelimits

import (
	"regexp"
	"strings"
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustFile(t *testing.T, src string) *lint.File {
	t.Helper()
	f, err := lint.NewFile("test.md", []byte(src))
	require.NoError(t, err)
	return f
}

func TestCheck_NoHeadings_NoDiagnostic(t *testing.T) {
	f := mustFile(t, "just some text\nand more\n")
	r := &Rule{Max: 5}
	assert.Empty(t, r.Check(f))
}

func TestCheck_SectionUnderLimit_NoDiagnostic(t *testing.T) {
	src := "# Title\nline 1\nline 2\n"
	r := &Rule{Max: 5}
	assert.Empty(t, r.Check(mustFile(t, src)))
}

func TestCheck_SectionOverLimit_Diagnostic(t *testing.T) {
	src := "# Title\na\nb\nc\nd\ne\n"
	r := &Rule{Max: 3}
	diags := r.Check(mustFile(t, src))
	require.Len(t, diags, 1)
	d := diags[0]
	assert.Equal(t, "MDS036", d.RuleID)
	assert.Equal(t, "section-size-limits", d.RuleName)
	assert.Equal(t, lint.Warning, d.Severity)
	assert.Equal(t, 1, d.Line)
	assert.Equal(t, 1, d.Column)
	assert.Contains(t, d.Message, "# Title")
	assert.Contains(t, d.Message, "6 > 3")
}

func TestCheck_SectionEndsAtNextHeading(t *testing.T) {
	src := "# A\nx\ny\n# B\nz\n"
	r := &Rule{Max: 2}
	diags := r.Check(mustFile(t, src))
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "# A")
	assert.Contains(t, diags[0].Message, "3 > 2")
}

func TestCheck_SubsectionExcludedFromParent(t *testing.T) {
	src := "# H1\n## H2\na\nb\nc\nd\ne\n"
	r := &Rule{Max: 3}
	diags := r.Check(mustFile(t, src))
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "## H2")
	assert.Contains(t, diags[0].Message, "6 > 3")
}

func TestCheck_PerLevelOverridesDefault(t *testing.T) {
	src := "# H1\na\nb\nc\n## H2\nx\ny\nz\n"
	r := &Rule{Max: 10, PerLevel: map[int]int{2: 2}}
	diags := r.Check(mustFile(t, src))
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "## H2")
	assert.Contains(t, diags[0].Message, "4 > 2")
}

func TestCheck_PerHeadingOverridesLevel(t *testing.T) {
	src := "## Intro\na\nb\nc\nd\n## Other\nx\ny\nz\n"
	r := &Rule{
		PerLevel: map[int]int{2: 2},
		PerHeading: []HeadingPattern{
			{Pattern: "^Intro$", Regex: regexp.MustCompile("^Intro$"), Max: 10},
		},
	}
	diags := r.Check(mustFile(t, src))
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "## Other")
}

func TestCheck_ZeroMaxMeansNoLimit(t *testing.T) {
	src := "# Title\n" + strings.Repeat("x\n", 1000)
	r := &Rule{Max: 0}
	assert.Empty(t, r.Check(mustFile(t, src)))
}

func TestCheck_HeadingInCodeBlockIgnored(t *testing.T) {
	src := "# Real\nline1\nline2\n```\n# Not a heading\n```\nline3\n"
	r := &Rule{Max: 3}
	diags := r.Check(mustFile(t, src))
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "# Real")
}

func TestApplySettings_Max(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"max": 50})
	require.NoError(t, err)
	assert.Equal(t, 50, r.Max)
}

func TestApplySettings_PerLevel(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"per-level": map[string]any{"1": 100, "2": 50},
	})
	require.NoError(t, err)
	assert.Equal(t, 100, r.PerLevel[1])
	assert.Equal(t, 50, r.PerLevel[2])
}

func TestApplySettings_PerHeading(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"per-heading": []any{
			map[string]any{"pattern": "^Intro$", "max": 10},
		},
	})
	require.NoError(t, err)
	require.Len(t, r.PerHeading, 1)
	assert.Equal(t, 10, r.PerHeading[0].Max)
	assert.NotNil(t, r.PerHeading[0].Regex)
}

func TestApplySettings_InvalidMaxType(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"max": "abc"})
	assert.Error(t, err)
}

func TestApplySettings_InvalidPerLevelKey(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"per-level": map[string]any{"notanint": 5},
	})
	assert.Error(t, err)
}

func TestApplySettings_InvalidPerLevelRange(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"per-level": map[string]any{"7": 5},
	})
	assert.Error(t, err)
}

func TestApplySettings_InvalidPerHeadingRegex(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"per-heading": []any{
			map[string]any{"pattern": "[unterminated", "max": 10},
		},
	})
	assert.Error(t, err)
}

func TestApplySettings_UnknownKey(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"unknown": 1})
	assert.Error(t, err)
}

func TestDefaultSettings(t *testing.T) {
	r := &Rule{}
	ds := r.DefaultSettings()
	assert.Equal(t, 0, ds["max"])
}

func TestID(t *testing.T) {
	assert.Equal(t, "MDS036", (&Rule{}).ID())
}

func TestName(t *testing.T) {
	assert.Equal(t, "section-size-limits", (&Rule{}).Name())
}

func TestCategory(t *testing.T) {
	assert.Equal(t, "heading", (&Rule{}).Category())
}
