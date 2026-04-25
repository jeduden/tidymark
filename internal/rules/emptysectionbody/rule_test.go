package emptysectionbody

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheck_EmptySectionAtEOF(t *testing.T) {
	src := []byte("# Doc\n\n## Empty\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 1, "expected 1 diagnostic, got %d", len(diags))

	d := diags[0]
	if d.RuleID != "MDS030" {
		t.Errorf("expected MDS030, got %s", d.RuleID)
	}
	if d.Line != 3 {
		t.Errorf("expected line 3, got %d", d.Line)
	}
	assert.Contains(t, d.Message, "## Empty", "expected heading text in diagnostic, got: %s", d.Message)
}

func TestCheck_CommentOnlySection(t *testing.T) {
	src := []byte(
		"# Doc\n\n## Placeholder\n\n<!-- TODO: fill in later -->\n\n## Next\n\nBody.\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 1, "expected 1 diagnostic, got %d", len(diags))
	if diags[0].Line != 3 {
		t.Errorf("expected line 3, got %d", diags[0].Line)
	}
}

func TestCheck_AllowMarkerSkipsDiagnostic(t *testing.T) {
	src := []byte(
		"# Doc\n\n## Template Slot\n\n<?allow-empty-section?>\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 0, "expected 0 diagnostics, got %d", len(diags))
}

func TestCheck_PrefixedMarkerDoesNotSkipByDefault(t *testing.T) {
	src := []byte(
		"# Doc\n\n## Template Slot\n\n<!-- mdsmith: allow-empty-section -->\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 1, "expected 1 diagnostic, got %d", len(diags))
}

func TestCheck_MarkerCaseSensitive(t *testing.T) {
	src := []byte(
		"# Doc\n\n## Template Slot\n\n<!-- ALLOW-EMPTY-SECTION -->\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 1, "expected 1 diagnostic, got %d", len(diags))
}

func TestCheck_CustomAllowMarkerUsesExactString(t *testing.T) {
	src := []byte(
		"# Doc\n\n## Template Slot\n\n<?allow-empty-section?>\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{
		MinLevel:    defaultMinLevel,
		MaxLevel:    defaultMaxLevel,
		AllowMarker: "docs: intentionally-empty",
	}
	diags := r.Check(f)
	require.Len(t, diags, 1, "expected 1 diagnostic, got %d", len(diags))
}

func TestCheck_ListContentIsMeaningful(t *testing.T) {
	src := []byte("# Doc\n\n## Steps\n\n- first\n- second\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 0, "expected 0 diagnostics, got %d", len(diags))
}

func TestCheck_CodeContentIsMeaningful(t *testing.T) {
	src := []byte("# Doc\n\n## Example\n\n```go\nfmt.Println(\"hello\")\n```\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 0, "expected 0 diagnostics, got %d", len(diags))
}

func TestCheck_TableContentIsMeaningful(t *testing.T) {
	src := []byte(
		"# Doc\n\n## Matrix\n\n| A | B |\n|---|---|\n| 1 | 2 |\n",
	)
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 0, "expected 0 diagnostics, got %d", len(diags))
}

func TestCheck_NestedHeadingWithContentIsNotEmpty(t *testing.T) {
	src := []byte("# Doc\n\n## Parent\n\n### Child\n\nDetails.\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 0, "expected 0 diagnostics, got %d", len(diags))
}

func TestCheck_NestedHeadingWithoutContentReportsBothSections(t *testing.T) {
	src := []byte("# Doc\n\n## Parent\n\n### Child\n\n## Next\n\nBody.\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	require.Len(t, diags, 2, "expected 2 diagnostics, got %d", len(diags))
	if diags[0].Line != 3 {
		t.Errorf("expected first diagnostic at line 3, got %d", diags[0].Line)
	}
	if diags[1].Line != 5 {
		t.Errorf("expected second diagnostic at line 5, got %d", diags[1].Line)
	}
}

func TestCheck_MinLevelSkipsH2WhenSetTo3(t *testing.T) {
	src := []byte("# Doc\n\n## Parent\n\n### Child\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{MinLevel: 3, MaxLevel: 6, AllowMarker: defaultAllowMarker}
	diags := r.Check(f)
	require.Len(t, diags, 1, "expected 1 diagnostic, got %d", len(diags))
	if diags[0].Line != 5 {
		t.Errorf("expected line 5, got %d", diags[0].Line)
	}
}

func TestApplySettings_Valid(t *testing.T) {
	r := &Rule{MinLevel: defaultMinLevel, MaxLevel: defaultMaxLevel, AllowMarker: defaultAllowMarker}
	err := r.ApplySettings(map[string]any{
		"min-level":    3,
		"max-level":    5,
		"allow-marker": "intentionally-empty",
	})
	require.NoError(t, err, "unexpected error: %v", err)
	if r.MinLevel != 3 {
		t.Errorf("expected MinLevel=3, got %d", r.MinLevel)
	}
	if r.MaxLevel != 5 {
		t.Errorf("expected MaxLevel=5, got %d", r.MaxLevel)
	}
	if r.AllowMarker != "intentionally-empty" {
		t.Errorf("unexpected allow marker: %s", r.AllowMarker)
	}
}

func TestApplySettings_InvalidType(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"min-level": "two"})
	require.Error(t, err, "expected error for invalid type")
}

func TestApplySettings_AllowMarkerWhitespaceOnly(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"allow-marker": "   "})
	require.Error(t, err, "expected error for whitespace-only allow-marker")
}

func TestApplySettings_AllowMarkerContainsWhitespace(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"allow-marker": "docs: empty"})
	require.Error(t, err, "expected error for allow-marker containing whitespace")
}

func TestApplySettings_InvalidRange(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"max-level": 8})
	require.Error(t, err, "expected error for invalid max-level")
}

func TestApplySettings_MinGreaterThanMax(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"min-level": 5,
		"max-level": 3,
	})
	require.Error(t, err, "expected error when min-level > max-level")
}

func TestApplySettings_UnknownKey(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"unknown": true})
	require.Error(t, err, "expected error for unknown setting")
}

func TestDefaultSettings(t *testing.T) {
	r := &Rule{}
	ds := r.DefaultSettings()
	if ds["min-level"] != defaultMinLevel {
		t.Errorf("expected min-level=%d, got %v", defaultMinLevel, ds["min-level"])
	}
	if ds["max-level"] != defaultMaxLevel {
		t.Errorf("expected max-level=%d, got %v", defaultMaxLevel, ds["max-level"])
	}
	if ds["allow-marker"] != defaultAllowMarker {
		t.Errorf("expected allow-marker=%q, got %v", defaultAllowMarker, ds["allow-marker"])
	}
}

func TestID(t *testing.T) {
	r := &Rule{}
	if r.ID() != "MDS030" {
		t.Errorf("expected MDS030, got %s", r.ID())
	}
}

func TestName(t *testing.T) {
	r := &Rule{}
	if r.Name() != "empty-section-body" {
		t.Errorf("expected empty-section-body, got %s", r.Name())
	}
}

func TestCategory(t *testing.T) {
	r := &Rule{}
	if r.Category() != "heading" {
		t.Errorf("expected heading, got %s", r.Category())
	}
}

// --- toInt coverage ---

func TestToInt_Int64(t *testing.T) {
	n, ok := toInt(int64(42))
	assert.True(t, ok)
	assert.Equal(t, 42, n)
}

func TestToInt_Float64_NonWhole(t *testing.T) {
	_, ok := toInt(float64(2.5))
	assert.False(t, ok, "non-whole float64 should return false")
}

func TestToInt_Float64_Whole(t *testing.T) {
	n, ok := toInt(float64(3))
	assert.True(t, ok)
	assert.Equal(t, 3, n)
}

// --- headingLine setext coverage ---

func TestHeadingLine_SetextHeading(t *testing.T) {
	// Setext headings have Lines().Len() > 0; headingLine should use Lines().At(0).Start
	src := []byte("# Doc\n\n## Setext Section\n-------------------\n\nContent.\n")
	// Actually goldmark parses "## Setext" as ATX (it starts with #).
	// Use a true setext heading to trigger Lines().Len() > 0 branch.
	src = []byte("Setext Section\n==============\n\nContent.\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	// With min-level=1 to also check level-1 headings
	r := &Rule{MinLevel: 1, MaxLevel: 6, AllowMarker: defaultAllowMarker}
	diags := r.Check(f)
	// The section has content, so no diagnostic expected
	assert.Len(t, diags, 0)
}

func TestHeadingLine_SetextEmptySection(t *testing.T) {
	// Setext level-1 heading with empty body to trigger headingLine via Lines()
	src := []byte("Empty Section\n=============\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{MinLevel: 1, MaxLevel: 6, AllowMarker: defaultAllowMarker}
	diags := r.Check(f)
	require.Len(t, diags, 1)
	// headingLine for setext heading should report line 1
	assert.Equal(t, 1, diags[0].Line)
}

// --- nodeHasText via Check ---

func TestNodeHasText_ParagraphContent(t *testing.T) {
	// A paragraph node hits the default branch in hasMeaningfulContent -> nodeHasText
	src := []byte("# Doc\n\n## Section\n\nSome paragraph text.\n")
	f, err := lint.NewFile("test.md", src)
	require.NoError(t, err)

	r := &Rule{}
	diags := r.Check(f)
	assert.Len(t, diags, 0, "section with paragraph should not be empty")
}

func TestApplySettings_Int64MinLevel(t *testing.T) {
	// ApplySettings passes int64 values through toInt (via YAML decoding scenario)
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"min-level": int64(2)})
	require.NoError(t, err)
	assert.Equal(t, 2, r.MinLevel)
}
