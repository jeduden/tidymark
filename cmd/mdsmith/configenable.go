package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// enableGitHookSyncRule ensures the git-hook-sync rule is enabled in .mdsmith.yml.
// If the config file doesn't exist, it creates one with the rule enabled.
// If it exists but doesn't have the rule enabled, it adds it.
// Returns an error if the config file cannot be read or written.
func enableGitHookSyncRule(repoRoot string) error {
	configPath := filepath.Join(repoRoot, ".mdsmith.yml")

	// Read existing config or create empty one
	var config map[string]any
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading config: %w", err)
	}

	if err == nil {
		// Parse existing config
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing config: %w", err)
		}
	}

	if config == nil {
		config = make(map[string]any)
	}

	// Ensure rules section exists
	rules, ok := config["rules"].(map[string]any)
	if !ok {
		rules = make(map[string]any)
		config["rules"] = rules
	}

	// Check if git-hook-sync is already enabled
	if val, ok := rules["git-hook-sync"]; ok {
		// If it's already set to something other than false, leave it alone
		if val != false {
			return nil
		}
	}

	// Enable the rule
	rules["git-hook-sync"] = true

	// Write back to file
	newData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
