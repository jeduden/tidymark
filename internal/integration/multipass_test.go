package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/fix"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiPassFix_TOCDirective verifies that a single fix run converts
// [TOC] into a populated <?toc?>...<?/toc?> block. MDS035 replaces [TOC]
// with the empty directive in pass 1; MDS038 fills the heading list in pass 2.
func TestMultiPassFix_TOCDirective(t *testing.T) {
	src := "# Document\n\n[TOC]\n\n## Section One\n\n## Section Two\n"

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(mdFile, []byte(src), 0o644))

	cfg := &config.Config{
		Rules: map[string]config.RuleCfg{
			"toc-directive": {Enabled: true},
			"toc":           {Enabled: true},
		},
	}

	fixer := &fix.Fixer{
		Config: cfg,
		Rules:  rule.All(),
	}

	result := fixer.Fix([]string{mdFile})
	require.Empty(t, result.Errors, "unexpected errors: %v", result.Errors)

	got, err := os.ReadFile(mdFile)
	require.NoError(t, err)

	content := string(got)
	assert.Contains(t, content, "<?toc?>", "toc directive start marker present")
	assert.Contains(t, content, "<?/toc?>", "toc directive end marker present")
	assert.Contains(t, content, "- [Section One](#section-one)", "heading link present")
	assert.Contains(t, content, "- [Section Two](#section-two)", "heading link present")
	assert.NotContains(t, content, "\n[TOC]\n", "original TOC token replaced")
}
