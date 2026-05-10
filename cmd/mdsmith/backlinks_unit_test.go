package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitTarget(t *testing.T) {
	cases := []struct {
		input      string
		wantPath   string
		wantAnchor string
	}{
		{"docs/api.md", "docs/api.md", ""},
		{"docs/api.md#section", "docs/api.md", "section"},
		{"docs/api.md#with-dashes", "docs/api.md", "with-dashes"},
		{"#anchor-only", "", "anchor-only"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			p, a := splitTarget(tc.input)
			assert.Equal(t, tc.wantPath, p)
			assert.Equal(t, tc.wantAnchor, a)
		})
	}
}

func TestResolveLinkTarget(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		linkPath string
		want     string
	}{
		{"sibling", "docs/index.md", "api.md", "docs/api.md"},
		{"dot-prefix", "docs/index.md", "./api.md", "docs/api.md"},
		{"parent dir", "docs/sub/index.md", "../api.md", "docs/api.md"},
		{"two levels up", "plan/045.md", "../docs/api.md", "docs/api.md"},
		{"escapes root", "docs/api.md", "../../etc/passwd", ""},
		{"absolute link", "docs/api.md", "/etc/passwd", ""},
		{"absolute source", "/abs/docs/api.md", "guide.md", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveLinkTarget(tc.src, tc.linkPath)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNormalizeWorkspacePath(t *testing.T) {
	root := t.TempDir()
	assert.Equal(t, "docs/api.md", normalizeWorkspacePath("docs/api.md", root))
	assert.Equal(t, "docs/api.md", normalizeWorkspacePath("./docs/api.md", root))
	// Absolute path inside root maps to relative.
	abs := filepath.Join(root, "docs", "api.md")
	assert.Equal(t, "docs/api.md", normalizeWorkspacePath(abs, root))
}

func TestSourceMatches(t *testing.T) {
	assert.True(t, sourceMatches("docs/api.md", nil))
	assert.True(t, sourceMatches("docs/api.md", []string{"docs/**"}))
	assert.False(t, sourceMatches("plan/045.md", []string{"docs/**"}))
	assert.True(t, sourceMatches("plan/045.md", []string{"docs/**", "plan/**"}))
}

func TestEmitBacklinks_Text(t *testing.T) {
	var buf bytes.Buffer
	records := []backlinkRecord{
		{Source: "a.md", Line: 1, Text: "ref", Target: "x.md"},
		{Source: "b.md", Line: 2, Text: "ref2", Target: "./x.md"},
	}
	code := emitBacklinks(&buf, records, "text", 0)
	assert.Equal(t, 0, code)
	out := buf.String()
	assert.Contains(t, out, "a.md:1: [ref](x.md)\n")
	assert.Contains(t, out, "b.md:2: [ref2](./x.md)\n")
}

func TestEmitBacklinks_JSON(t *testing.T) {
	var buf bytes.Buffer
	records := []backlinkRecord{
		{Source: "a.md", Line: 1, Text: "ref", Target: "x.md"},
	}
	code := emitBacklinks(&buf, records, "json", 0)
	assert.Equal(t, 0, code)
	var decoded []backlinkRecord
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Equal(t, records, decoded)
}

func TestEmitBacklinks_JSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	code := emitBacklinks(&buf, nil, "json", 0)
	assert.Equal(t, 1, code, "no records → exit 1")
	// `[]` is the documented stable shape; never `null`.
	assert.Contains(t, buf.String(), "[]")
	assert.NotContains(t, buf.String(), "null")
}

func TestEmitBacklinks_Limit(t *testing.T) {
	records := []backlinkRecord{
		{Source: "a.md", Line: 1},
		{Source: "b.md", Line: 1},
		{Source: "c.md", Line: 1},
	}
	var buf bytes.Buffer
	code := emitBacklinks(&buf, records, "text", 2)
	assert.Equal(t, 0, code)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)
}

func TestEmitBacklinks_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	code := emitBacklinks(&buf, []backlinkRecord{{Source: "a.md"}}, "yaml", 0)
	assert.Equal(t, 2, code)
}

// TestCollectBacklinks_End2End covers the path/anchor combinations the
// plan's acceptance criteria call out: three sources, anchor scoping,
// include filter, limit.
func TestCollectBacklinks_End2End(t *testing.T) {
	root := t.TempDir()
	mkdir := func(rel string) {
		require.NoError(t, os.MkdirAll(filepath.Join(root, rel), 0o755))
	}
	write := func(rel, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644))
	}
	mkdir("docs")
	mkdir("plan")
	mkdir("docs/sub")
	write("docs/api.md", "# API\n\n## Authentication\n\n## Endpoints\n")
	write("docs/index.md", "# Index\n\nSee [API reference](api.md).\n")
	write("docs/sub/guide.md", "# Guide\n\nUse [api docs](../api.md#authentication).\n")
	write("plan/045_api-overhaul.md", "# Plan\n\n[api](../docs/api.md)\n")
	// File that does NOT link to api.md.
	write("docs/changelog.md", "# Changelog\n\n[plan](../plan/045_api-overhaul.md)\n")

	files := []string{
		filepath.Join(root, "docs/api.md"),
		filepath.Join(root, "docs/changelog.md"),
		filepath.Join(root, "docs/index.md"),
		filepath.Join(root, "docs/sub/guide.md"),
		filepath.Join(root, "plan/045_api-overhaul.md"),
	}

	t.Run("three sources, no anchor", func(t *testing.T) {
		got := collectBacklinks(files, root, "docs/api.md", "", nil, 0)
		require.Len(t, got, 3)
		assert.Equal(t, "docs/index.md", got[0].Source)
		assert.Equal(t, "docs/sub/guide.md", got[1].Source)
		assert.Equal(t, "plan/045_api-overhaul.md", got[2].Source)
	})

	t.Run("anchor scopes to one source", func(t *testing.T) {
		got := collectBacklinks(files, root, "docs/api.md", "authentication", nil, 0)
		require.Len(t, got, 1)
		assert.Equal(t, "docs/sub/guide.md", got[0].Source)
	})

	t.Run("anchor with no hits returns empty", func(t *testing.T) {
		got := collectBacklinks(files, root, "docs/api.md", "no-such-section", nil, 0)
		assert.Empty(t, got)
	})

	t.Run("include filter excludes plan/", func(t *testing.T) {
		got := collectBacklinks(files, root, "docs/api.md", "", []string{"docs/**"}, 0)
		require.Len(t, got, 2)
		assert.Equal(t, "docs/index.md", got[0].Source)
		assert.Equal(t, "docs/sub/guide.md", got[1].Source)
	})
}
