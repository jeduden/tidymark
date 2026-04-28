package markdownflavor

import (
	"fmt"
	"sort"
	"strings"
)

// Profile is an opinionated bundle that pairs a Markdown flavor with a
// table of rule presets. Selecting a profile in config applies both:
// MDS034 runs against the named flavor, and the named rule presets are
// applied as a base layer beneath the user's own rule config.
//
// Profiles are codebase-versioned. The Rules table may reference rules
// that are not yet registered (presets for upcoming MDS04x rules ship
// alongside the profile so that adding the rule does not require
// updating every consumer's config). The config loader treats presets
// for unregistered rules as a no-op at check time; the settings remain
// in the merged config and activate automatically once the rule lands.
type Profile struct {
	// Name is the lowercase identifier used in YAML config.
	Name string
	// Flavor is the Markdown flavor MDS034 should validate against.
	Flavor Flavor
	// Rules maps rule name (e.g. "no-inline-html") to the preset that
	// the profile applies for that rule.
	Rules map[string]RulePreset
}

// RulePreset is a profile's preset for a single rule. It mirrors the
// shape of config.RuleCfg without depending on the config package, so
// the markdownflavor package can declare profile tables without the
// import cycle that would otherwise result.
type RulePreset struct {
	Enabled  bool
	Settings map[string]any
}

// profiles is the built-in profile table. Each entry pairs a target
// flavor with rule-by-rule presets. New profiles are added here; the
// table is consulted via Lookup.
var profiles = map[string]Profile{
	"portable": {
		Name:   "portable",
		Flavor: FlavorCommonMark,
		Rules: map[string]RulePreset{
			"markdown-flavor": {
				Enabled:  true,
				Settings: map[string]any{"flavor": "commonmark"},
			},
			"no-inline-html": {Enabled: true},
			"no-reference-style": {
				Enabled:  true,
				Settings: map[string]any{"allow-footnotes": false},
			},
			"emphasis-style": {
				Enabled: true,
				Settings: map[string]any{
					"bold":   "asterisk",
					"italic": "underscore",
				},
			},
			"horizontal-rule-style": {
				Enabled: true,
				Settings: map[string]any{
					"style":               "dash",
					"length":              3,
					"require-blank-lines": true,
				},
			},
			"list-marker-style": {
				Enabled:  true,
				Settings: map[string]any{"style": "dash"},
			},
			"ordered-list-numbering": {
				Enabled: true,
				Settings: map[string]any{
					"style": "sequential",
					"start": 1,
				},
			},
			"ambiguous-emphasis": {
				Enabled:  true,
				Settings: map[string]any{"max-run": 2},
			},
		},
	},
	"github": {
		Name:   "github",
		Flavor: FlavorGFM,
		Rules: map[string]RulePreset{
			"markdown-flavor": {
				Enabled:  true,
				Settings: map[string]any{"flavor": "gfm"},
			},
			"no-inline-html": {
				Enabled:  true,
				Settings: map[string]any{"allow": []any{"details", "summary"}},
			},
			"emphasis-style": {
				Enabled: true,
				Settings: map[string]any{
					"bold":   "asterisk",
					"italic": "underscore",
				},
			},
			"list-marker-style": {
				Enabled:  true,
				Settings: map[string]any{"style": "dash"},
			},
		},
	},
	"plain": {
		Name:   "plain",
		Flavor: FlavorCommonMark,
		Rules: map[string]RulePreset{
			"markdown-flavor": {
				Enabled:  true,
				Settings: map[string]any{"flavor": "commonmark"},
			},
			"no-inline-html": {
				Enabled:  true,
				Settings: map[string]any{"allow-comments": false},
			},
			"no-reference-style": {
				Enabled:  true,
				Settings: map[string]any{"allow-footnotes": false},
			},
			"emphasis-style": {
				Enabled: true,
				Settings: map[string]any{
					"bold":   "asterisk",
					"italic": "underscore",
				},
			},
			"horizontal-rule-style": {
				Enabled: true,
				Settings: map[string]any{
					"style":               "dash",
					"length":              3,
					"require-blank-lines": true,
				},
			},
			"list-marker-style": {
				Enabled:  true,
				Settings: map[string]any{"style": "dash"},
			},
			"ordered-list-numbering": {
				Enabled: true,
				Settings: map[string]any{
					"style": "sequential",
					"start": 1,
				},
			},
			"ambiguous-emphasis": {
				Enabled:  true,
				Settings: map[string]any{"max-run": 2},
			},
		},
	},
}

// Lookup returns the profile table entry for name. It returns an error
// naming the field and listing valid names when name is not a known
// profile, matching the failure-mode contract in plan 112.
func Lookup(name string) (Profile, error) {
	p, ok := profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf(
			"unknown profile %q (valid: %s)", name, strings.Join(ProfileNames(), ", "),
		)
	}
	return p, nil
}

// ProfileNames returns the sorted list of built-in profile names.
func ProfileNames() []string {
	names := make([]string, 0, len(profiles))
	for k := range profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
