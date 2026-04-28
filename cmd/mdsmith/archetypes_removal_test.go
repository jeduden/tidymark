package main_test

// Tests for plan 98: Remove archetypes CLI subcommand, config key, and
// internal/archetypes package.
//
// These tests cover the acceptance criteria:
//   - `mdsmith archetypes` exits 2 with "unknown command"
//   - `archetypes:` keys in .mdsmith.yml produce a config error directing
//     user to kinds:
//   - `mdsmith init` writes no archetypes: key

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArchetypesSubcommand_ExitsUnknownCommand verifies that
// `mdsmith archetypes` exits with code 2 and an "unknown command" message.
func TestArchetypesSubcommand_ExitsUnknownCommand(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".mdsmith.yml"), []byte("rules: {}\n"), 0o644))

	_, stderr, code := runBinaryInDir(t, dir, "", "archetypes")
	assert.Equal(t, 2, code)
	assert.Contains(t, strings.ToLower(stderr), "unknown command")
}

// TestArchetypesSubcommand_ListExitsUnknownCommand ensures that
// `mdsmith archetypes list` also exits 2.
func TestArchetypesSubcommand_ListExitsUnknownCommand(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".mdsmith.yml"), []byte("rules: {}\n"), 0o644))

	_, stderr, code := runBinaryInDir(t, dir, "", "archetypes", "list")
	assert.Equal(t, 2, code)
	assert.Contains(t, strings.ToLower(stderr), "unknown command")
}

// TestArchetypesConfigKey_ProducesError verifies that a config with
// archetypes: key causes a config load error directing user to kinds:.
func TestArchetypesConfigKey_ProducesError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	cfg := "archetypes:\n  roots:\n    - schemas\nrules: {}\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".mdsmith.yml"), []byte(cfg), 0o644))

	// Running any command that loads the config should fail with a helpful error.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc.md"),
		[]byte("# Hello\n"), 0o644))
	_, stderr, code := runBinaryInDir(t, dir, "", "check", "doc.md")
	assert.Equal(t, 2, code)
	assert.Contains(t, stderr, "kinds")
}

// TestInit_NoArchetypesKey verifies that mdsmith init generates a
// .mdsmith.yml that contains no archetypes: key and is accepted by
// the loader.
func TestInit_NoArchetypesKey(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	_, _, code := runBinaryInDir(t, dir, "", "init")
	require.Equal(t, 0, code)

	data, err := os.ReadFile(filepath.Join(dir, ".mdsmith.yml"))
	require.NoError(t, err)

	assert.NotContains(t, string(data), "archetypes",
		"generated .mdsmith.yml must not contain archetypes: key")

	// The generated file must be accepted by the loader — verify by running check.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.md"),
		[]byte("# Hello\n"), 0o644))
	_, stderr, code := runBinaryInDir(t, dir, "", "check", "test.md")
	assert.Equal(t, 0, code, "check should succeed, stderr: %s", stderr)
}
