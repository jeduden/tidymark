package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Register markdown-flavor so rule.ByName lookups resolve while
	// the convention mechanism is exercised.
	_ "github.com/jeduden/mdsmith/internal/rules/markdownflavor"
)

// TestLoad_UserConventionParsed verifies that a conventions: block in
// .mdsmith.yml is parsed and the user-defined convention is resolvable
// via convention: selector.
func TestLoad_UserConventionParsed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `conventions:
  our-team:
    flavor: gfm
    rules:
      list-marker-style:
        style: dash
convention: our-team
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "our-team", cfg.Convention)
	require.NotNil(t, cfg.ConventionPreset)
	lm, ok := cfg.ConventionPreset["list-marker-style"]
	require.True(t, ok, "preset must contain list-marker-style")
	assert.True(t, lm.Enabled)
	assert.Equal(t, "dash", lm.Settings["style"])
}

// TestLoad_UserConventionReservedNameRejected verifies that defining a
// user convention with a built-in name (portable, github, plain)
// produces a config error.
func TestLoad_UserConventionReservedNameRejected(t *testing.T) {
	for _, name := range []string{"portable", "github", "plain"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".mdsmith.yml")
			yaml := "conventions:\n  " + name + ":\n    flavor: gfm\n"
			require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

			_, err := Load(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), name)
			assert.True(t,
				strings.Contains(err.Error(), "reserved") ||
					strings.Contains(err.Error(), "built-in"),
				"error must mention reserved or built-in, got: %s", err.Error())
		})
	}
}

// TestLoad_UnknownConventionListsUserAndBuiltinNames verifies that an
// unknown convention: selector includes both built-in and user-defined
// convention names in the error message.
func TestLoad_UnknownConventionListsUserAndBuiltinNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `conventions:
  our-team:
    flavor: gfm
convention: bogus
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	// Must list built-in names.
	assert.Contains(t, err.Error(), "portable")
	assert.Contains(t, err.Error(), "github")
	assert.Contains(t, err.Error(), "plain")
	// Must list user-defined name.
	assert.Contains(t, err.Error(), "our-team")
}

// TestLoad_UserConventionUnknownRuleNameRejected verifies that a rule
// name inside a user convention that does not name a registered rule
// produces a config error naming both the convention and the rule.
func TestLoad_UserConventionUnknownRuleNameRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `conventions:
  our-team:
    flavor: gfm
    rules:
      no-such-rule:
        enabled: true
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "no-such-rule")
}

// TestLoad_UserConventionInvalidRuleSettingRejected verifies that an
// invalid rule setting inside a user convention produces a config error
// naming both the convention and the rule.
func TestLoad_UserConventionInvalidRuleSettingRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	// list-marker-style exists and is registered; "invalid-key" is not
	// a valid setting for it.
	yaml := `conventions:
  our-team:
    flavor: gfm
    rules:
      list-marker-style:
        invalid-key: oops
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "list-marker-style")
}

// TestLoad_UserConventionInvalidFlavorRejected verifies that an invalid
// flavor value in a user convention produces a config error.
func TestLoad_UserConventionInvalidFlavorRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `conventions:
  our-team:
    flavor: not-a-flavor
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "our-team")
	assert.Contains(t, err.Error(), "not-a-flavor")
}

// TestLoad_UserConventionTopLevelRulesOverrideWins verifies that a
// top-level rules: override beats the user-defined convention preset
// via deep-merge (the convention is the base layer, user rules win).
// Uses Merge(Defaults(), loaded) so ExplicitRules is populated, which
// is how the CLI uses configs.
func TestLoad_UserConventionTopLevelRulesOverrideWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mdsmith.yml")
	yaml := `conventions:
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
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	loaded, err := Load(path)
	require.NoError(t, err)
	cfg := Merge(Defaults(), loaded)

	got := Effective(cfg, "doc.md", nil)
	rc, ok := got["list-marker-style"]
	require.True(t, ok)
	assert.Equal(t, "asterisk", rc.Settings["style"],
		"user top-level rules must override user convention preset")
}

// TestConventionsMerge_PreservesUserConventions verifies that Merge
// carries the UserConventions map from the loaded config to the merged
// result.
func TestConventionsMerge_PreservesUserConventions(t *testing.T) {
	loaded := &Config{
		Convention: "our-team",
		UserConventions: map[string]UserConventionBody{
			"our-team": {
				Flavor: "gfm",
				Rules: map[string]RuleCfg{
					"list-marker-style": {Enabled: true, Settings: map[string]any{"style": "dash"}},
				},
			},
		},
	}
	merged := Merge(&Config{Rules: map[string]RuleCfg{}}, loaded)
	require.NotNil(t, merged.UserConventions)
	body, ok := merged.UserConventions["our-team"]
	require.True(t, ok)
	assert.Equal(t, "gfm", body.Flavor)
}

// TestProvenance_UserConventionLayerHasSuffix verifies that a
// user-defined convention is labeled with "(user)" in the layer source
// string so `mdsmith kinds resolve` can distinguish it from built-ins.
func TestProvenance_UserConventionLayerHasSuffix(t *testing.T) {
	cfg := &Config{
		Convention: "our-team",
		UserConventions: map[string]UserConventionBody{
			"our-team": {Flavor: "gfm"},
		},
		ConventionPreset: map[string]RuleCfg{
			"line-length": {Enabled: true, Settings: map[string]any{"max": 80}},
		},
	}

	res := ResolveFile(cfg, "doc.md", nil)
	rr, ok := res.Rules["line-length"]
	require.True(t, ok, "line-length must appear in resolution")

	var sources []string
	for _, l := range rr.Layers {
		sources = append(sources, l.Source)
	}
	require.Contains(t, sources, "convention.our-team (user)",
		"user convention layer must have (user) suffix; got %v", sources)
}

// TestProvenance_BuiltinConventionLayerNoSuffix verifies that a built-in
// convention does NOT have a (user) suffix in the layer source.
func TestProvenance_BuiltinConventionLayerNoSuffix(t *testing.T) {
	cfg := &Config{
		Convention: "portable",
		ConventionPreset: map[string]RuleCfg{
			"line-length": {Enabled: true, Settings: map[string]any{"max": 80}},
		},
	}

	res := ResolveFile(cfg, "doc.md", nil)
	rr, ok := res.Rules["line-length"]
	require.True(t, ok, "line-length must appear in resolution")

	for _, l := range rr.Layers {
		assert.NotContains(t, l.Source, "(user)",
			"built-in convention must not have (user) suffix")
	}
	var sources []string
	for _, l := range rr.Layers {
		sources = append(sources, l.Source)
	}
	require.Contains(t, sources, "convention.portable")
}
