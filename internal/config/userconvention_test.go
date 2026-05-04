package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Register markdown-flavor and no-inline-html so ByName lookups
	// resolve during user-convention validation.
	_ "github.com/jeduden/mdsmith/internal/rules/markdownflavor"
	_ "github.com/jeduden/mdsmith/internal/rules/noinlinehtml"
	_ "github.com/jeduden/mdsmith/internal/rules/listmarkerstyle"
)

// --- Unit tests for parseUserConventions ---

func TestParseUserConventions_ValidConvention(t *testing.T) {
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      no-inline-html:
        allow: [details, summary, kbd]
      list-marker-style:
        style: dash
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg.Conventions)
	c, ok := cfg.Conventions["our-team"]
	require.True(t, ok, "our-team convention must be parsed")
	assert.Equal(t, "gfm", c.Flavor)
	assert.Contains(t, c.Rules, "no-inline-html")
	assert.Contains(t, c.Rules, "list-marker-style")
}

func TestParseUserConventions_ReservedNameRejected(t *testing.T) {
	for _, name := range []string{"portable", "github", "plain"} {
		t.Run(name, func(t *testing.T) {
			yaml := "conventions:\n  " + name + ":\n    flavor: gfm\n    rules: {}\n"
			dir := t.TempDir()
			path := filepath.Join(dir, ".mdsmith.yml")
			require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

			_, err := Load(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), name)
			assert.Contains(t, err.Error(), "reserved")
		})
	}
}

func TestParseUserConventions_UnknownRuleRejected(t *testing.T) {
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      no-such-rule:
        style: dash
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "no-such-rule")
}

func TestParseUserConventions_InvalidRuleSettingRejected(t *testing.T) {
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      list-marker-style:
        not-a-real-setting: dash
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "list-marker-style")
}

func TestApplyConvention_UserConventionSelected(t *testing.T) {
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      list-marker-style:
        style: dash
convention: our-team
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg.ConventionPreset)
	rc, ok := cfg.ConventionPreset["list-marker-style"]
	require.True(t, ok, "list-marker-style must be in convention preset")
	assert.True(t, rc.Enabled)
	assert.Equal(t, "dash", rc.Settings["style"])
}

func TestApplyConvention_UnknownConventionListsUserDefined(t *testing.T) {
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules: {}
convention: bogus
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	// Must list both built-in and user-defined options.
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "github")
}

func TestApplyConvention_UserConventionTopLevelRulesWin(t *testing.T) {
	// User sets convention + overrides list-marker-style. User rule wins.
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      list-marker-style:
        style: dash
convention: our-team
rules:
  list-marker-style:
    style: asterisk
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	got := Effective(cfg, "doc.md", nil)
	rc, ok := got["list-marker-style"]
	require.True(t, ok)
	assert.Equal(t, "asterisk", rc.Settings["style"],
		"user top-level rules must win over user convention preset")
}

func TestApplyConvention_UserConventionInProvenanceLayer(t *testing.T) {
	yaml := `
conventions:
  our-team:
    flavor: gfm
    rules:
      list-marker-style:
        style: dash
convention: our-team
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	res := ResolveFile(cfg, "doc.md", nil)
	rr, ok := res.Rules["list-marker-style"]
	require.True(t, ok)

	var sources []string
	for _, l := range rr.Layers {
		sources = append(sources, l.Source)
	}
	// The convention layer source should carry the user suffix "(user)"
	// so it's distinguishable from built-in conventions.
	var found bool
	for _, s := range sources {
		if s == "convention.our-team (user)" {
			found = true
			break
		}
	}
	assert.True(t, found, "convention layer must have (user) suffix; got sources: %v", sources)
}

func TestMerge_PreservesUserConventions(t *testing.T) {
	loaded := &Config{
		Convention: "our-team",
		Conventions: map[string]UserConvention{
			"our-team": {Flavor: "gfm", Rules: map[string]RuleCfg{}},
		},
		ConventionPreset: map[string]RuleCfg{
			"list-marker-style": {Enabled: true, Settings: map[string]any{"style": "dash"}},
		},
	}
	merged := Merge(&Config{Rules: map[string]RuleCfg{}}, loaded)
	assert.Equal(t, "our-team", merged.Convention)
	require.Contains(t, merged.Conventions, "our-team")
	require.Contains(t, merged.ConventionPreset, "list-marker-style")
}
