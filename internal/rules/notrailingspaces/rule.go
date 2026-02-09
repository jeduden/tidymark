package notrailingspaces

import (
	"bytes"
	"strings"

	"github.com/jeduden/tidymark/internal/lint"
	"github.com/jeduden/tidymark/internal/rule"
)

func init() {
	rule.Register(&Rule{})
}

// Rule checks that no line ends with trailing spaces or tabs.
type Rule struct{}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "TM006" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "no-trailing-spaces" }

// Check implements rule.Rule.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	var diags []lint.Diagnostic
	for i, line := range f.Lines {
		trimmed := bytes.TrimRight(line, " \t")
		if len(trimmed) < len(line) {
			diags = append(diags, lint.Diagnostic{
				File:     f.Path,
				Line:     i + 1,
				Column:   len(trimmed) + 1,
				RuleID:   r.ID(),
				RuleName: r.Name(),
				Severity: lint.Warning,
				Message:  "trailing whitespace",
			})
		}
	}
	return diags
}

// Fix implements rule.FixableRule.
func (r *Rule) Fix(f *lint.File) []byte {
	var result []string
	for _, line := range f.Lines {
		trimmed := bytes.TrimRight(line, " \t")
		result = append(result, string(trimmed))
	}
	return []byte(strings.Join(result, "\n"))
}
