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
// users cannot redefine. Defining any of these in the conventions: block
// is a config error.
var reservedConventionNames = map[string]bool{
	"portable": true,
	"github":   true,
	"plain":    true,
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
//   - Reserved convention names (portable, github, plain) in the
//     user-defined conventions map → config error.
//   - Unknown convention name → error naming the field and listing
//     valid names from both built-in and user-defined sets.
//   - Each rule name under a user-defined convention must be a
//     registered rule.
//   - Each rule's settings must pass ApplySettings validation.
//   - Convention and a user-supplied rules.markdown-flavor.flavor
//     disagree → error naming both values. A convention sets a
//     flavor; a user-supplied flavor that does not match is
//     rejected at config load so the error surfaces once, not on
//     every check.
func applyConvention(cfg *Config) error {
	if cfg == nil {
		return nil
	}

	userMap, err := validateAndBuildUserConventions(cfg.Conventions)
	if err != nil {
		return err
	}

	if cfg.Convention == "" {
		return nil
	}

	// Determine whether the selected convention is user-defined.
	_, isUser := userMap[cfg.Convention]
	cfg.ConventionIsUser = isUser

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
	return nil
}

// validateAndBuildUserConventions validates the user-defined conventions
// map and converts it to the markdownflavor.Convention type. Errors are
// returned for:
//   - Reserved names (portable, github, plain).
//   - Invalid flavor strings.
//   - Unknown rule names.
//   - Invalid rule settings (via each rule's ApplySettings).
func validateAndBuildUserConventions(
	bodies map[string]ConventionBody,
) (map[string]markdownflavor.Convention, error) {
	if len(bodies) == 0 {
		return nil, nil
	}

	// Sort names for deterministic error messages.
	names := make([]string, 0, len(bodies))
	for n := range bodies {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make(map[string]markdownflavor.Convention, len(bodies))
	for _, name := range names {
		c, err := buildUserConvention(name, bodies[name])
		if err != nil {
			return nil, err
		}
		out[name] = c
	}
	return out, nil
}

// buildUserConvention validates and converts one user-defined convention
// body into a markdownflavor.Convention value.
func buildUserConvention(name string, body ConventionBody) (markdownflavor.Convention, error) {
	if reservedConventionNames[name] {
		return markdownflavor.Convention{}, fmt.Errorf(
			"conventions.%s: %q is a reserved built-in convention name",
			name, name,
		)
	}

	flavor, err := parseUserConventionFlavor(name, body.Flavor)
	if err != nil {
		return markdownflavor.Convention{}, err
	}

	rules, err := buildUserConventionRules(name, body.Rules)
	if err != nil {
		return markdownflavor.Convention{}, err
	}

	return markdownflavor.Convention{
		Name:   name,
		Flavor: flavor,
		Rules:  rules,
	}, nil
}

// parseUserConventionFlavor validates and returns the flavor for a
// user-defined convention. An empty flavor string is accepted (returns
// the zero Flavor value).
func parseUserConventionFlavor(convName, flavorStr string) (markdownflavor.Flavor, error) {
	if flavorStr == "" {
		return 0, nil
	}
	f, ok := markdownflavor.ParseFlavor(flavorStr)
	if !ok {
		const validFlavors = "commonmark, gfm, goldmark, any, pandoc, phpextra, multimarkdown, myst"
		return 0, fmt.Errorf(
			"conventions.%s: unknown flavor %q (valid: %s)",
			convName, flavorStr, validFlavors,
		)
	}
	return f, nil
}

// buildUserConventionRules validates and converts the rules map from a
// user-defined convention body. Each rule name must be registered and
// each rule's settings must pass ApplySettings validation.
func buildUserConventionRules(
	convName string,
	cfgRules map[string]RuleCfg,
) (map[string]markdownflavor.RulePreset, error) {
	rules := make(map[string]markdownflavor.RulePreset, len(cfgRules))
	for ruleName, rc := range cfgRules {
		r := rule.ByName(ruleName)
		if r == nil {
			return nil, fmt.Errorf(
				"convention %q rule %q: unknown rule name",
				convName, ruleName,
			)
		}
		if rc.Settings != nil {
			if err := validateUserConventionRuleSettings(convName, ruleName, r, rc.Settings); err != nil {
				return nil, err
			}
		}
		rules[ruleName] = markdownflavor.RulePreset{
			Enabled:  rc.Enabled,
			Settings: rc.Settings,
		}
	}
	return rules, nil
}

// validateUserConventionRuleSettings validates a rule's settings by
// applying them to a fresh clone of the rule. This avoids mutating the
// shared registry entry.
func validateUserConventionRuleSettings(
	convName, ruleName string,
	r rule.Rule,
	settings map[string]any,
) error {
	if _, ok := r.(rule.Configurable); !ok {
		return nil
	}
	// Validate settings by applying them to a fresh clone.
	// CloneRule returns a zero-value instance so we do not
	// mutate the shared registry entry.
	cloned := rule.CloneRule(r)
	cr, ok := cloned.(rule.Configurable)
	if !ok {
		return nil
	}
	if err := cr.ApplySettings(settings); err != nil {
		return fmt.Errorf("convention %q rule %q: %w", convName, ruleName, err)
	}
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
