package engine

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test exposes the issue with empty lines at the end of files
func TestCheckRules_PopulatesSourceContextAtFileEnd(t *testing.T) {
	// File with trailing newline: will have empty element in Lines array
	source := "line one\nline two\nline three\nline four\nline five\n"
	f, err := lint.NewFile("test.md", []byte(source))
	require.NoError(t, err)

	// Diagnostic on the last actual line (line 5)
	r := &mockRuleAtLine{id: "MDS998", name: "end-rule", line: 5, col: 1}
	effective := map[string]config.RuleCfg{
		"end-rule": {Enabled: true},
	}

	diags, errs := CheckRules(f, []rule.Rule{r}, effective)
	require.Len(t, errs, 0)
	require.Len(t, diags, 1)

	d := diags[0]
	assert.Equal(t, 5, d.Line)
	// With context=2 from line 5: should include lines 3, 4, 5
	require.Len(t, d.SourceLines, 3, "expected 3 context lines (line 5 - 2)")

	// Check that no empty line is included
	for i, line := range d.SourceLines {
		assert.NotEmpty(t, line, "SourceLines[%d] should not be empty", i)
	}

	assert.Equal(t, "line three", d.SourceLines[0])
	assert.Equal(t, "line four", d.SourceLines[1])
	assert.Equal(t, "line five", d.SourceLines[2])
}
