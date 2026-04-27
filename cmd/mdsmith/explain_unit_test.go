package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/output"
)

// --- attachExplanations ---

func TestAttachExplanations_DefaultLayer(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	target := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(target, []byte("# Title\n"), 0o644))

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"line-length": {Enabled: true, Settings: map[string]any{"max": 80}},
		},
	}
	diags := []lint.Diagnostic{
		{File: "doc.md", RuleID: "MDS001", RuleName: "line-length", Message: "too long"},
	}
	attachExplanations(cfg, diags)

	require.NotNil(t, diags[0].Explanation)
	assert.Equal(t, "line-length", diags[0].Explanation.Rule)
	assert.Equal(t, "default", diags[0].Explanation.Source)
}

func TestAttachExplanations_KindLayer(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	mdDir := filepath.Join(dir, "wide")
	require.NoError(t, os.MkdirAll(mdDir, 0o755))
	target := filepath.Join(mdDir, "doc.md")
	require.NoError(t, os.WriteFile(target, []byte("# Title\n"), 0o644))

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"line-length": {Enabled: true, Settings: map[string]any{"max": 80}},
		},
		Kinds: map[string]config.KindBody{
			"wide": {Rules: map[string]config.RuleCfg{
				"line-length": {Enabled: true, Settings: map[string]any{"max": 200}},
			}},
		},
		KindAssignment: []config.KindAssignmentEntry{
			{Files: []string{"wide/*.md"}, Kinds: []string{"wide"}},
		},
	}
	diags := []lint.Diagnostic{
		{File: "wide/doc.md", RuleID: "MDS001", RuleName: "line-length", Message: "too long"},
	}
	attachExplanations(cfg, diags)
	require.NotNil(t, diags[0].Explanation)
	assert.Equal(t, "kinds.wide", diags[0].Explanation.Source)
	assert.Contains(t, diags[0].Explanation.Kinds, "wide")
}

func TestAttachExplanations_NilCfgIsSafe(t *testing.T) {
	diags := []lint.Diagnostic{
		{File: "doc.md", RuleName: "line-length"},
	}
	// Passing a non-nil but empty cfg with a missing rule is the
	// realistic edge — explanations should simply not be added.
	cfg := &config.Config{}
	attachExplanations(cfg, diags)
	// Resolution is created from defaults; rule not present -> no explanation.
	assert.Nil(t, diags[0].Explanation)
}

// --- output formatters honor Explanation ---

func TestTextFormatter_RendersExplanationTrailer(t *testing.T) {
	diags := []lint.Diagnostic{
		{
			File:     "doc.md",
			Line:     1,
			Column:   1,
			RuleID:   "MDS001",
			RuleName: "line-length",
			Message:  "too long",
			Explanation: &lint.Explanation{
				Rule:   "line-length",
				Source: "kinds.wide",
			},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, (&output.TextFormatter{}).Format(&buf, diags))
	out := buf.String()
	assert.Contains(t, out, "line-length=kinds.wide")
}

func TestJSONFormatter_IncludesExplanation(t *testing.T) {
	diags := []lint.Diagnostic{
		{
			File:     "doc.md",
			Line:     1,
			Column:   1,
			RuleID:   "MDS001",
			RuleName: "line-length",
			Message:  "too long",
			Explanation: &lint.Explanation{
				Rule:        "line-length",
				Source:      "kinds.wide",
				Kinds:       []string{"wide"},
				LeafSources: map[string]string{"max": "kinds.wide"},
			},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, (&output.JSONFormatter{}).Format(&buf, diags))

	var got []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got, 1)
	exp, ok := got[0]["explanation"].(map[string]any)
	require.True(t, ok, "expected explanation object in JSON output")
	assert.Equal(t, "line-length", exp["rule"])
	assert.Equal(t, "kinds.wide", exp["source"])
}

func TestJSONFormatter_OmitsExplanationWhenNil(t *testing.T) {
	diags := []lint.Diagnostic{
		{File: "doc.md", RuleID: "MDS001", RuleName: "line-length", Message: "x"},
	}
	var buf bytes.Buffer
	require.NoError(t, (&output.JSONFormatter{}).Format(&buf, diags))

	var got []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	_, present := got[0]["explanation"]
	assert.False(t, present, "explanation must be omitted when nil")
}
