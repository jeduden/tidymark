package mcp

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/engine"
	fixpkg "github.com/jeduden/mdsmith/internal/fix"
	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
)

// MdsmithTools implements toolsHandler using the mdsmith lint and fix pipeline.
type MdsmithTools struct{}

func (MdsmithTools) list() []toolDef {
	return []toolDef{
		{
			Name:        "mdsmith_check",
			Description: "Lint Markdown content with mdsmith and return diagnostics as JSON.",
			InputSchema: inputSch{
				Type: "object",
				Properties: map[string]schemaProp{
					"content": {
						Type:        "string",
						Description: "Markdown text to lint.",
					},
					"filename": {
						Type:        "string",
						Description: "Logical filename (used for config discovery and rule context).",
					},
					"config": {
						Type:        "string",
						Description: "Path to .mdsmith.yml override (equivalent to -c flag).",
					},
				},
				Required: []string{"content"},
			},
		},
		{
			Name:        "mdsmith_fix",
			Description: "Auto-fix Markdown content with mdsmith and return corrected content.",
			InputSchema: inputSch{
				Type: "object",
				Properties: map[string]schemaProp{
					"content": {
						Type:        "string",
						Description: "Markdown text to fix.",
					},
					"filename": {
						Type:        "string",
						Description: "Logical filename for config discovery.",
					},
					"config": {
						Type:        "string",
						Description: "Path to .mdsmith.yml override (equivalent to -c flag).",
					},
				},
				Required: []string{"content"},
			},
		},
	}
}

func (MdsmithTools) call(name string, args json.RawMessage) (any, error) {
	switch name {
	case "mdsmith_check":
		return callCheck(args)
	case "mdsmith_fix":
		return callFix(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// toolArgs holds the common input fields for both tools.
type toolArgs struct {
	Content  string `json:"content"`
	Filename string `json:"filename"`
	Config   string `json:"config"`
}

// diagnosticJSON mirrors the JSON output shape of mdsmith check --format json.
type diagnosticJSON struct {
	File            string   `json:"file"`
	Line            int      `json:"line"`
	Column          int      `json:"column"`
	Rule            string   `json:"rule"`
	Name            string   `json:"name"`
	Severity        string   `json:"severity"`
	Message         string   `json:"message"`
	SourceLines     []string `json:"source_lines,omitempty"`
	SourceStartLine int      `json:"source_start_line,omitempty"`
}

func diagsToJSON(diags []lint.Diagnostic) []diagnosticJSON {
	out := make([]diagnosticJSON, 0, len(diags))
	for _, d := range diags {
		j := diagnosticJSON{
			File:     d.File,
			Line:     d.Line,
			Column:   d.Column,
			Rule:     d.RuleID,
			Name:     d.RuleName,
			Severity: string(d.Severity),
			Message:  d.Message,
		}
		if len(d.SourceLines) > 0 {
			j.SourceLines = d.SourceLines
			j.SourceStartLine = d.SourceStartLine
		}
		out = append(out, j)
	}
	return out
}

func loadCfg(configPath, filename string) (*config.Config, string, error) {
	defaults := config.Defaults()

	if configPath != "" {
		loaded, err := config.Load(configPath)
		if err != nil {
			return nil, "", fmt.Errorf("loading config %q: %w", configPath, err)
		}
		merged := config.Merge(defaults, loaded)
		config.InjectBuildConfig(merged, configPath)
		return merged, configPath, nil
	}

	// Walk up from the directory of filename to discover .mdsmith.yml.
	dir := "."
	if filename != "" {
		if filepath.IsAbs(filename) {
			dir = filepath.Dir(filename)
		} else {
			dir = filepath.Dir(filename)
		}
	}
	discovered, err := config.Discover(dir)
	if err != nil || discovered == "" {
		merged := config.Merge(defaults, nil)
		config.InjectBuildConfig(merged, "")
		return merged, "", nil
	}
	loaded, err := config.Load(discovered)
	if err != nil {
		return nil, "", fmt.Errorf("loading config %q: %w", discovered, err)
	}
	merged := config.Merge(defaults, loaded)
	config.InjectBuildConfig(merged, discovered)
	return merged, discovered, nil
}

func callCheck(raw json.RawMessage) (any, error) {
	var args toolArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	cfg, cfgPath, err := loadCfg(args.Config, args.Filename)
	if err != nil {
		return nil, err
	}

	name := args.Filename
	if name == "" {
		name = "<content>"
	}

	runner := &engine.Runner{
		Config:           cfg,
		Rules:            rule.All(),
		StripFrontMatter: frontMatterEnabled(cfg),
		RootDir:          rootDir(cfgPath),
	}
	result := runner.RunSource(name, []byte(args.Content))
	if len(result.Errors) > 0 {
		return nil, result.Errors[0]
	}
	return diagsToJSON(result.Diagnostics), nil
}

// fixResult is the response shape for mdsmith_fix.
type fixResult struct {
	Content   string           `json:"content"`
	Changed   bool             `json:"changed"`
	Remaining []diagnosticJSON `json:"remaining"`
}

func callFix(raw json.RawMessage) (any, error) {
	var args toolArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	cfg, cfgPath, err := loadCfg(args.Config, args.Filename)
	if err != nil {
		return nil, err
	}

	name := args.Filename
	if name == "" {
		name = "<content>"
	}

	fixer := &fixpkg.Fixer{
		Config:           cfg,
		Rules:            rule.All(),
		StripFrontMatter: frontMatterEnabled(cfg),
		RootDir:          rootDir(cfgPath),
	}
	source := []byte(args.Content)
	fixed, remaining, errs := fixer.FixSource(name, source)
	if len(errs) > 0 {
		return nil, errs[0]
	}

	return fixResult{
		Content:   string(fixed),
		Changed:   string(fixed) != args.Content,
		Remaining: diagsToJSON(remaining),
	}, nil
}

// frontMatterEnabled returns true when front-matter stripping is configured
// (defaults to true when the setting is unset).
func frontMatterEnabled(cfg *config.Config) bool {
	if cfg.FrontMatter != nil {
		return *cfg.FrontMatter
	}
	return true
}

// rootDir extracts the project root directory from the config file path.
func rootDir(cfgPath string) string {
	if cfgPath == "" {
		return ""
	}
	return filepath.Dir(cfgPath)
}
