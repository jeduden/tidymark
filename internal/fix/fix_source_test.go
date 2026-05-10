package fix

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/rule"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/jeduden/mdsmith/internal/rules/linelength"
	_ "github.com/jeduden/mdsmith/internal/rules/notrailingspaces"
)

func TestFixSource_FixesTrailingSpaces(t *testing.T) {
	cfg := config.Merge(config.Defaults(), nil)
	f := &Fixer{
		Config:           cfg,
		Rules:            rule.All(),
		StripFrontMatter: true,
	}

	input := "# Hello  \n\nsome text   \n"
	fixed, remaining, errs := f.FixSource("test.md", []byte(input))

	require.Empty(t, errs)
	assert.NotEqual(t, input, string(fixed), "expected trailing spaces to be removed")
	assert.Empty(t, remaining, "expected no remaining diagnostics after fix")
}

func TestFixSource_CleanInputUnchanged(t *testing.T) {
	cfg := config.Merge(config.Defaults(), nil)
	f := &Fixer{
		Config:           cfg,
		Rules:            rule.All(),
		StripFrontMatter: true,
	}

	input := "# Hello\n\nsome text\n"
	fixed, remaining, errs := f.FixSource("test.md", []byte(input))

	require.Empty(t, errs)
	assert.Equal(t, input, string(fixed), "clean input should be returned unchanged")
	assert.Empty(t, remaining)
}

func TestFixSource_NonFixableViolationInRemaining(t *testing.T) {
	cfg := config.Merge(config.Defaults(), nil)
	f := &Fixer{
		Config:           cfg,
		Rules:            rule.All(),
		StripFrontMatter: true,
	}

	// line-length (MDS001) is not auto-fixable
	longLine := "# Hello\n\nThis line is intentionally very long to exceed the eighty character limit imposed by the default line-length rule.\n"
	_, remaining, errs := f.FixSource("test.md", []byte(longLine))

	require.Empty(t, errs)
	assert.NotEmpty(t, remaining, "expected line-length diagnostic in remaining")
	assert.Equal(t, "MDS001", remaining[0].RuleID)
}
