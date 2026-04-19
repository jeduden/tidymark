// Package markdownflavor implements MDS034, which validates Markdown
// against a declared target flavor (commonmark, gfm, goldmark) and
// flags syntax the target renderer will not understand.
package markdownflavor

// Flavor identifies a target Markdown flavor.
type Flavor int

// Flavor constants. The zero value is intentionally invalid so that
// unparsed settings are caught.
const (
	flavorInvalid Flavor = iota
	FlavorCommonMark
	FlavorGFM
	FlavorGoldmark
)

// String returns the canonical lowercase name of the flavor.
func (f Flavor) String() string {
	switch f {
	case FlavorCommonMark:
		return "commonmark"
	case FlavorGFM:
		return "gfm"
	case FlavorGoldmark:
		return "goldmark"
	}
	return ""
}

// ParseFlavor converts a config string into a Flavor. The match is
// case-sensitive to reject typos like "GFM" that would otherwise
// silently validate against the wrong flavor.
func ParseFlavor(s string) (Flavor, bool) {
	switch s {
	case "commonmark":
		return FlavorCommonMark, true
	case "gfm":
		return FlavorGFM, true
	case "goldmark":
		return FlavorGoldmark, true
	}
	return 0, false
}

// Feature identifies one Markdown syntax feature whose support varies
// across flavors.
type Feature int

// Feature constants. Keep in sync with AllFeatures and featureNames.
const (
	FeatureTables Feature = iota
	FeatureTaskLists
	FeatureStrikethrough
	FeatureBareURLAutolinks
	FeatureFootnotes
	FeatureDefinitionLists
	FeatureHeadingIDs
	FeatureSuperscript
	FeatureSubscript
	FeatureMathBlock
	FeatureMathInline
	FeatureAbbreviations
)

// AllFeatures returns every tracked feature in declaration order.
func AllFeatures() []Feature {
	return []Feature{
		FeatureTables,
		FeatureTaskLists,
		FeatureStrikethrough,
		FeatureBareURLAutolinks,
		FeatureFootnotes,
		FeatureDefinitionLists,
		FeatureHeadingIDs,
		FeatureSuperscript,
		FeatureSubscript,
		FeatureMathBlock,
		FeatureMathInline,
		FeatureAbbreviations,
	}
}

// Name returns the human-readable feature name used in diagnostics.
func (f Feature) Name() string {
	switch f {
	case FeatureTables:
		return "tables"
	case FeatureTaskLists:
		return "task lists"
	case FeatureStrikethrough:
		return "strikethrough"
	case FeatureBareURLAutolinks:
		return "bare-URL autolinks"
	case FeatureFootnotes:
		return "footnotes"
	case FeatureDefinitionLists:
		return "definition lists"
	case FeatureHeadingIDs:
		return "heading IDs"
	case FeatureSuperscript:
		return "superscript"
	case FeatureSubscript:
		return "subscript"
	case FeatureMathBlock:
		return "math blocks"
	case FeatureMathInline:
		return "inline math"
	case FeatureAbbreviations:
		return "abbreviations"
	}
	return ""
}

// support maps (flavor, feature) to whether the flavor accepts it.
// CommonMark rejects every tracked feature. GFM adds tables, task
// lists, strikethrough, and bare-URL autolinks. The goldmark profile
// further adds heading IDs but still rejects the optional extensions
// (footnotes, definition lists, math, sub/sup, abbreviations).
var support = map[Flavor]map[Feature]bool{
	FlavorGFM: {
		FeatureTables:           true,
		FeatureTaskLists:        true,
		FeatureStrikethrough:    true,
		FeatureBareURLAutolinks: true,
	},
	FlavorGoldmark: {
		FeatureTables:           true,
		FeatureTaskLists:        true,
		FeatureStrikethrough:    true,
		FeatureBareURLAutolinks: true,
		FeatureHeadingIDs:       true,
	},
}

// Supports reports whether the flavor accepts the given feature.
func (f Flavor) Supports(feat Feature) bool {
	return support[f][feat]
}
