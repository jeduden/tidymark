package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKindsJSONSchema_Resolve documents the exact set of top-level keys
// produced by `kinds resolve --json`. If a future change adds, removes,
// or renames a top-level key, this test must be updated and the
// docs/reference/cli.md JSON schema block must be reviewed in the same
// commit. Treat it as a public-API guard for tooling that consumes
// this output.
func TestKindsJSONSchema_Resolve(t *testing.T) {
	dir := writeYAMLConfig(t, `
kinds:
  wide:
    rules:
      line-length:
        max: 200
kind-assignment:
  - files: ["wide/*.md"]
    kinds: [wide]
`)
	mdDir := filepath.Join(dir, "wide")
	require.NoError(t, os.MkdirAll(mdDir, 0o755))
	target := filepath.Join(mdDir, "doc.md")
	require.NoError(t, os.WriteFile(target, []byte("# Title\n"), 0o644))

	out := captureStdout(func() {
		code := runKindsResolve([]string{"--json", "wide/doc.md"})
		require.Equal(t, 0, code)
	})

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got))

	wantKeys := []string{"file", "kinds", "rules", "categories", "explicit"}
	gotKeys := topLevelKeys(got)
	assert.Equal(t, wantKeys, gotKeys,
		"top-level keys of kinds resolve --json are part of the public schema")

	// Inspect kinds[0] shape.
	kinds, ok := got["kinds"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, kinds)
	kind0 := kinds[0].(map[string]any)
	assert.Equal(t, []string{"name", "sources"}, sortedKeys(kind0))

	// Inspect a rule's leaf shape.
	rules := got["rules"].(map[string]any)
	rule := rules["line-length"].(map[string]any)
	assert.Equal(t, []string{"final", "leaves"}, sortedKeys(rule))
	leaves := rule["leaves"].(map[string]any)
	for _, leaf := range leaves {
		l := leaf.(map[string]any)
		assert.Equal(t, []string{"chain", "final", "winning_source"}, sortedKeys(l))
		chain := l["chain"].([]any)
		require.NotEmpty(t, chain)
		entry := chain[0].(map[string]any)
		// "value" is omitempty when the layer did not touch the leaf,
		// so we only require the always-present keys here.
		for _, k := range []string{"layer", "source", "touched"} {
			_, ok := entry[k]
			assert.True(t, ok, "chain entry missing required key %q", k)
		}
	}
}

// TestKindsJSONSchema_Why guards the shape of the why output.
func TestKindsJSONSchema_Why(t *testing.T) {
	dir := writeYAMLConfig(t, `
rules:
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
`)
	mdDir := filepath.Join(dir, "wide")
	require.NoError(t, os.MkdirAll(mdDir, 0o755))
	target := filepath.Join(mdDir, "doc.md")
	require.NoError(t, os.WriteFile(target, []byte("# Title\n"), 0o644))

	out := captureStdout(func() {
		code := runKindsWhy([]string{"--json", "wide/doc.md", "line-length"})
		require.Equal(t, 0, code)
	})

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got))

	assert.Equal(t, []string{"chain", "file", "final", "rule"}, sortedKeys(got))
	chain := got["chain"].([]any)
	require.NotEmpty(t, chain)
}

// TestKindsJSONSchema_ListAndShow guards the shape of list/show.
func TestKindsJSONSchema_ListAndShow(t *testing.T) {
	writeYAMLConfig(t, `
kinds:
  plan:
    rules:
      line-length: false
`)
	listOut := captureStdout(func() {
		code := runKindsList([]string{"--json"})
		require.Equal(t, 0, code)
	})
	var listGot map[string]any
	require.NoError(t, json.Unmarshal([]byte(listOut), &listGot))
	assert.Equal(t, []string{"kinds"}, sortedKeys(listGot))
	items := listGot["kinds"].([]any)
	require.NotEmpty(t, items)
	first := items[0].(map[string]any)
	assert.Equal(t, []string{"body", "name"}, sortedKeys(first))

	showOut := captureStdout(func() {
		code := runKindsShow([]string{"--json", "plan"})
		require.Equal(t, 0, code)
	})
	var showGot map[string]any
	require.NoError(t, json.Unmarshal([]byte(showOut), &showGot))
	assert.Equal(t, []string{"body", "name"}, sortedKeys(showGot))
}

// topLevelKeys returns the sorted list of top-level keys of m. Used as
// a stable shape descriptor for schema regression tests.
func topLevelKeys(m map[string]any) []string {
	keys := []string{"file", "kinds", "rules", "categories", "explicit"}
	have := map[string]bool{}
	for k := range m {
		have[k] = true
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if have[k] {
			out = append(out, k)
		}
	}
	// Surface unexpected keys at the end so failures are descriptive.
	for k := range m {
		if !contains(keys, k) {
			out = append(out, k)
		}
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
