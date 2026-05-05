package config

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/jeduden/mdsmith/internal/rules/markdownflavor"
)

// validConventionFlavors lists the flavor names accepted in a
// user-defined convention. The full flavor set accepted by MDS034
// includes renderer-specific dialects that are not suitable as
// convention bases.
var validConventionFlavors = map[string]bool{
	"commonmark": true,
	"gfm":        true,
	"goldmark":   true,
}

// applyConvention reads the top-level Convention selector from the
// loaded config (if any) and stores its rule presets on
// cfg.ConventionPreset. The preset is applied as a base layer
// beneath the user's own rule config during effective-rule
// resolution; cfg.Rules is left untouched here so per-file
// provenance (`mdsmith kinds resolve`) can show the convention as
// its own layer rather than collapsing it into the default layer.
//
// Validation:
//
//   - Unknown convention name → error naming the field and listing
//     valid names.
//   - Convention and a user-supplied rules.markdown-flavor.flavor
//     disagree → error naming both values. A convention sets a
//     flavor; a user-supplied flavor that does not match is
//     rejected at config load so the error surfaces once, not on
//     every check.
func applyConvention(cfg *Config) error {
	if cfg == nil || cfg.Convention == "" {
		return nil
	}
	userMap, err := buildUserConventionMap(cfg)
	if err != nil {
		return err
	}
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

	// Mark whether the active convention is user-defined so provenance
	// can append a "(user)" label to the convention layer source.
	if _, isUser := userMap[cfg.Convention]; isUser {
		cfg.ConventionIsUser = true
	}
	return nil
}

// buildUserConventionMap validates the user-defined conventions block
// and returns a markdownflavor.Convention map for Lookup. Each entry
// is validated:
//   - name must not collide with the reserved built-in names
//   - flavor must be one of commonmark, gfm, goldmark
//   - each rule name must be registered
//   - each rule's settings must pass ApplySettings validation
//
// The returned map is nil when cfg.Conventions is empty.
func buildUserConventionMap(cfg *Config) (map[string]markdownflavor.Convention, error) {
	if len(cfg.Conventions) == 0 {
		return nil, nil
	}
	reserved := markdownflavor.ReservedNames()
	out := make(map[string]markdownflavor.Convention, len(cfg.Conventions))
	names := sortedKeys(cfg.Conventions)
	for _, name := range names {
		entry := cfg.Conventions[name]
		if reserved[name] {
			return nil, fmt.Errorf(
				"conventions.%s: name %q is reserved for the built-in convention",
				name, name,
			)
		}
		c, err := validateConventionEntry(name, entry)
		if err != nil {
			return nil, err
		}
		out[name] = c
	}
	return out, nil
}

// sortedKeys returns the keys of m in sorted order.
func sortedKeys(m map[string]ConventionEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// validateConventionEntry validates a single user-defined convention entry
// and returns a markdownflavor.Convention ready for the Lookup table.
func validateConventionEntry(name string, entry ConventionEntry) (markdownflavor.Convention, error) {
	if !validConventionFlavors[entry.Flavor] {
		valid := []string{"commonmark", "gfm", "goldmark"}
		return markdownflavor.Convention{}, fmt.Errorf(
			"conventions.%s: flavor %q is not valid for a user convention (valid: %s)",
			name, entry.Flavor, strings.Join(valid, ", "),
		)
	}
	fl, _ := markdownflavor.ParseFlavor(entry.Flavor)
	rules, err := validateConventionRules(name, entry.Rules)
	if err != nil {
		return markdownflavor.Convention{}, err
	}
	return markdownflavor.Convention{Name: name, Flavor: fl, Rules: rules}, nil
}

// validateConventionRules validates the rules block of a user-defined
// convention entry. Returns the converted preset map or an error.
func validateConventionRules(
	conventionName string, cfgRules map[string]RuleCfg,
) (map[string]markdownflavor.RulePreset, error) {
	rules := make(map[string]markdownflavor.RulePreset, len(cfgRules))
	for ruleName, rc := range cfgRules {
		r := rule.ByName(ruleName)
		if r == nil {
			return nil, fmt.Errorf(
				"conventions.%s: rule %q is not registered",
				conventionName, ruleName,
			)
		}
		if rc.Settings != nil {
			if c, ok := r.(rule.Configurable); ok {
				clone := c
				if err := clone.ApplySettings(rc.Settings); err != nil {
					return nil, fmt.Errorf(
						"conventions.%s rule %q: %w",
						conventionName, ruleName, err,
					)
				}
			}
		}
		rules[ruleName] = markdownflavor.RulePreset{
			Enabled:  rc.Enabled,
			Settings: cloneSettings(rc.Settings),
		}
	}
	return rules, nil
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
