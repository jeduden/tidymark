package main

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// composedSchemaFor must refuse when required-structure is absent
// or disabled for the file: gateExtractCheck would skip MDS020 in
// that configuration, so projecting would emit data for a
// never-validated file (Copilot review on extract.go).
func TestComposedSchemaFor_RefusesWhenRuleDisabled(t *testing.T) {
	f, err := lint.NewFile("doc.md", []byte("# T\n"))
	require.NoError(t, err)

	// Rule absent from the resolution.
	_, code := composedSchemaFor(f, &config.FileResolution{}, "k")
	assert.Equal(t, 2, code)

	// Rule present but disabled.
	res := &config.FileResolution{
		Rules: map[string]config.RuleResolution{
			"required-structure": {Final: config.RuleCfg{Enabled: false}},
		},
	}
	_, code = composedSchemaFor(f, res, "k")
	assert.Equal(t, 2, code)
}

func TestKindAssigned(t *testing.T) {
	kinds := []config.ResolvedKind{{Name: "a"}, {Name: "b"}}
	assert.True(t, kindAssigned(kinds, "b"))
	assert.False(t, kindAssigned(kinds, "c"))
}
