package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Register rules so ApplySettings validation paths are exercised.
	_ "github.com/jeduden/mdsmith/internal/rules/markdownflavor"
	_ "github.com/jeduden/mdsmith/internal/rules/noinlinehtml"
)

// ---------------------------------------------------------------------------
// Tests for user-defined conventions: YAML parsing
// ---------------------------------------------------------------------------

func TestLoad_UserConventions_ParsedFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      no-inline-html:
        allow: [details, summary, kbd]
convention: our-team
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "our-team", cfg.Convention)
	require.NotNil(t, cfg.Conventions)
	require.Contains(t, cfg.Conventions, "our-team")
	assert.Equal(t, "gfm", cfg.Conventions["our-team"].Flavor)
	require.Contains(t, cfg.Conventions["our-team"].Rules, "no-inline-html")
}

// ---------------------------------------------------------------------------
// Tests for reserved-name validation
// ---------------------------------------------------------------------------

func TestLoad_UserConvention_ReservedNamePortable_IsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := "conventions:\n  portable:\n    flavor: commonmark\n    rules: {}\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "portable")
	assert.Contains(t, strings.ToLower(err.Error()), "reserved")
}

func TestLoad_UserConvention_ReservedNameGithub_IsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := "conventions:\n  github:\n    flavor: gfm\n    rules: {}\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "github")
	assert.Contains(t, strings.ToLower(err.Error()), "reserved")
}

func TestLoad_UserConvention_ReservedNamePlain_IsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := "conventions:\n  plain:\n    flavor: commonmark\n    rules: {}\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plain")
	assert.Contains(t, strings.ToLower(err.Error()), "reserved")
}

// ---------------------------------------------------------------------------
// Tests for unknown flavor validation
// ---------------------------------------------------------------------------

func TestLoad_UserConvention_UnknownFlavor_IsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := "conventions:\n  our-team:\n    flavor: bad-flavor\n    rules: {}\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad-flavor")
}

// ---------------------------------------------------------------------------
// Tests for rule settings validation
// ---------------------------------------------------------------------------

func TestLoad_UserConvention_UnknownRuleName_IsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := "conventions:\n  our-team:\n    flavor: gfm\n    rules:\n      no-such-rule:\n        setting: value\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no-such-rule")
}

func TestLoad_UserConvention_InvalidRuleSetting_IsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	// no-inline-html has valid settings; "totally-unknown" is not one
	yaml := "conventions:\n  our-team:\n    flavor: gfm\n    rules:\n" +
		"      no-inline-html:\n        totally-unknown: 42\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	// Error must name both the convention and the rule
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "no-inline-html")
}

// ---------------------------------------------------------------------------
// Tests for user-convention lookup (resolution order)
// ---------------------------------------------------------------------------

func TestApplyConvention_UserConventionSelected_SetsPreset(t *testing.T) {
	cfg := &Config{
		Convention: "our-team",
		Conventions: map[string]UserConventionBody{
			"our-team": {
				Flavor: "gfm",
				Rules: map[string]RuleCfg{
					"no-inline-html": {
						Enabled:  true,
						Settings: map[string]any{"allow": []any{"details"}},
					},
				},
			},
		},
	}
	require.NoError(t, applyConvention(cfg))
	require.NotNil(t, cfg.ConventionPreset)
	rc, ok := cfg.ConventionPreset["no-inline-html"]
	require.True(t, ok, "user convention preset must include no-inline-html")
	assert.True(t, rc.Enabled)
	assert.Equal(t, []any{"details"}, rc.Settings["allow"])
}

func TestApplyConvention_UnknownName_ListsUserAndBuiltins(t *testing.T) {
	// When neither user nor built-in table has the name, the error must
	// list both user and built-in names so the user sees all options.
	cfg := &Config{
		Convention: "bogus",
		Conventions: map[string]UserConventionBody{
			"our-team": {Flavor: "gfm"},
		},
	}
	err := applyConvention(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "github")
}

// ---------------------------------------------------------------------------
// Tests for user convention + top-level rules override
// ---------------------------------------------------------------------------

func TestEffectiveRules_UserConventionIsBaseLayerUnderUserRules(t *testing.T) {
	// User convention sets no-inline-html allow: [details].
	// User's top-level rules set no-inline-html allow: [sub, sup].
	// User list replaces convention list (MergeReplace default).
	cfg := &Config{
		Convention: "our-team",
		Conventions: map[string]UserConventionBody{
			"our-team": {
				Flavor: "gfm",
				Rules: map[string]RuleCfg{
					"no-inline-html": {
						Enabled:  true,
						Settings: map[string]any{"allow": []any{"details"}},
					},
				},
			},
		},
		Rules: map[string]RuleCfg{
			"no-inline-html": {
				Enabled:  true,
				Settings: map[string]any{"allow": []any{"sub", "sup"}},
			},
		},
		ExplicitRules: map[string]bool{"no-inline-html": true},
	}
	require.NoError(t, applyConvention(cfg))

	got := Effective(cfg, "doc.md", nil)
	rc, ok := got["no-inline-html"]
	require.True(t, ok)
	assert.True(t, rc.Enabled)
	assert.Equal(t, []any{"sub", "sup"}, rc.Settings["allow"],
		"user list replaces user-convention preset list")
}

// ---------------------------------------------------------------------------
// End-to-end: load + merge + check convention selects user convention
// ---------------------------------------------------------------------------

func TestLoad_UserConvention_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      no-inline-html:
        allow: [details, summary, kbd]
convention: our-team
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.NotNil(t, cfg.ConventionPreset)
	rc, ok := cfg.ConventionPreset["no-inline-html"]
	require.True(t, ok, "user convention preset should include no-inline-html")
	assert.True(t, rc.Enabled)
}

// ---------------------------------------------------------------------------
// Tests for kinds resolve: user convention shows "(user)" suffix
// (these are integration-level checks on the provenance layer source)
// ---------------------------------------------------------------------------

func TestProvenance_UserConventionLayerVisible(t *testing.T) {
	cfg := &Config{
		Convention: "our-team",
		Conventions: map[string]UserConventionBody{
			"our-team": {
				Flavor: "gfm",
				Rules: map[string]RuleCfg{
					"line-length": {Enabled: true, Settings: map[string]any{"max": 120}},
				},
			},
		},
	}
	require.NoError(t, applyConvention(cfg))

	res := ResolveFile(cfg, "doc.md", nil)
	rr, ok := res.Rules["line-length"]
	require.True(t, ok)

	var sources []string
	for _, l := range rr.Layers {
		sources = append(sources, l.Source)
	}
	// The convention layer source for a user convention must contain "(user)"
	assert.Contains(t, sources, "convention.our-team (user)")
}
