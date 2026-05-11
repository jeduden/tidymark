package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeDiagForTest is the diagnostic constructor the tests use; it
// matches MDS020's shape so message formats are exercised the same
// way the real rule emits them.
func makeDiagForTest(file string, line int, msg string) lint.Diagnostic {
	return lint.Diagnostic{
		File:    file,
		Line:    line,
		Column:  1,
		RuleID:  "MDS020",
		Message: msg,
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func newDocFile(t *testing.T, path, source string) *lint.File {
	t.Helper()
	f, err := lint.NewFile(path, []byte(source))
	require.NoError(t, err)
	return f
}

// ---- ParseFile (compat with legacy heading-template) ----

func TestParseFile_FlatHeadings(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md", "# ?\n\n## Goal\n\n## Tasks\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	assert.Equal(t, 1, sch.RootLevel)
	require.Len(t, sch.Sections, 1)
	require.Equal(t, "?", sch.Sections[0].Heading)
	require.True(t, sch.Sections[0].Closed)
	require.Len(t, sch.Sections[0].Sections, 2)
	assert.Equal(t, "Goal", sch.Sections[0].Sections[0].Heading)
	assert.Equal(t, "Tasks", sch.Sections[0].Sections[1].Heading)
}

func TestParseFile_NoH1(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md", "## ...\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	assert.Equal(t, 2, sch.RootLevel)
	require.Len(t, sch.Sections, 1)
	assert.True(t, sch.Sections[0].Wildcard)
}

func TestParseFile_FrontmatterCUE(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md", "---\nid: 'string'\n---\n# ?\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	assert.Equal(t, "string", sch.Frontmatter["id"])
}

func TestParseFile_RequireFilename(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md",
		"<?require\nfilename: \"foo-*.md\"\n?>\n# ?\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	assert.Equal(t, "foo-*.md", sch.Require.Filename)
}

// ---- ParseInline ----

func TestParseInline_Empty(t *testing.T) {
	sch, err := ParseInline(map[string]any{}, "kind rfc")
	require.NoError(t, err)
	require.NotNil(t, sch)
	assert.Equal(t, 2, sch.RootLevel)
	assert.True(t, sch.IsEmpty())
}

func TestParseInline_FrontmatterAndSections(t *testing.T) {
	raw := map[string]any{
		"frontmatter": map[string]any{
			"id":     `=~"^RFC-[0-9]{4}$"`,
			"title?": `string`,
		},
		"require": map[string]any{
			"filename": "RFC-[0-9][0-9][0-9][0-9].md",
		},
		"closed": true,
		"sections": []any{
			map[string]any{
				"heading":  "Overview",
				"required": true,
			},
			map[string]any{
				"heading":  "Decision",
				"required": true,
				"sections": []any{map[string]any{"heading": "Outcome"}},
				"aliases":  []any{"Resolution"},
			},
			"...",
		},
	}
	sch, err := ParseInline(raw, "kind rfc")
	require.NoError(t, err)
	assert.True(t, sch.Closed)
	assert.Equal(t, "RFC-[0-9][0-9][0-9][0-9].md", sch.Require.Filename)
	require.Len(t, sch.Sections, 3)
	assert.Equal(t, "Overview", sch.Sections[0].Heading)
	assert.Equal(t, []string{"Resolution"}, sch.Sections[1].Aliases)
	require.Len(t, sch.Sections[1].Sections, 1)
	assert.Equal(t, "Outcome", sch.Sections[1].Sections[0].Heading)
	assert.True(t, sch.Sections[2].Wildcard)

	assert.Contains(t, sch.FrontmatterCUE(), `id: =~"^RFC-[0-9]{4}$"`)
	assert.Contains(t, sch.FrontmatterCUE(), `title?: string`)
}

func TestParseInline_UnknownTopKey(t *testing.T) {
	_, err := ParseInline(map[string]any{"foo": 1}, "kind x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown schema key")
}

func TestParseInline_BadScopeKey(t *testing.T) {
	raw := map[string]any{
		"sections": []any{map[string]any{"unknown": true}},
	}
	_, err := ParseInline(raw, "kind x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown scope key")
}

// ---- Validate (legacy fixtures behaviour) ----

func TestValidate_MissingSection(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md", "# ?\n\n## Goal\n\n## Tasks\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md", "# My Plan\n\n## Goal\n\nGoal text.\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	require.Len(t, diags, 1)
	assert.Equal(t, `missing required section "## Tasks"`, diags[0].Message)
}

func TestValidate_ExtraSectionFlagged(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md", "# ?\n\n## Goal\n\n## Tasks\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md",
		"# My Plan\n\n## Extra\n\nx\n\n## Goal\n\ny\n\n## Tasks\n\nz\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `unexpected section "## Extra"`)
}

func TestValidate_OutOfOrder(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md", "# ?\n\n## Goal\n\n## Tasks\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md",
		"# My Plan\n\n## Tasks\n\nx\n\n## Goal\n\ny\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	require.Len(t, diags, 1)
	assert.Equal(t,
		`section "## Tasks" out of order: expected after "## Goal"`,
		diags[0].Message)
}

func TestValidate_WildcardH1LevelMismatch(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md", "# ?\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md", "## Title\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	require.Len(t, diags, 1)
	assert.Equal(t,
		`heading level mismatch for "Title": expected h1, got h2`,
		diags[0].Message)
}

func TestValidate_FilenameMismatch(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "proto.md",
		"<?require\nfilename: \"[0-9]*_*.md\"\n?>\n# ?\n")
	sch, err := ParseFile(&FileReader{}, p)
	require.NoError(t, err)
	doc := newDocFile(t, "filename-mismatch.md", "# My Doc\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message,
		`filename "filename-mismatch.md" does not match required pattern "[0-9]*_*.md"`)
}

// ---- Validate (inline schemas) ----

func TestValidate_Inline_OpenScopeTolerates(t *testing.T) {
	raw := map[string]any{
		"sections": []any{
			map[string]any{"heading": "Overview"},
			map[string]any{"heading": "Decision"},
		},
	}
	sch, err := ParseInline(raw, "kind rfc")
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md",
		"# Title\n\n## Overview\n\nx\n\n## Notes\n\ny\n\n## Decision\n\nz\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	assert.Empty(t, diags,
		"open scope should tolerate the unlisted Notes heading")
}

func TestValidate_Inline_ClosedRejectsUnlisted(t *testing.T) {
	raw := map[string]any{
		"closed": true,
		"sections": []any{
			map[string]any{"heading": "Overview"},
			map[string]any{"heading": "Decision"},
		},
	}
	sch, err := ParseInline(raw, "kind rfc")
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md",
		"# Title\n\n## Overview\n\nx\n\n## Notes\n\ny\n\n## Decision\n\nz\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, `unexpected section "## Notes"`)
}

func TestValidate_Inline_WildcardSlotTolerates(t *testing.T) {
	raw := map[string]any{
		"closed": true,
		"sections": []any{
			map[string]any{"heading": "Overview"},
			"...",
			map[string]any{"heading": "References"},
		},
	}
	sch, err := ParseInline(raw, "kind rfc")
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md",
		"# Title\n\n## Overview\n\nx\n\n## A\n\ny\n\n## B\n\nz\n\n## References\n\nw\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	assert.Empty(t, diags,
		"wildcard slot should tolerate unlisted sections at that position")
}

func TestValidate_Inline_AliasMatches(t *testing.T) {
	raw := map[string]any{
		"sections": []any{
			map[string]any{
				"heading": "Symptoms",
				"aliases": []any{"Indicators"},
			},
		},
	}
	sch, err := ParseInline(raw, "kind runbook")
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md", "# T\n\n## Indicators\n\nx\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	assert.Empty(t, diags)
}

func TestValidate_Inline_NestedThreeLevels(t *testing.T) {
	raw := map[string]any{
		"sections": []any{
			map[string]any{
				"heading":  "Diagnosis",
				"required": true,
				"sections": []any{
					map[string]any{
						"heading":  "Step",
						"required": true,
						"sections": []any{
							map[string]any{"heading": "Check"},
							map[string]any{"heading": "Expected"},
						},
					},
				},
			},
		},
	}
	sch, err := ParseInline(raw, "kind runbook")
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md",
		"# T\n\n## Diagnosis\n\n### Step\n\n#### Check\n\nx\n\n#### Expected\n\ny\n")
	diags := Validate(doc, sch, nil, false, makeDiagForTest)
	assert.Empty(t, diags, "three-level tree should validate cleanly")
}

func TestValidate_Inline_FrontmatterCUE(t *testing.T) {
	raw := map[string]any{
		"frontmatter": map[string]any{
			"id": `=~"^RFC-[0-9]{4}$"`,
		},
	}
	sch, err := ParseInline(raw, "kind rfc")
	require.NoError(t, err)
	doc := newDocFile(t, "doc.md", "# T\n")
	// Document FM has the wrong shape.
	diags := Validate(doc, sch, map[string]any{"id": "BAD"}, false, makeDiagForTest)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "front matter does not satisfy schema")
}
