package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/jeduden/mdsmith/internal/rules/markdownflavor"
)

// reservedConventionNames is the set of built-in convention names that
// cannot be used as user-defined convention names. Rejecting them at
// config load time preserves the meaning of docs and tutorials that
// reference them by name.
var reservedConventionNames = map[string]bool{
	"portable": true,
	"github":   true,
	"plain":    true,
}

// buildUserConventions validates the cfg.Conventions map and converts it to
// the markdownflavor.Convention map that Lookup expects. Validation checks:
//
//   - Reserved names ("portable", "github", "plain") are rejected.
//   - Each convention's flavor must be a recognized Markdown flavor.
//   - Each rule name under the convention's rules: block must be registered.
//   - Each rule's settings must pass the rule's ApplySettings validation.
//
// The returned map is nil when cfg.Conventions is empty.
func buildUserConventions(cfg *Config) (map[string]markdownflavor.Convention, error) {
	if len(cfg.Conventions) == 0 {
		return nil, nil
	}
	out := make(map[string]markdownflavor.Convention, len(cfg.Conventions))
	for name, body := range cfg.Conventions {
		if reservedConventionNames[name] {
			return nil, fmt.Errorf(
				"conventions.%s: %q is a reserved built-in convention name", name, name,
			)
		}
		flavor, ok := markdownflavor.ParseFlavor(body.Flavor)
		if !ok {
			return nil, fmt.Errorf(
				"conventions.%s.flavor: unknown flavor %q (valid: commonmark, gfm, goldmark)",
				name, body.Flavor,
			)
		}
		rulePresets, err := validateConventionRules(name, body.Rules)
		if err != nil {
			return nil, err
		}
		out[name] = markdownflavor.Convention{
			Name:   name,
			Flavor: flavor,
			Rules:  rulePresets,
		}
	}
	return out, nil
}

// validateConventionRules validates each rule name and its settings inside a
// user-defined convention block. It returns an error naming the convention
// and the rule when either the rule name is unknown or its settings are
// invalid.
func validateConventionRules(
	conventionName string, rules map[string]RuleCfg,
) (map[string]markdownflavor.RulePreset, error) {
	if len(rules) == 0 {
		return nil, nil
	}
	out := make(map[string]markdownflavor.RulePreset, len(rules))
	for ruleName, rc := range rules {
		r := rule.ByName(ruleName)
		if r == nil {
			return nil, fmt.Errorf(
				"convention %q rule %q: unknown rule name", conventionName, ruleName,
			)
		}
		if len(rc.Settings) > 0 {
			if c, ok := r.(rule.Configurable); ok {
				clone, err := ruleCloneWithDefaults(c)
				if err != nil {
					return nil, fmt.Errorf(
						"convention %q rule %q: cloning defaults: %w",
						conventionName, ruleName, err,
					)
				}
				if err := clone.ApplySettings(rc.Settings); err != nil {
					return nil, fmt.Errorf(
						"convention %q rule %q: %w", conventionName, ruleName, err,
					)
				}
			}
		}
		out[ruleName] = markdownflavor.RulePreset{
			Enabled:  rc.Enabled,
			Settings: cloneSettings(rc.Settings),
		}
	}
	return out, nil
}

// ruleCloneWithDefaults creates a fresh instance of a Configurable rule
// with its default settings applied, used as the validation target for
// ApplySettings. The clone is produced via rule.CloneRule so the original
// registered singleton is not mutated.
func ruleCloneWithDefaults(c rule.Configurable) (rule.Configurable, error) {
	cloned := rule.CloneRule(c.(rule.Rule))
	cc, ok := cloned.(rule.Configurable)
	if !ok {
		// The rule is not configurable after cloning; no settings to validate.
		return c, nil
	}
	return cc, nil
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
	if cfg == nil {
		return nil
	}
	userConventions, err := buildUserConventions(cfg)
	if err != nil {
		return err
	}
	if cfg.Convention == "" {
		return nil
	}
	convention, err := markdownflavor.Lookup(cfg.Convention, userConventions)
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
