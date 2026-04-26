package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Register the required-structure rule so its registration appears in
	// rule.All() when the test reaches into the registry.
	_ "github.com/jeduden/mdsmith/internal/rules/requiredstructure"
)

// TestRunner_KindRequiredStructureSchemaApplied verifies that a kind whose
// body sets `rules.required-structure.schema:` causes files bound to that
// kind to be validated against the schema. A document missing a required
// heading from the schema should produce a diagnostic.
func TestRunner_KindRequiredStructureSchemaApplied(t *testing.T) {
	dir := t.TempDir()

	// Write a schema requiring "## Goal".
	schemaPath := filepath.Join(dir, "schema.md")
	require.NoError(t, os.WriteFile(schemaPath,
		[]byte("# ?\n\n## Goal\n"), 0o644))

	// Doc bound to the kind by front matter; intentionally missing
	// "## Goal" so the schema is violated.
	docPath := filepath.Join(dir, "doc.md")
	body := "---\nkinds: [proto]\n---\n# Title\n"
	require.NoError(t, os.WriteFile(docPath, []byte(body), 0o644))

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"required-structure": {Enabled: true},
		},
		Kinds: map[string]config.Kind{
			"proto": {Rules: map[string]config.RuleCfg{
				"required-structure": {
					Enabled:  true,
					Settings: map[string]any{"schema": schemaPath},
				},
			}},
		},
	}

	// Find the registered required-structure rule.
	var rs rule.Rule
	for _, r := range rule.All() {
		if r.Name() == "required-structure" {
			rs = r
			break
		}
	}
	require.NotNil(t, rs, "required-structure rule must be registered")

	runner := &Runner{
		Config:           cfg,
		Rules:            []rule.Rule{rs},
		StripFrontMatter: true,
	}

	res := runner.Run([]string{docPath})
	require.Empty(t, res.Errors, "unexpected errors: %v", res.Errors)
	require.NotEmpty(t, res.Diagnostics,
		"expected at least one diagnostic from schema validation")
	found := false
	for _, d := range res.Diagnostics {
		if d.RuleName == "required-structure" &&
			containsStr(d.Message, "Goal") {
			found = true
			break
		}
	}
	assert.True(t, found,
		"expected a required-structure diagnostic mentioning the missing Goal heading")
}

func containsStr(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
