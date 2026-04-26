package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKinds_ParseFromYAML verifies that a `kinds:` map at the top level of
// the config parses into Config.Kinds with the same shape as an Override
// entry minus `files:` (a `rules:` map and optional `categories:` map).
func TestKinds_ParseFromYAML(t *testing.T) {
	yml := `
kinds:
  plan:
    rules:
      line-length:
        max: 100
      heading-style: false
  proto:
    rules:
      no-bare-urls: false
    categories:
      heading: false
`
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(yml), 0o644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err, "Load returned error: %v", err)

	require.Len(t, cfg.Kinds, 2, "expected 2 kinds, got %d", len(cfg.Kinds))

	plan, ok := cfg.Kinds["plan"]
	require.True(t, ok, "expected plan kind to exist")
	assert.True(t, plan.Rules["line-length"].Enabled)
	assert.Equal(t, 100, plan.Rules["line-length"].Settings["max"])
	assert.False(t, plan.Rules["heading-style"].Enabled)

	proto, ok := cfg.Kinds["proto"]
	require.True(t, ok, "expected proto kind to exist")
	assert.False(t, proto.Rules["no-bare-urls"].Enabled)
	assert.False(t, proto.Categories["heading"])
}

// TestKindAssignment_ParseFromYAML verifies that a `kind-assignment:` list
// at the top level parses into Config.KindAssignment with `files:` and
// `kinds:` slices.
func TestKindAssignment_ParseFromYAML(t *testing.T) {
	yml := `
kinds:
  plan:
    rules:
      line-length: false
  proto:
    rules:
      no-bare-urls: false
kind-assignment:
  - files: ["plan/[0-9]*_*.md"]
    kinds: [plan]
  - files: ["**/proto.md"]
    kinds: [proto]
`
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".mdsmith.yml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(yml), 0o644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	require.Len(t, cfg.KindAssignment, 2)
	assert.Equal(t, []string{"plan/[0-9]*_*.md"}, cfg.KindAssignment[0].Files)
	assert.Equal(t, []string{"plan"}, cfg.KindAssignment[0].Kinds)
	assert.Equal(t, []string{"**/proto.md"}, cfg.KindAssignment[1].Files)
	assert.Equal(t, []string{"proto"}, cfg.KindAssignment[1].Kinds)
}

// TestEffectiveKindsForFile_FromAssignment verifies that EffectiveKinds
// returns the kinds bound to a file via kind-assignment glob entries,
// in config order with duplicates dropped.
func TestEffectiveKindsForFile_FromAssignment(t *testing.T) {
	cfg := &Config{
		Kinds: map[string]Kind{
			"plan":  {Rules: map[string]RuleCfg{"line-length": {Enabled: false}}},
			"proto": {Rules: map[string]RuleCfg{"no-bare-urls": {Enabled: false}}},
		},
		KindAssignment: []KindAssignment{
			{Files: []string{"plan/*.md"}, Kinds: []string{"plan"}},
			{Files: []string{"proto.md"}, Kinds: []string{"proto"}},
		},
	}

	got, err := EffectiveKinds(cfg, "plan/01_x.md", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"plan"}, got)

	got2, err := EffectiveKinds(cfg, "proto.md", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"proto"}, got2)

	// Non-matching file gets empty list.
	got3, err := EffectiveKinds(cfg, "README.md", nil)
	require.NoError(t, err)
	assert.Empty(t, got3)
}

// TestEffectiveKinds_FromFrontMatterFirst verifies that front-matter kinds
// come before kind-assignment kinds in the effective list.
func TestEffectiveKinds_FromFrontMatterFirst(t *testing.T) {
	cfg := &Config{
		Kinds: map[string]Kind{
			"a": {Rules: map[string]RuleCfg{"line-length": {Enabled: false}}},
			"b": {Rules: map[string]RuleCfg{"no-bare-urls": {Enabled: false}}},
		},
		KindAssignment: []KindAssignment{
			{Files: []string{"x.md"}, Kinds: []string{"b"}},
		},
	}

	got, err := EffectiveKinds(cfg, "x.md", []string{"a"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, got)
}

// TestEffectiveKinds_DropsDuplicates verifies that duplicate kind names
// are dropped after their first occurrence in the effective list.
func TestEffectiveKinds_DropsDuplicates(t *testing.T) {
	cfg := &Config{
		Kinds: map[string]Kind{
			"a": {Rules: map[string]RuleCfg{"line-length": {Enabled: false}}},
			"b": {Rules: map[string]RuleCfg{"no-bare-urls": {Enabled: false}}},
		},
		KindAssignment: []KindAssignment{
			{Files: []string{"x.md"}, Kinds: []string{"a", "b"}},
			{Files: []string{"x.md"}, Kinds: []string{"a"}},
		},
	}

	got, err := EffectiveKinds(cfg, "x.md", []string{"b", "a"})
	require.NoError(t, err)
	assert.Equal(t, []string{"b", "a"}, got)
}

// TestEffectiveKinds_UndeclaredFromFrontMatterErrors verifies that a
// front-matter `kinds:` referencing an undeclared kind name produces an
// error containing the offending name.
func TestEffectiveKinds_UndeclaredFromFrontMatterErrors(t *testing.T) {
	cfg := &Config{
		Kinds: map[string]Kind{
			"a": {Rules: map[string]RuleCfg{"line-length": {Enabled: false}}},
		},
	}

	_, err := EffectiveKinds(cfg, "x.md", []string{"missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

// TestEffectiveKinds_UndeclaredFromAssignmentErrors verifies that a
// kind-assignment entry referencing an undeclared kind name produces an
// error containing the offending name.
func TestEffectiveKinds_UndeclaredFromAssignmentErrors(t *testing.T) {
	cfg := &Config{
		Kinds: map[string]Kind{
			"a": {Rules: map[string]RuleCfg{"line-length": {Enabled: false}}},
		},
		KindAssignment: []KindAssignment{
			{Files: []string{"x.md"}, Kinds: []string{"ghost"}},
		},
	}

	_, err := EffectiveKinds(cfg, "x.md", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}

// TestEffective_KindsAppliedBeforeOverrides verifies that the effective rule
// config for a file applies kind-resolved settings before file-glob
// overrides — overrides win on conflicts because they come later.
func TestEffective_KindsAppliedBeforeOverrides(t *testing.T) {
	cfg := &Config{
		Rules: map[string]RuleCfg{
			"line-length": {Enabled: true, Settings: map[string]any{"max": 80}},
		},
		Kinds: map[string]Kind{
			"plan": {Rules: map[string]RuleCfg{
				"line-length": {Enabled: true, Settings: map[string]any{"max": 100}},
			}},
		},
		KindAssignment: []KindAssignment{
			{Files: []string{"plan/*.md"}, Kinds: []string{"plan"}},
		},
		Overrides: []Override{
			{
				Files: []string{"plan/special.md"},
				Rules: map[string]RuleCfg{
					"line-length": {Enabled: true, Settings: map[string]any{"max": 200}},
				},
			},
		},
	}

	// Plan file gets kind settings.
	eff, err := EffectiveWithKinds(cfg, "plan/01.md", nil)
	require.NoError(t, err)
	assert.Equal(t, 100, eff["line-length"].Settings["max"])

	// Plan file with matching override gets the override.
	eff2, err := EffectiveWithKinds(cfg, "plan/special.md", nil)
	require.NoError(t, err)
	assert.Equal(t, 200, eff2["line-length"].Settings["max"])

	// Non-plan file keeps top-level setting.
	eff3, err := EffectiveWithKinds(cfg, "README.md", nil)
	require.NoError(t, err)
	assert.Equal(t, 80, eff3["line-length"].Settings["max"])
}

// TestEffective_KindOrderBlockReplacesRule verifies that when two kinds set
// the same rule, the kind appearing later in the effective list replaces
// the earlier kind's entire rule config block.
func TestEffective_KindOrderBlockReplacesRule(t *testing.T) {
	cfg := &Config{
		Rules: map[string]RuleCfg{
			"line-length": {Enabled: true, Settings: map[string]any{"max": 80}},
		},
		Kinds: map[string]Kind{
			"a": {Rules: map[string]RuleCfg{
				"line-length": {Enabled: true, Settings: map[string]any{"max": 100, "exclude": []any{"x"}}},
			}},
			"b": {Rules: map[string]RuleCfg{
				"line-length": {Enabled: true, Settings: map[string]any{"max": 120}},
			}},
		},
	}

	// Front-matter order: a then b. b's block replaces a's entire rule cfg.
	eff, err := EffectiveWithKinds(cfg, "x.md", []string{"a", "b"})
	require.NoError(t, err)
	rc := eff["line-length"]
	assert.Equal(t, 120, rc.Settings["max"])
	_, hasExclude := rc.Settings["exclude"]
	assert.False(t, hasExclude, "later kind should fully replace earlier kind's rule block")
}

// TestEffective_KindsComposeOnDifferentRules verifies that two kinds that
// configure different rules both contribute their settings.
func TestEffective_KindsComposeOnDifferentRules(t *testing.T) {
	cfg := &Config{
		Rules: map[string]RuleCfg{
			"line-length": {Enabled: true},
			"no-bare-urls": {Enabled: true},
		},
		Kinds: map[string]Kind{
			"a": {Rules: map[string]RuleCfg{
				"line-length": {Enabled: false},
			}},
			"b": {Rules: map[string]RuleCfg{
				"no-bare-urls": {Enabled: false},
			}},
		},
	}

	eff, err := EffectiveWithKinds(cfg, "x.md", []string{"a", "b"})
	require.NoError(t, err)
	assert.False(t, eff["line-length"].Enabled, "kind a should disable line-length")
	assert.False(t, eff["no-bare-urls"].Enabled, "kind b should disable no-bare-urls")
}
