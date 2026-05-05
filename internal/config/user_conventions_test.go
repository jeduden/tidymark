package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Register rules so ByName lookups succeed during validation.
	_ "github.com/jeduden/mdsmith/internal/rules/markdownflavor"
)

// TestUserConvention_ValidDefinition verifies that a valid user-defined
// convention is accepted and its preset is applied when selected.
func TestUserConvention_ValidDefinition(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      markdown-flavor:
        flavor: gfm
convention: our-team
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "our-team", cfg.Convention)
	require.NotNil(t, cfg.ConventionPreset)
	mf, ok := cfg.ConventionPreset["markdown-flavor"]
	require.True(t, ok, "preset must contain markdown-flavor")
	assert.Equal(t, "gfm", mf.Settings["flavor"])
}

// TestUserConvention_ReservedNamePortableRejected verifies that using a
// built-in name as a user-defined convention name produces a config error.
func TestUserConvention_ReservedNamePortableRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  portable:
    flavor: gfm
    rules: {}
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "portable")
	assert.Contains(t, err.Error(), "reserved")
}

// TestUserConvention_ReservedNameGithubRejected verifies "github" is reserved.
func TestUserConvention_ReservedNameGithubRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  github:
    flavor: gfm
    rules: {}
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "github")
	assert.Contains(t, err.Error(), "reserved")
}

// TestUserConvention_ReservedNamePlainRejected verifies "plain" is reserved.
func TestUserConvention_ReservedNamePlainRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  plain:
    flavor: commonmark
    rules: {}
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plain")
	assert.Contains(t, err.Error(), "reserved")
}

// TestUserConvention_UnknownRuleNameRejected verifies that a user convention
// referencing a rule name that does not exist produces a config error.
func TestUserConvention_UnknownRuleNameRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      no-such-rule: true
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "no-such-rule")
}

// TestUserConvention_InvalidRuleSettingRejected verifies that invalid rule
// settings inside a user convention produce a config error naming the
// convention and the rule.
func TestUserConvention_InvalidRuleSettingRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      markdown-flavor:
        flavor: invalid-flavor-value
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "markdown-flavor")
}

// TestUserConvention_UnknownConventionListsBothSets verifies that selecting
// an unknown convention name produces an error listing both built-in and
// user-defined names.
func TestUserConvention_UnknownConventionListsBothSets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  my-team:
    flavor: gfm
    rules: {}
convention: bogus
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	// Must list built-in names
	assert.Contains(t, err.Error(), "github")
	assert.Contains(t, err.Error(), "portable")
	assert.Contains(t, err.Error(), "plain")
	// Must list user-defined name
	assert.Contains(t, err.Error(), "my-team")
}

// TestUserConvention_TopLevelRulesOverridePreset verifies that a top-level
// rules: block overrides user convention presets via deep-merge. Tests a
// non-flavor setting so the flavor-mismatch guard does not interfere.
func TestUserConvention_TopLevelRulesOverridePreset(t *testing.T) {
	// Convention presets horizontal-rule-style with style: dash.
	// User overrides it with style: asterisk at the top-level rules block.
	cfg := &Config{
		Convention: "our-team",
		Conventions: map[string]UserConventionBody{
			"our-team": {
				Flavor: "gfm",
				Rules: map[string]RuleCfg{
					"horizontal-rule-style": {
						Enabled:  true,
						Settings: map[string]any{"style": "dash"},
					},
				},
			},
		},
		Rules: map[string]RuleCfg{
			"horizontal-rule-style": {
				Enabled:  true,
				Settings: map[string]any{"style": "asterisk"},
			},
		},
		ExplicitRules: map[string]bool{"horizontal-rule-style": true},
	}
	require.NoError(t, applyConvention(cfg))

	got := Effective(cfg, "doc.md", nil)
	hr, ok := got["horizontal-rule-style"]
	require.True(t, ok)
	// User's top-level rules win over convention preset
	assert.Equal(t, "asterisk", hr.Settings["style"])
}

// TestUserConvention_InvalidFlavorRejected verifies that an unknown flavor
// value in a user convention produces a config error.
func TestUserConvention_InvalidFlavorRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `
conventions:
  our-team:
    flavor: not-a-real-flavor
    rules: {}
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "flavor")
}

// TestUserConvention_ProvenanceShowsUserSuffix verifies that a user-defined
// convention produces a layer source with a "(user)" suffix in the
// provenance chain, distinguishing it from built-in conventions.
func TestUserConvention_ProvenanceShowsUserSuffix(t *testing.T) {
	cfg := &Config{
		Convention: "our-team",
		Conventions: map[string]UserConventionBody{
			"our-team": {
				Flavor: "gfm",
				Rules: map[string]RuleCfg{
					"horizontal-rule-style": {
						Enabled:  true,
						Settings: map[string]any{"style": "dash"},
					},
				},
			},
		},
	}
	require.NoError(t, applyConvention(cfg))
	assert.True(t, cfg.ConventionIsUser, "ConventionIsUser should be true for user conventions")

	res := ResolveFile(cfg, "doc.md", nil)
	rr, ok := res.Rules["horizontal-rule-style"]
	require.True(t, ok)

	var sources []string
	for _, l := range rr.Layers {
		sources = append(sources, l.Source)
	}
	require.Contains(t, sources, "convention.our-team (user)",
		"user convention layer must have (user) suffix in provenance")
}

// TestBuiltInConvention_ProvenanceNoUserSuffix verifies that a built-in
// convention does NOT get the "(user)" suffix in provenance output.
func TestBuiltInConvention_ProvenanceNoUserSuffix(t *testing.T) {
	cfg := &Config{Convention: "portable"}
	require.NoError(t, applyConvention(cfg))
	assert.False(t, cfg.ConventionIsUser, "ConventionIsUser should be false for built-in conventions")

	res := ResolveFile(cfg, "doc.md", nil)
	rr, ok := res.Rules["horizontal-rule-style"]
	require.True(t, ok)

	var sources []string
	for _, l := range rr.Layers {
		sources = append(sources, l.Source)
	}
	require.Contains(t, sources, "convention.portable",
		"built-in convention layer must not have (user) suffix")
	for _, s := range sources {
		assert.NotContains(t, s, "(user)", "built-in convention layer must not have (user) suffix")
	}
}
