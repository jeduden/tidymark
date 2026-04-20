// Package archetypes ships built-in required-structure schemas
// for common agentic Markdown document types.
package archetypes

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed *.md
var files embed.FS

// Lookup returns the bytes of the built-in archetype schema with the
// given name (for example "story-file"). The name is the basename
// without extension. An unknown name returns an error whose message
// lists the available archetypes.
func Lookup(name string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("archetype name must not be empty")
	}
	data, err := files.ReadFile(name + ".md")
	if err != nil {
		return nil, fmt.Errorf(
			"unknown archetype %q: available: %s",
			name, strings.Join(List(), ", "))
	}
	return data, nil
}

// List returns the names of all built-in archetypes, sorted.
func List() []string {
	entries, _ := files.ReadDir(".")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".md"))
	}
	sort.Strings(names)
	return names
}
