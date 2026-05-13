// Package markdownflavor implements MDS034, which validates Markdown
// against a declared target flavor (commonmark, gfm, goldmark,
// pandoc, phpextra, multimarkdown, myst, or any) and flags syntax
// the target renderer will not understand. The Flavor type itself,
// along with the convention bundles that select it, lives in
// internal/convention; this package consumes those data shapes.
package markdownflavor

import "github.com/jeduden/mdsmith/internal/convention"

// Feature identifies one Markdown syntax feature whose support varies
// across flavors.
type Feature int

// Feature constants. Keep in sync with AllFeatures and Feature.Name.
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
	FeatureGitHubAlerts
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
		FeatureGitHubAlerts,
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
	case FeatureGitHubAlerts:
		return "github alerts"
	}
	return ""
}

// support maps (flavor, feature) to whether the flavor accepts it.
// CommonMark rejects every tracked feature. GFM adds tables, task
// lists, strikethrough, and bare-URL autolinks. The goldmark flavor
// further adds heading IDs. Pandoc, PHP Markdown Extra, MultiMarkdown,
// and MyST each pick a different combination of the optional
// features; convention.FlavorAny is handled specially in Supports.
var support = map[convention.Flavor]map[Feature]bool{
	convention.FlavorGFM: {
		FeatureTables:           true,
		FeatureTaskLists:        true,
		FeatureStrikethrough:    true,
		FeatureBareURLAutolinks: true,
		FeatureGitHubAlerts:     true,
	},
	convention.FlavorGoldmark: {
		FeatureTables:           true,
		FeatureTaskLists:        true,
		FeatureStrikethrough:    true,
		FeatureBareURLAutolinks: true,
		FeatureHeadingIDs:       true,
	},
	convention.FlavorPandoc: {
		FeatureTables:           true,
		FeatureTaskLists:        true,
		FeatureStrikethrough:    true,
		FeatureBareURLAutolinks: true,
		FeatureFootnotes:        true,
		FeatureDefinitionLists:  true,
		FeatureHeadingIDs:       true,
		FeatureSuperscript:      true,
		FeatureSubscript:        true,
		FeatureMathBlock:        true,
		FeatureMathInline:       true,
	},
	convention.FlavorPHPExtra: {
		FeatureTables:          true,
		FeatureFootnotes:       true,
		FeatureDefinitionLists: true,
		FeatureHeadingIDs:      true,
		FeatureAbbreviations:   true,
	},
	convention.FlavorMultiMarkdown: {
		FeatureTables:          true,
		FeatureFootnotes:       true,
		FeatureDefinitionLists: true,
		FeatureHeadingIDs:      true,
		FeatureAbbreviations:   true,
		FeatureMathBlock:       true,
		FeatureMathInline:      true,
	},
	convention.FlavorMyST: {
		FeatureTables:          true,
		FeatureStrikethrough:   true,
		FeatureFootnotes:       true,
		FeatureDefinitionLists: true,
		FeatureHeadingIDs:      true,
		FeatureMathBlock:       true,
		FeatureMathInline:      true,
	},
}

// Supports reports whether the flavor accepts the given feature.
// convention.FlavorAny accepts every feature; other flavors consult
// the support table.
func Supports(f convention.Flavor, feat Feature) bool {
	if f == convention.FlavorAny {
		return true
	}
	return support[f][feat]
}
