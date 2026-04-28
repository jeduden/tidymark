package config

import (
	"fmt"

	"github.com/jeduden/mdsmith/internal/rules/markdownflavor"
)

// applyProfile reads the markdown-flavor profile from the loaded
// config (if any) and stores its rule presets on cfg.ProfilePreset.
// The preset is applied as a base layer beneath the user's own rule
// config during effective-rule resolution; cfg.Rules is left
// untouched here so per-file provenance (`mdsmith kinds resolve`)
// can show the profile as its own layer rather than collapsing it
// into the default layer.
//
// Validation:
//
//   - Unknown profile name → error naming the field and listing valid
//     names.
//   - Profile and flavor disagree → error naming both values. A
//     profile sets a flavor; a user-supplied flavor that does not
//     match is rejected at config load so the error surfaces once,
//     not on every check.
func applyProfile(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	rc, ok := cfg.Rules["markdown-flavor"]
	if !ok {
		return nil
	}
	profileName, _ := rc.Settings["profile"].(string)
	if profileName == "" {
		return nil
	}
	profile, err := markdownflavor.Lookup(profileName)
	if err != nil {
		return fmt.Errorf("rules.markdown-flavor.profile: %w", err)
	}
	if userFlavor, ok := rc.Settings["flavor"].(string); ok && userFlavor != "" {
		if userFlavor != profile.Flavor.String() {
			return fmt.Errorf(
				"rules.markdown-flavor: profile %q requires flavor %q, but flavor is set to %q",
				profile.Name, profile.Flavor, userFlavor,
			)
		}
	}

	preset := make(map[string]RuleCfg, len(profile.Rules))
	for ruleName, p := range profile.Rules {
		preset[ruleName] = RuleCfg{
			Enabled:  p.Enabled,
			Settings: cloneSettings(p.Settings),
		}
	}
	cfg.Profile = profile.Name
	cfg.ProfilePreset = preset
	return nil
}

// copyProfilePreset returns a deep copy of a profile preset map. Each
// RuleCfg's settings map is cloned so callers can mutate the result
// without affecting the source.
func copyProfilePreset(p map[string]RuleCfg) map[string]RuleCfg {
	if p == nil {
		return nil
	}
	out := make(map[string]RuleCfg, len(p))
	for k, v := range p {
		out[k] = copyRuleCfg(v)
	}
	return out
}
