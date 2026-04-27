package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_CheckExplain_TextTrailer asserts that `mdsmith check --explain`
// emits a provenance trailer line under each diagnostic naming the
// rule and the source layer.
func TestE2E_CheckExplain_TextTrailer(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)
	// Disable rules other than line-length so the diagnostic order is
	// predictable and the trailer we assert on is stable.
	cfg := `categories:
  heading: false
  whitespace: false
  meta: false
  line: true
rules:
  line-length:
    max: 5
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".mdsmith.yml"), []byte(cfg), 0o644))
	target := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(target, []byte("# this line is way too long\n"), 0o644))

	_, stderr, exit := runBinaryInDir(t, dir, "", "check", "--explain", "--no-color", "doc.md")
	assert.Equal(t, 1, exit, "diagnostics should exit 1; stderr=%s", stderr)
	assert.Contains(t, stderr, "└─")
	assert.Contains(t, stderr, "line-length=default")
}

// TestE2E_CheckExplain_JSONExplanation asserts the JSON output carries
// an "explanation" object on each diagnostic.
func TestE2E_CheckExplain_JSONExplanation(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)
	cfg := `rules:
  line-length:
    max: 1000
kinds:
  wide:
    rules:
      line-length:
        max: 5
kind-assignment:
  - files: ["wide/*.md"]
    kinds: [wide]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".mdsmith.yml"), []byte(cfg), 0o644))
	wideDir := filepath.Join(dir, "wide")
	require.NoError(t, os.MkdirAll(wideDir, 0o755))
	target := filepath.Join(wideDir, "doc.md")
	require.NoError(t, os.WriteFile(target, []byte("# t\nlong line text\n"), 0o644))

	stdout, stderr, exit := runBinaryInDir(t, dir,
		"", "check", "--explain", "--format=json", "wide/doc.md")
	assert.Equal(t, 1, exit, "diagnostics should exit 1; stderr=%s", stderr)

	// JSON output goes to stderr by historical convention; tolerate either.
	body := stderr
	if body == "" || body[0] != '[' {
		body = stdout
	}
	if body == "" {
		body = stderr
	}

	var diags []map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &diags))
	require.NotEmpty(t, diags)

	var lineLength map[string]any
	for _, d := range diags {
		if d["name"] == "line-length" {
			lineLength = d
			break
		}
	}
	require.NotNil(t, lineLength, "expected a line-length diagnostic; got body=%s", body)
	exp, ok := lineLength["explanation"].(map[string]any)
	require.True(t, ok, "expected explanation object; got %v", lineLength)
	assert.Equal(t, "kinds.wide", exp["source"])
}

// TestE2E_KindsResolve runs the kinds subcommand against a temp config
// and parses its JSON output end-to-end.
func TestE2E_KindsResolve(t *testing.T) {
	dir := t.TempDir()
	isolateDir(t, dir)
	cfg := `rules:
  line-length:
    max: 80
kinds:
  wide:
    rules:
      line-length:
        max: 200
kind-assignment:
  - files: ["wide/*.md"]
    kinds: [wide]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".mdsmith.yml"), []byte(cfg), 0o644))
	wideDir := filepath.Join(dir, "wide")
	require.NoError(t, os.MkdirAll(wideDir, 0o755))
	target := filepath.Join(wideDir, "doc.md")
	require.NoError(t, os.WriteFile(target, []byte("# x\n"), 0o644))

	stdout, _, exit := runBinaryInDir(t, dir,
		"", "kinds", "resolve", "--json", "wide/doc.md")
	require.Equal(t, 0, exit)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	assert.Equal(t, "wide/doc.md", got["file"])
}
