package markdownflavor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jeduden/mdsmith/internal/convention"
)

// assertSupports checks every feature in the supported set is
// accepted by flavor and every feature not in that set is rejected.
func assertSupports(t *testing.T, f convention.Flavor, supported ...Feature) {
	t.Helper()
	want := map[Feature]bool{}
	for _, feat := range supported {
		want[feat] = true
	}
	for _, feat := range AllFeatures() {
		got := Supports(f, feat)
		assert.Equal(t, want[feat], got,
			"flavor %s feature %s: want=%v got=%v",
			f.String(), feat.Name(), want[feat], got)
	}
}

func TestFeatureNameUnknownIsEmpty(t *testing.T) {
	assert.Equal(t, "", Feature(999).Name())
}

func TestFeatureSupportCommonMark(t *testing.T) {
	assertSupports(t, convention.FlavorCommonMark)
}

func TestFeatureSupportGFM(t *testing.T) {
	assertSupports(t, convention.FlavorGFM,
		FeatureTables, FeatureTaskLists, FeatureStrikethrough,
		FeatureBareURLAutolinks, FeatureGitHubAlerts)
}

func TestFeatureSupportGoldmark(t *testing.T) {
	assertSupports(t, convention.FlavorGoldmark,
		FeatureTables, FeatureTaskLists, FeatureStrikethrough,
		FeatureBareURLAutolinks, FeatureHeadingIDs)
}

func TestFeatureSupportAny(t *testing.T) {
	assertSupports(t, convention.FlavorAny, AllFeatures()...)
}

func TestFeatureSupportPandoc(t *testing.T) {
	assertSupports(t, convention.FlavorPandoc,
		FeatureTables, FeatureTaskLists, FeatureStrikethrough,
		FeatureBareURLAutolinks, FeatureFootnotes, FeatureDefinitionLists,
		FeatureHeadingIDs, FeatureSuperscript, FeatureSubscript,
		FeatureMathBlock, FeatureMathInline)
}

func TestFeatureSupportPHPExtra(t *testing.T) {
	assertSupports(t, convention.FlavorPHPExtra,
		FeatureTables, FeatureFootnotes, FeatureDefinitionLists,
		FeatureHeadingIDs, FeatureAbbreviations)
}

func TestFeatureSupportMultiMarkdown(t *testing.T) {
	assertSupports(t, convention.FlavorMultiMarkdown,
		FeatureTables, FeatureFootnotes, FeatureDefinitionLists,
		FeatureHeadingIDs, FeatureAbbreviations,
		FeatureMathBlock, FeatureMathInline)
}

func TestFeatureSupportMyST(t *testing.T) {
	assertSupports(t, convention.FlavorMyST,
		FeatureTables, FeatureStrikethrough, FeatureFootnotes,
		FeatureDefinitionLists, FeatureHeadingIDs,
		FeatureMathBlock, FeatureMathInline)
}

func TestAllFeaturesComplete(t *testing.T) {
	// Ensure AllFeatures enumerates exactly the 13 features we track.
	require.Len(t, AllFeatures(), 13)
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
	assert.Equal(t, "github alerts", FeatureGitHubAlerts.Name())
}
