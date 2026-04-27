package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEffectiveWithProvenanceTracksDefaults verifies the default layer
// is recorded for every leaf supplied by the top-level rules.
func TestEffectiveWithProvenanceTracksDefaults(t *testing.T) {
	cfg := &Config{
		Rules: map[string]RuleCfg{
			"first-line-heading": {
				Enabled:  true,
				Settings: map[string]any{"level": 1},
			},
		},
	}
	_, prov := EffectiveWithProvenance(cfg, "doc.md", nil)

	rp, ok := prov["first-line-heading"]
	require.True(t, ok)
	require.Len(t, rp.Settings["level"], 1)
	assert.Equal(t, LayerRef{Kind: LayerDefault},
		rp.Settings["level"][0].Layer)
	assert.Equal(t, 1, rp.Settings["level"][0].Value)
}

// TestEffectiveWithProvenanceTracksKindLayer verifies a kind that
// touches one leaf adds a LayerKindBody entry to that leaf's chain
// while sibling leaves keep their default-layer source.
func TestEffectiveWithProvenanceTracksKindLayer(t *testing.T) {
	cfg := &Config{
		Rules: map[string]RuleCfg{
			"first-line-heading": {
				Enabled: true,
				Settings: map[string]any{
					"level":        1,
					"placeholders": []any{},
				},
			},
		},
		Kinds: map[string]KindBody{
			"plan": {Rules: map[string]RuleCfg{
				"first-line-heading": {
					Enabled:  true,
					Settings: map[string]any{"level": 2},
				},
			}},
		},
	}
	_, prov := EffectiveWithProvenance(cfg, "doc.md", []string{"plan"})

	rp := prov["first-line-heading"]
	require.Len(t, rp.Settings["level"], 2)
	assert.Equal(t, LayerRef{Kind: LayerDefault},
		rp.Settings["level"][0].Layer)
	assert.Equal(t,
		LayerRef{Kind: LayerKindBody, Name: "plan"},
		rp.Settings["level"][1].Layer)

	// placeholders untouched by the kind: still single-layer default.
	require.Len(t, rp.Settings["placeholders"], 1)
	assert.Equal(t, LayerRef{Kind: LayerDefault},
		rp.Settings["placeholders"][0].Layer)
}

// TestEffectiveWithProvenanceTracksOverrideIndex verifies override
// entries record their position in the overrides: list.
func TestEffectiveWithProvenanceTracksOverrideIndex(t *testing.T) {
	cfg := &Config{
		Rules: map[string]RuleCfg{
			"first-line-heading": {
				Enabled:  true,
				Settings: map[string]any{"level": 1},
			},
		},
		Overrides: []Override{
			{Files: []string{"other/*.md"}, Rules: map[string]RuleCfg{
				"first-line-heading": {
					Enabled:  true,
					Settings: map[string]any{"level": 4},
				},
			}},
			{Files: []string{"docs/*.md"}, Rules: map[string]RuleCfg{
				"first-line-heading": {
					Enabled:  true,
					Settings: map[string]any{"level": 3},
				},
			}},
		},
	}
	_, prov := EffectiveWithProvenance(cfg, "docs/foo.md", nil)
	rp := prov["first-line-heading"]
	chain := rp.Settings["level"]
	require.Len(t, chain, 2, "default + matching override")
	assert.Equal(t, LayerRef{Kind: LayerDefault}, chain[0].Layer)
	assert.Equal(t,
		LayerRef{Kind: LayerOverride, Index: 1},
		chain[1].Layer,
		"second override entry's index is 1, not 0")
}

// TestWinningLayerForLeaf verifies WinningLayer returns the last layer
// that contributed to a leaf.
func TestWinningLayerForLeaf(t *testing.T) {
	cfg := &Config{
		Rules: map[string]RuleCfg{
			"first-line-heading": {
				Enabled:  true,
				Settings: map[string]any{"level": 1},
			},
		},
		Kinds: map[string]KindBody{
			"plan": {Rules: map[string]RuleCfg{
				"first-line-heading": {
					Enabled:  true,
					Settings: map[string]any{"level": 2},
				},
			}},
		},
		Overrides: []Override{
			{Files: []string{"docs/*.md"}, Rules: map[string]RuleCfg{
				"first-line-heading": {
					Enabled:  true,
					Settings: map[string]any{"level": 3},
				},
			}},
		},
	}
	_, prov := EffectiveWithProvenance(cfg, "docs/foo.md", []string{"plan"})
	rp := prov["first-line-heading"]
	winner := rp.WinningLayer("level")
	assert.Equal(t, LayerOverride, winner.Kind)
	assert.Equal(t, 0, winner.Index)
}

// TestLayerRefStringRendering verifies the human-readable form used by
// `mdsmith kinds resolve`.
func TestLayerRefStringRendering(t *testing.T) {
	tests := []struct {
		ref  LayerRef
		want string
	}{
		{LayerRef{Kind: LayerDefault}, "default"},
		{LayerRef{Kind: LayerKindBody, Name: "plan"}, "kinds.plan"},
		{LayerRef{Kind: LayerOverride, Index: 2}, "overrides[2]"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.ref.String())
	}
}
