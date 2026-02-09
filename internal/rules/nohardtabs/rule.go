package nohardtabs

import (
	"bytes"
	"strings"

	"github.com/jeduden/tidymark/internal/lint"
	"github.com/jeduden/tidymark/internal/rule"
)

func init() {
	rule.Register(&Rule{})
}

// Rule checks that no line contains hard tab characters.
type Rule struct{}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "TM007" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "no-hard-tabs" }

// Check implements rule.Rule.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	var diags []lint.Diagnostic
	for i, line := range f.Lines {
		idx := bytes.IndexByte(line, '\t')
		if idx >= 0 {
			diags = append(diags, lint.Diagnostic{
				File:     f.Path,
				Line:     i + 1,
				Column:   idx + 1,
				RuleID:   r.ID(),
				RuleName: r.Name(),
				Severity: lint.Warning,
				Message:  "hard tab character",
			})
		}
	}
	return diags
}

// Fix implements rule.FixableRule.
func (r *Rule) Fix(f *lint.File) []byte {
	var result []string
	for _, line := range f.Lines {
		replaced := strings.ReplaceAll(string(line), "\t", "    ")
		result = append(result, replaced)
	}
	return []byte(strings.Join(result, "\n"))
}
