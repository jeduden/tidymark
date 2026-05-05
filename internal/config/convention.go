package config

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/jeduden/mdsmith/internal/rules/markdownflavor"
)

// reservedConventionNames is the set of built-in convention names that
// users may not redefine. Protecting these names keeps docs and
// tutorials consistent — a team that defines `conventions.portable`
// would shadow the built-in meaning of that name for everyone reading
// their config.
var reservedConventionNames = map[string]bool{
	"portable": true,
	"github":   true,
	"plain":    true,
}

// validateUserConventions checks every entry in cfg.Conventions:
//
//  1. The name must not collide with the three reserved built-in names.
//  2. The flavor string must be one of the values ParseFlavor accepts.
//  3. Each key in the rules map must name a registered rule.
//  4. Each rule's settings block must pass ApplySettings validation.
//
// Errors name the offending convention and, where applicable, the
// offending rule.
func validateUserConventions(cfg *Config) error {
	if len(cfg.Conventions) == 0 {
		return nil
	}

	// Iterate in sorted order so errors are deterministic.
	names := make([]string, 0, len(cfg.Conventions))
	for name := range cfg.Conventions {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		body := cfg.Conventions[name]

		if reservedConventionNames[name] {
			return fmt.Errorf(
				"conventions.%s: %q is a reserved built-in convention name",
				name, name,
			)
		}

		if body.Flavor != "" {
			if _, ok := markdownflavor.ParseFlavor(body.Flavor); !ok {
				return fmt.Errorf(
					"conventions.%s: unknown flavor %q",
					name, body.Flavor,
				)
			}
		}

		for ruleName, rc := range body.Rules {
			r := rule.ByName(ruleName)
			if r == nil {
				return fmt.Errorf(
					"conventions.%s: unknown rule %q",
					name, ruleName,
				)
			}
			if rc.Settings != nil {
				c, ok := r.(rule.Configurable)
				if !ok {
					return fmt.Errorf(
						"conventions.%s rule %q: rule has no configurable settings",
						name, ruleName,
					)
				}
				// Clone so validation does not mutate the stored settings.
				if err := c.ApplySettings(cloneSettings(rc.Settings)); err != nil {
					return fmt.Errorf(
						"conventions.%s rule %q: %w",
						name, ruleName, err,
					)
				}
			}
		}
	}
	return nil
}

// userConventionsAsMap converts cfg.Conventions into the
// markdownflavor.Convention map expected by Lookup. Each user
// convention body is translated into a Convention value using the
// stored flavor string and rule presets.
func userConventionsAsMap(cfg *Config) map[string]markdownflavor.Convention {
	if len(cfg.Conventions) == 0 {
		return nil
	}
	out := make(map[string]markdownflavor.Convention, len(cfg.Conventions))
	for name, body := range cfg.Conventions {
		flavor, _ := markdownflavor.ParseFlavor(body.Flavor)
		rules := make(map[string]markdownflavor.RulePreset, len(body.Rules))
		for ruleName, rc := range body.Rules {
			rules[ruleName] = markdownflavor.RulePreset{
				Enabled:  rc.Enabled,
				Settings: cloneSettings(rc.Settings),
			}
		}
		out[name] = markdownflavor.Convention{
			Name:   name,
			Flavor: flavor,
			Rules:  rules,
		}
	}
	return out
}

// isUserConvention reports whether name is defined in cfg.Conventions
// (as opposed to being a built-in convention).
func isUserConvention(cfg *Config, name string) bool {
	_, ok := cfg.Conventions[name]
	return ok
}

// applyConvention reads the top-level Convention selector from the
// loaded config (if any) and stores its rule presets on
// cfg.ConventionPreset. The preset is applied as a base layer
// beneath the user's own rule config during effective-rule
// resolution; cfg.Rules is left untouched here so per-file
// provenance (`mdsmith kinds resolve`) can show the convention as
// its own layer rather than collapsing it into the default layer.
//
// Resolution order: user-defined conventions (cfg.Conventions) are
// consulted first, then the built-in table. Collisions between user
// names and built-in names are rejected by validateUserConventions
// before this function runs, so the lookup order is unambiguous.
//
// Validation:
//
//   - Unknown convention name → error naming the field and listing
//     valid names (both user-defined and built-in).
//   - Convention and a user-supplied rules.markdown-flavor.flavor
//     disagree → error naming both values. A convention sets a
//     flavor; a user-supplied flavor that does not match is
//     rejected at config load so the error surfaces once, not on
//     every check.
func applyConvention(cfg *Config) error {
	if cfg == nil || cfg.Convention == "" {
		return nil
	}
	userMap := userConventionsAsMap(cfg)
	convention, err := markdownflavor.Lookup(cfg.Convention, userMap)
	if err != nil {
		return fmt.Errorf("convention: %w", err)
	}
	if rc, ok := cfg.Rules["markdown-flavor"]; ok {
		userFlavor, err := stringSetting(
			rc.Settings, "flavor", "rules.markdown-flavor.flavor",
		)
		if err != nil {
			return err
		}
		if userFlavor != "" && userFlavor != convention.Flavor.String() {
			return fmt.Errorf(
				"rules.markdown-flavor: convention %q requires flavor %q, but flavor is set to %q",
				convention.Name, convention.Flavor, userFlavor,
			)
		}
	}

	preset := make(map[string]RuleCfg, len(convention.Rules))
	for ruleName, p := range convention.Rules {
		preset[ruleName] = RuleCfg{
			Enabled:  p.Enabled,
			Settings: cloneSettings(p.Settings),
		}
	}
	cfg.ConventionPreset = preset
	// Record whether this is a user convention so provenance labels
	// can distinguish "(user)" from built-in.
	cfg.conventionIsUser = isUserConvention(cfg, cfg.Convention)
	return nil
}

// stringSetting reads a string-typed setting from a settings map. A
// missing key returns "" with no error; a present key with a
// non-string value returns an error naming the offending field path
// so users see the problem at config load time.
func stringSetting(settings map[string]any, key, fieldPath string) (string, error) {
	v, ok := settings[key]
	if !ok {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s: must be a string, got %T", fieldPath, v)
	}
	return s, nil
}

// copyConventionPreset returns a deep copy of a convention preset
// map. Each RuleCfg's settings map is cloned so callers can mutate
// the result without affecting the source.
func copyConventionPreset(p map[string]RuleCfg) map[string]RuleCfg {
	if p == nil {
		return nil
	}
	out := make(map[string]RuleCfg, len(p))
	for k, v := range p {
		out[k] = copyRuleCfg(v)
	}
	return out
}

// validateConventionScalar returns an error when the top-level
// `convention:` value in the raw YAML is not a string scalar.
// yaml.v3 silently coerces bare ints and bools into string fields,
// which would surface as "unknown convention 123" instead of a
// clean type error. Inspecting the raw node tag is the only way to
// catch the type mismatch before that coercion happens.
func validateConventionScalar(data []byte) error {
	// yaml.Unmarshal into yaml.Node does not expand aliases, so this
	// is safe without an alias-rejection pre-check. Errors are swallowed
	// because Load's subsequent UnmarshalSafe call will surface them.
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil
	}
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return nil
	}
	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value != "convention" {
			continue
		}
		v := mapping.Content[i+1]
		if v.Kind != yaml.ScalarNode {
			return fmt.Errorf("convention: must be a string scalar")
		}
		if v.Tag != "" && v.Tag != "!!str" {
			return fmt.Errorf(
				"convention: must be a string, got %s",
				strings.TrimPrefix(v.Tag, "!!"),
			)
		}
		return nil
	}
	return nil
}
