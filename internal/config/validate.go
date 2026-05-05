package config

import "fmt"

// validateUserConventions validates the user-defined conventions: block.
// It delegates to buildUserConventionMap, which enforces reserved-name
// checks, flavor restrictions, and rule-name and settings validation.
// This is called at Load time even when no convention: selector is set,
// so that a malformed conventions: block surfaces immediately.
func validateUserConventions(cfg *Config) error {
	if len(cfg.Conventions) == 0 {
		return nil
	}
	_, err := buildUserConventionMap(cfg)
	return err
}

// ValidateKinds returns an error if any kind named in a kind-assignment
// entry is not declared in cfg.Kinds. Front-matter kinds are validated
// at lint time via ValidateFrontMatterKinds (see engine).
func ValidateKinds(cfg *Config) error {
	if len(cfg.Kinds) == 0 && len(cfg.KindAssignment) == 0 {
		return nil
	}
	for i, entry := range cfg.KindAssignment {
		for _, name := range entry.Kinds {
			if _, ok := cfg.Kinds[name]; !ok {
				return fmt.Errorf(
					"kind-assignment[%d]: references undeclared kind %q", i, name,
				)
			}
		}
	}
	return nil
}

// ValidateFrontMatterKinds returns an error if any of the supplied front-matter
// kind names is not declared in cfg.Kinds. filePath is used in the message.
func ValidateFrontMatterKinds(cfg *Config, filePath string, kinds []string) error {
	for _, name := range kinds {
		if _, ok := cfg.Kinds[name]; !ok {
			return fmt.Errorf(
				"%s: front matter references undeclared kind %q", filePath, name,
			)
		}
	}
	return nil
}
