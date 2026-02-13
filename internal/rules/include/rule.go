package include

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jeduden/mdsmith/internal/archetype/gensection"
	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
)

func init() {
	rule.Register(&Rule{})
}

// Rule checks that include sections contain the correct file content.
type Rule struct {
	engine *gensection.Engine
}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS021" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "include" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "meta" }

// RuleID implements gensection.Directive.
func (r *Rule) RuleID() string { return "MDS021" }

// RuleName implements gensection.Directive.
func (r *Rule) RuleName() string { return "include" }

func (r *Rule) getEngine() *gensection.Engine {
	if r.engine == nil {
		r.engine = gensection.NewEngine(r)
	}
	return r.engine
}

// Check implements rule.Rule.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	if f.FS == nil {
		return nil
	}
	return r.getEngine().Check(f)
}

// Fix implements rule.FixableRule.
func (r *Rule) Fix(f *lint.File) []byte {
	if f.FS == nil {
		return f.Source
	}
	return r.getEngine().Fix(f)
}

// Validate implements gensection.Directive.
func (r *Rule) Validate(
	filePath string, line int,
	params map[string]string,
	columns map[string]gensection.ColumnConfig,
) []lint.Diagnostic {
	return validateIncludeDirective(filePath, line, params)
}

// Generate implements gensection.Directive.
func (r *Rule) Generate(
	f *lint.File, filePath string, line int,
	params map[string]string,
	columns map[string]gensection.ColumnConfig,
) (string, []lint.Diagnostic) {
	return generateIncludeContent(f, filePath, line, params)
}

func validateIncludeDirective(
	filePath string, line int,
	params map[string]string,
) []lint.Diagnostic {
	file, hasFile := params["file"]
	if !hasFile || strings.TrimSpace(file) == "" {
		return []lint.Diagnostic{makeDiag(filePath, line,
			`include directive missing required "file" parameter`)}
	}

	if filepath.IsAbs(file) {
		return []lint.Diagnostic{makeDiag(filePath, line,
			"include directive has absolute file path")}
	}

	if containsDotDot(file) {
		return []lint.Diagnostic{makeDiag(filePath, line,
			`include directive has file path with ".." traversal`)}
	}

	// Validate wrap parameter if present.
	if wrap, ok := params["wrap"]; ok && strings.TrimSpace(wrap) == "" {
		return []lint.Diagnostic{makeDiag(filePath, line,
			`include directive has empty "wrap" value`)}
	}

	// Validate strip-frontmatter parameter if present.
	if sfm, ok := params["strip-frontmatter"]; ok {
		if sfm != "true" && sfm != "false" {
			return []lint.Diagnostic{makeDiag(filePath, line,
				`include directive "strip-frontmatter" must be "true" or "false"`)}
		}
	}

	return nil
}

func generateIncludeContent(
	f *lint.File, filePath string, line int,
	params map[string]string,
) (string, []lint.Diagnostic) {
	file := params["file"]

	data, err := fs.ReadFile(f.FS, file)
	if err != nil {
		return "", []lint.Diagnostic{makeDiag(filePath, line,
			fmt.Sprintf("include file %q not found: %v", file, err))}
	}

	content := data

	// strip-frontmatter defaults to true.
	stripFM := true
	if sfm, ok := params["strip-frontmatter"]; ok && sfm == "false" {
		stripFM = false
	}

	if stripFM {
		_, stripped := lint.StripFrontMatter(content)
		content = stripped
	}

	text := string(content)

	// Trim leading blank line (common after stripping frontmatter).
	text = strings.TrimLeft(text, "\n")

	// Wrap in code fence if requested.
	if wrap, ok := params["wrap"]; ok {
		text = "```" + wrap + "\n" + text
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "```\n"
	}

	return gensection.EnsureTrailingNewline(text), nil
}

func containsDotDot(path string) bool {
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if p == ".." {
			return true
		}
	}
	return false
}

func makeDiag(file string, line int, msg string) lint.Diagnostic {
	return lint.Diagnostic{
		File:     file,
		Line:     line,
		Column:   1,
		RuleID:   "MDS021",
		RuleName: "include",
		Severity: lint.Error,
		Message:  msg,
	}
}

var _ rule.FixableRule = (*Rule)(nil)
