package config

import "fmt"

// EffectiveKinds returns the ordered list of kind names that apply to the
// file at filePath. The list is built by concatenating, in order:
//
//  1. frontMatterKinds — the value of the file's front-matter `kinds:`
//     field (or nil if absent).
//  2. The kinds from each entry in cfg.KindAssignment whose files glob
//     matches filePath; entries are visited in config order, kinds within
//     an entry are taken in the order listed.
//
// Duplicate names are dropped after their first occurrence (so the first
// position wins for ordering). Referencing a kind name that is not declared
// in cfg.Kinds is a config error and produces an error mentioning the
// offending name.
func EffectiveKinds(cfg *Config, filePath string, frontMatterKinds []string) ([]string, error) {
	if cfg == nil {
		return nil, nil
	}

	var ordered []string
	seen := make(map[string]bool)

	add := func(name, source string) error {
		if _, declared := cfg.Kinds[name]; !declared {
			return fmt.Errorf("undeclared kind %q referenced by %s", name, source)
		}
		if seen[name] {
			return nil
		}
		seen[name] = true
		ordered = append(ordered, name)
		return nil
	}

	for _, k := range frontMatterKinds {
		if err := add(k, "front matter"); err != nil {
			return nil, err
		}
	}

	for _, ka := range cfg.KindAssignment {
		if !matchesAny(ka.Files, filePath) {
			continue
		}
		for _, k := range ka.Kinds {
			if err := add(k, "kind-assignment"); err != nil {
				return nil, err
			}
		}
	}

	return ordered, nil
}

// EffectiveWithKinds returns the effective rule configuration for filePath
// after applying, in order:
//
//  1. Top-level cfg.Rules.
//  2. Each kind in the file's effective kind list (block-replace per rule).
//  3. Each cfg.Overrides entry whose files glob matches filePath
//     (block-replace per rule).
//
// Multiple kinds configuring the same rule resolve by block replacement —
// the kind appearing later in the effective list replaces the earlier
// kind's entire rule config block. This mirrors the existing override
// merge semantics so both layers reuse the same rule.
func EffectiveWithKinds(
	cfg *Config,
	filePath string,
	frontMatterKinds []string,
) (map[string]RuleCfg, error) {
	result := make(map[string]RuleCfg, len(cfg.Rules))
	for k, v := range cfg.Rules {
		result[k] = v
	}

	kinds, err := EffectiveKinds(cfg, filePath, frontMatterKinds)
	if err != nil {
		return nil, err
	}
	for _, name := range kinds {
		k := cfg.Kinds[name]
		for rk, rv := range k.Rules {
			result[rk] = rv
		}
	}

	for _, o := range cfg.Overrides {
		if matchesAny(o.Files, filePath) {
			for k, v := range o.Rules {
				result[k] = v
			}
		}
	}

	return result, nil
}

// EffectiveCategoriesWithKinds returns the effective category settings for
// filePath after applying, in order: top-level cfg.Categories, the
// categories of each kind in the effective kind list, and the categories
// of each matching override.
func EffectiveCategoriesWithKinds(
	cfg *Config,
	filePath string,
	frontMatterKinds []string,
) (map[string]bool, error) {
	// Start with all categories enabled.
	result := make(map[string]bool, len(ValidCategories))
	for _, cat := range ValidCategories {
		result[cat] = true
	}

	// Apply top-level category settings.
	for k, v := range cfg.Categories {
		result[k] = v
	}

	kinds, err := EffectiveKinds(cfg, filePath, frontMatterKinds)
	if err != nil {
		return nil, err
	}
	for _, name := range kinds {
		k := cfg.Kinds[name]
		for ck, cv := range k.Categories {
			result[ck] = cv
		}
	}

	for _, o := range cfg.Overrides {
		if matchesAny(o.Files, filePath) {
			for k, v := range o.Categories {
				result[k] = v
			}
		}
	}

	return result, nil
}

// EffectiveExplicitRulesWithKinds returns the set of rule names that were
// explicitly configured for the given file path. Rules set in any kind
// applied to the file are considered explicit, as are rules set in any
// matching override and rules in cfg.ExplicitRules.
func EffectiveExplicitRulesWithKinds(
	cfg *Config,
	filePath string,
	frontMatterKinds []string,
) (map[string]bool, error) {
	result := make(map[string]bool, len(cfg.ExplicitRules))
	for k := range cfg.ExplicitRules {
		result[k] = true
	}

	kinds, err := EffectiveKinds(cfg, filePath, frontMatterKinds)
	if err != nil {
		return nil, err
	}
	for _, name := range kinds {
		k := cfg.Kinds[name]
		for rk := range k.Rules {
			result[rk] = true
		}
	}

	for _, o := range cfg.Overrides {
		if matchesAny(o.Files, filePath) {
			for k := range o.Rules {
				result[k] = true
			}
		}
	}

	return result, nil
}
