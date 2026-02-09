package config

import (
	"github.com/gobwas/glob"
)

// Merge merges a loaded config on top of defaults. The loaded config's rules
// override the defaults; any rule not mentioned in loaded keeps its default
// value. Ignore and Overrides come from the loaded config only.
func Merge(defaults, loaded *Config) *Config {
	if loaded == nil {
		// No user config: return a copy of defaults.
		rules := make(map[string]RuleCfg, len(defaults.Rules))
		for k, v := range defaults.Rules {
			rules[k] = v
		}
		return &Config{Rules: rules, FrontMatter: defaults.FrontMatter}
	}

	rules := make(map[string]RuleCfg, len(defaults.Rules))
	for k, v := range defaults.Rules {
		rules[k] = v
	}

	// Apply loaded rules on top.
	for k, v := range loaded.Rules {
		rules[k] = v
	}

	fm := defaults.FrontMatter
	if loaded.FrontMatter != nil {
		fm = loaded.FrontMatter
	}

	return &Config{
		Rules:       rules,
		Ignore:      loaded.Ignore,
		Overrides:   loaded.Overrides,
		FrontMatter: fm,
	}
}

// Effective returns the effective rule configuration for a given file path.
// It starts with the top-level rules and then applies each override whose
// file patterns match filePath, in order. Later overrides take precedence.
func Effective(cfg *Config, filePath string) map[string]RuleCfg {
	result := make(map[string]RuleCfg, len(cfg.Rules))
	for k, v := range cfg.Rules {
		result[k] = v
	}

	for _, o := range cfg.Overrides {
		if matchesAny(o.Files, filePath) {
			for k, v := range o.Rules {
				result[k] = v
			}
		}
	}

	return result
}

// matchesAny returns true if filePath matches any of the given glob patterns.
func matchesAny(patterns []string, filePath string) bool {
	for _, pattern := range patterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			// Skip invalid patterns silently.
			continue
		}
		if g.Match(filePath) {
			return true
		}
	}
	return false
}
