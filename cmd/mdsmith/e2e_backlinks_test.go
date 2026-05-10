package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// backlinkRecordE2E mirrors cmd/mdsmith.backlinkRecord with exported
// JSON keys. Kept local to avoid coupling the test package to the
// internal struct definition.
type backlinkRecordE2E struct {
	Source string `json:"source"`
	Line   int    `json:"line"`
	Text   string `json:"text"`
	Target string `json:"target"`
}

// setupBacklinksWorkspace creates a minimal workspace with three
// distinct sources linking to docs/api.md, plus an anchor case.
func setupBacklinksWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mk := func(rel string) {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, rel), 0o755))
	}
	wf := func(rel, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644))
	}
	mk("docs/sub")
	mk("plan")
	// Mark the temp dir as a project root so config discovery does
	// not climb out into the repo's own .mdsmith.yml.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	wf(".mdsmith.yml", "files:\n  - \"**/*.md\"\nrules:\n  cross-file-reference-integrity: false\n")
	wf("docs/api.md", "# API\n\n## Authentication\n\n## Endpoints\n")
	wf("docs/index.md", "# Index\n\nSee [API reference](api.md).\n")
	wf("docs/sub/guide.md", "# Guide\n\nUse [api docs](../api.md#authentication).\n")
	wf("plan/045_api-overhaul.md", "# Plan\n\n[api](../docs/api.md)\n")
	wf("docs/changelog.md", "# Changelog\n\n[plan](../plan/045_api-overhaul.md)\n")
	return dir
}

func TestE2E_Backlinks_ThreeSources_Text(t *testing.T) {
	dir := setupBacklinksWorkspace(t)
	stdout, _, exitCode := runBinaryInDir(t, dir, "", "backlinks", "docs/api.md")
	require.Equal(t, 0, exitCode, "expected exit 0 when matches found")
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	require.Len(t, lines, 3, "expected 3 backlink rows, got: %q", stdout)
	// Sorted by source path.
	assert.True(t, strings.HasPrefix(lines[0], "docs/index.md:"))
	assert.True(t, strings.HasPrefix(lines[1], "docs/sub/guide.md:"))
	assert.True(t, strings.HasPrefix(lines[2], "plan/045_api-overhaul.md:"))
	assert.Contains(t, lines[0], "[API reference](api.md)")
}

func TestE2E_Backlinks_Anchor_Filter(t *testing.T) {
	dir := setupBacklinksWorkspace(t)
	stdout, _, exitCode := runBinaryInDir(t, dir, "", "backlinks", "docs/api.md#authentication")
	require.Equal(t, 0, exitCode)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	require.Len(t, lines, 1, "anchor-scoped query expected to return one row, got: %q", stdout)
	assert.Contains(t, lines[0], "docs/sub/guide.md")
}

func TestE2E_Backlinks_Anchor_NoHits_ExitsOne(t *testing.T) {
	dir := setupBacklinksWorkspace(t)
	stdout, _, exitCode := runBinaryInDir(t, dir, "", "backlinks", "docs/api.md#no-such-section")
	require.Equal(t, 1, exitCode, "no matches → exit 1")
	assert.Empty(t, strings.TrimSpace(stdout))
}

func TestE2E_Backlinks_JSON_Shape(t *testing.T) {
	dir := setupBacklinksWorkspace(t)
	stdout, _, exitCode := runBinaryInDir(t, dir, "", "backlinks", "--format", "json", "docs/api.md")
	require.Equal(t, 0, exitCode)
	var got []backlinkRecordE2E
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Len(t, got, 3)
	// Stable shape per docs/reference/cli/backlinks.md.
	assert.Equal(t, "docs/index.md", got[0].Source)
	assert.Equal(t, "API reference", got[0].Text)
	assert.Equal(t, "api.md", got[0].Target)
	assert.Greater(t, got[0].Line, 0)
}

func TestE2E_Backlinks_IncludeAndLimit(t *testing.T) {
	dir := setupBacklinksWorkspace(t)
	// --include docs/** filters out the plan/ source; --limit 1
	// then caps the remaining two rows.
	stdout, _, exitCode := runBinaryInDir(
		t, dir, "", "backlinks", "--include", "docs/**", "--limit", "1", "docs/api.md")
	require.Equal(t, 0, exitCode)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	require.Len(t, lines, 1, "include + limit: expected 1 row, got: %q", stdout)
	assert.Contains(t, lines[0], "docs/index.md")
	assert.NotContains(t, stdout, "plan/")
}

func TestE2E_Backlinks_NoTarget_ExitsTwo(t *testing.T) {
	_, stderr, exitCode := runBinary(t, "", "backlinks")
	assert.Equal(t, 2, exitCode)
	assert.Contains(t, stderr, "requires a target file argument")
}

func TestE2E_Backlinks_HelpFlag(t *testing.T) {
	_, stderr, exitCode := runBinary(t, "", "backlinks", "--help")
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stderr, "Usage: mdsmith backlinks")
}
