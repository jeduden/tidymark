package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration.
type Config struct {
	Rules       map[string]RuleCfg `yaml:"rules"`
	Ignore      []string           `yaml:"ignore"`
	Overrides   []Override         `yaml:"overrides"`
	FrontMatter *bool              `yaml:"front-matter"`
}

// Override applies rule settings to files matching glob patterns.
type Override struct {
	Files []string           `yaml:"files"`
	Rules map[string]RuleCfg `yaml:"rules"`
}

// RuleCfg is a YAML union: can be bool (enable/disable) or map[string]any (settings).
type RuleCfg struct {
	Enabled  bool
	Settings map[string]any
}

// UnmarshalYAML implements custom YAML unmarshalling for RuleCfg.
// It handles three forms:
//   - false -> Enabled=false, Settings=nil
//   - true  -> Enabled=true,  Settings=nil
//   - {key: val, ...} -> Enabled=true, Settings={key: val, ...}
func (r *RuleCfg) UnmarshalYAML(value *yaml.Node) error {
	// Try bool first
	if value.Kind == yaml.ScalarNode {
		var b bool
		if err := value.Decode(&b); err == nil {
			r.Enabled = b
			r.Settings = nil
			return nil
		}
	}

	// Try map
	if value.Kind == yaml.MappingNode {
		var m map[string]any
		if err := value.Decode(&m); err != nil {
			return fmt.Errorf("invalid rule config: %w", err)
		}
		r.Enabled = true
		r.Settings = m
		return nil
	}

	return fmt.Errorf("rule config must be a bool or a mapping, got %v", value.Kind)
}
