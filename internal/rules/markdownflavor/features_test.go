package markdownflavor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlavor(t *testing.T) {
	tests := []struct {
		in   string
		want Flavor
		ok   bool
	}{
		{"commonmark", FlavorCommonMark, true},
		{"gfm", FlavorGFM, true},
		{"goldmark", FlavorGoldmark, true},
		{"GFM", 0, false},
		{"", 0, false},
		{"markdown", 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := ParseFlavor(tc.in)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestFlavorString(t *testing.T) {
	assert.Equal(t, "commonmark", FlavorCommonMark.String())
	assert.Equal(t, "gfm", FlavorGFM.String())
	assert.Equal(t, "goldmark", FlavorGoldmark.String())
}

func TestFeatureSupport(t *testing.T) {
	// CommonMark rejects every feature MDS034 tracks.
	for _, f := range AllFeatures() {
		assert.False(t, FlavorCommonMark.Supports(f),
			"CommonMark must reject %s", f.Name())
	}

	// GFM supports tables, task lists, strikethrough, bare-URL autolinks.
	assert.True(t, FlavorGFM.Supports(FeatureTables))
	assert.True(t, FlavorGFM.Supports(FeatureTaskLists))
	assert.True(t, FlavorGFM.Supports(FeatureStrikethrough))
	assert.True(t, FlavorGFM.Supports(FeatureBareURLAutolinks))

	// GFM rejects footnotes, definition lists, heading IDs, math, sub/sup, abbr.
	assert.False(t, FlavorGFM.Supports(FeatureFootnotes))
	assert.False(t, FlavorGFM.Supports(FeatureDefinitionLists))
	assert.False(t, FlavorGFM.Supports(FeatureHeadingIDs))
	assert.False(t, FlavorGFM.Supports(FeatureSuperscript))
	assert.False(t, FlavorGFM.Supports(FeatureSubscript))
	assert.False(t, FlavorGFM.Supports(FeatureMathBlock))
	assert.False(t, FlavorGFM.Supports(FeatureMathInline))
	assert.False(t, FlavorGFM.Supports(FeatureAbbreviations))

	// goldmark profile: GFM features + heading IDs.
	assert.True(t, FlavorGoldmark.Supports(FeatureTables))
	assert.True(t, FlavorGoldmark.Supports(FeatureTaskLists))
	assert.True(t, FlavorGoldmark.Supports(FeatureStrikethrough))
	assert.True(t, FlavorGoldmark.Supports(FeatureBareURLAutolinks))
	assert.True(t, FlavorGoldmark.Supports(FeatureHeadingIDs))
	assert.False(t, FlavorGoldmark.Supports(FeatureFootnotes))
	assert.False(t, FlavorGoldmark.Supports(FeatureDefinitionLists))
}

func TestAllFeaturesComplete(t *testing.T) {
	// Ensure AllFeatures enumerates exactly the 12 features we track.
	require.Len(t, AllFeatures(), 12)
}

func TestFeatureName(t *testing.T) {
	assert.Equal(t, "tables", FeatureTables.Name())
	assert.Equal(t, "task lists", FeatureTaskLists.Name())
	assert.Equal(t, "strikethrough", FeatureStrikethrough.Name())
	assert.Equal(t, "bare-URL autolinks", FeatureBareURLAutolinks.Name())
	assert.Equal(t, "footnotes", FeatureFootnotes.Name())
	assert.Equal(t, "definition lists", FeatureDefinitionLists.Name())
	assert.Equal(t, "heading IDs", FeatureHeadingIDs.Name())
	assert.Equal(t, "superscript", FeatureSuperscript.Name())
	assert.Equal(t, "subscript", FeatureSubscript.Name())
	assert.Equal(t, "math blocks", FeatureMathBlock.Name())
	assert.Equal(t, "inline math", FeatureMathInline.Name())
	assert.Equal(t, "abbreviations", FeatureAbbreviations.Name())
}
