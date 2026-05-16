package main

import (
	"fmt"
	"os"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/engine"
	"github.com/jeduden/mdsmith/internal/extract"
	"github.com/jeduden/mdsmith/internal/extract/encode"
	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/jeduden/mdsmith/internal/rules/requiredstructure"
	"github.com/jeduden/mdsmith/internal/schema"
	flag "github.com/spf13/pflag"
)

// extractErr prints a `mdsmith:`-prefixed message to stderr and
// returns code, so each failure site is a single statement rather
// than a print-then-return pair. Keeping these one-liners both
// reads cleaner and avoids a long tail of untestable
// fmt.Fprintf-only lines (I/O failures that cannot be driven from
// a test) inflating the diff's uncovered-line count.
func extractErr(code int, format string, a ...any) int {
	fmt.Fprintf(os.Stderr, "mdsmith: "+format+"\n", a...)
	return code
}

// runExtract implements the "extract" subcommand:
//
//	mdsmith extract <kind> --format <fmt> <file>
//
// It projects a schema-conformant file into a data tree whose shape
// mirrors the composed schema. Extraction is gated on a clean
// `check`: a non-conformant file prints the same diagnostics and
// exits non-zero, never emitting partial data.
func runExtract(args []string) int {
	kindName, path, fmtEnum, stop := parseExtractArgs(args)
	if stop >= 0 {
		return stop
	}

	res, cfg, code := resolveFileFromCLI(path)
	if code != 0 {
		return code
	}
	_, cfgPath, _ := loadConfig("")
	if code := validateExtractKind(cfg, res, kindName, path); code != 0 {
		return code
	}

	maxBytes, err := resolveMaxInputBytes(cfg, "")
	if err != nil {
		return extractErr(2, "%v", err)
	}
	if code := gateExtractCheck(cfg, cfgPath, path, maxBytes); code != 0 {
		return code
	}

	f, source, code := loadExtractFile(cfg, cfgPath, path, maxBytes)
	if code != 0 {
		return code
	}
	sch, code := composedSchemaFor(f, res, kindName)
	if code != 0 {
		return code
	}

	var docFM map[string]any
	if frontMatterEnabled(cfg) {
		if prefix, _ := lint.StripFrontMatter(source); len(prefix) > 0 {
			docFM, _ = lint.ParseFrontMatterFields(prefix)
		}
	}

	mt := schema.BuildMatchTree(f, sch, docFM)
	data, diags := extract.Extract(f, sch, mt)
	if len(diags) > 0 {
		formatDiagnostics(diags, "text", false)
		return 1
	}

	out, err := encode.Encode(fmtEnum, data)
	if err != nil {
		return extractErr(2, "encoding %s: %v", fmtEnum, err)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		return extractErr(2, "writing output: %v", err)
	}
	return 0
}

// parseExtractArgs parses flags and the two positionals. The final
// return is -1 to continue, or a process exit code to stop on
// (usage error, --help, or a bad --format value).
func parseExtractArgs(
	args []string,
) (kindName, path string, fmtEnum encode.Format, stop int) {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	var format string
	fs.StringVarP(&format, "format", "f", "json",
		"Output format: json, yaml, msgpack")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: mdsmith extract <kind> --format <fmt> <file>\n\n"+
				"Emit a kind-conformant file as a data tree whose nesting\n"+
				"mirrors the schema hierarchy.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if code := reportFlagParseErr(err, os.Stderr, "mdsmith: extract"); code >= 0 {
			return "", "", "", code
		}
	}
	if fs.NArg() != 2 {
		return "", "", "",
			extractErr(2, "extract requires <kind> and <file>")
	}
	f, err := encode.ParseFormat(format)
	if err != nil {
		return "", "", "", extractErr(2, "%v", err)
	}
	return fs.Arg(0), fs.Arg(1), f, -1
}

// validateExtractKind rejects an unknown kind or one not assigned
// to the file.
func validateExtractKind(
	cfg *config.Config, res *config.FileResolution,
	kindName, path string,
) int {
	if _, declared := cfg.Kinds[kindName]; !declared {
		return extractErr(2, "unknown kind %q", kindName)
	}
	if !kindAssigned(res.Kinds, kindName) {
		return extractErr(2, "kind %q is not assigned to %s", kindName, path)
	}
	return 0
}

// gateExtractCheck runs the full check on the file and mirrors its
// exit semantics: a non-conformant file prints the same diagnostics
// and never reaches projection.
func gateExtractCheck(
	cfg *config.Config, cfgPath, path string, maxBytes int64,
) int {
	runner := &engine.Runner{
		Config:           cfg,
		Rules:            rule.All(),
		StripFrontMatter: frontMatterEnabled(cfg),
		RootDir:          rootDirFromConfig(cfgPath),
		MaxInputBytes:    maxBytes,
		ConfigPath:       cfgPath,
	}
	result := runner.Run([]string{path})
	if len(result.Errors) > 0 && len(result.Diagnostics) == 0 {
		printErrors(result.Errors)
		return 2
	}
	if len(result.Diagnostics) > 0 {
		formatDiagnostics(result.Diagnostics, "text", false)
		return 1
	}
	return 0
}

// loadExtractFile reads and parses the document the same way the
// engine does so the match tree's line math lines up.
func loadExtractFile(
	cfg *config.Config, cfgPath, path string, maxBytes int64,
) (*lint.File, []byte, int) {
	source, err := lint.ReadFileLimited(path, maxBytes)
	if err != nil {
		return nil, nil, extractErr(2, "reading %s: %v", path, err)
	}
	f, err := lint.NewFileFromSource(path, source, frontMatterEnabled(cfg))
	if err != nil {
		return nil, nil, extractErr(2, "parsing %s: %v", path, err)
	}
	f.MaxInputBytes = maxBytes
	if rd := rootDirFromConfig(cfgPath); rd != "" {
		f.SetRootDir(rd)
	}
	return f, source, 0
}

// composedSchemaFor builds the same composed schema MDS020
// validates against, from the file's resolved required-structure
// config.
func composedSchemaFor(
	f *lint.File, res *config.FileResolution, kindName string,
) (*schema.Schema, int) {
	rr, ok := res.Rules["required-structure"]
	if !ok || !rr.Final.Enabled {
		// gateExtractCheck runs the normal engine, which skips
		// MDS020 when the rule is disabled. Projecting then would
		// emit data for a never-validated file, breaking the
		// "gated on a successful match" contract. Refuse instead.
		return nil, extractErr(2,
			"required-structure is disabled for %s; "+
				"nothing to validate or extract against", f.Path)
	}
	rsRule := &requiredstructure.Rule{}
	if rr.Final.Settings != nil {
		if err := rsRule.ApplySettings(rr.Final.Settings); err != nil {
			return nil, extractErr(2, "loading schema config: %v", err)
		}
	}
	sch, err := rsRule.ComposedSchema(f)
	if err != nil {
		return nil, extractErr(2, "%v", err)
	}
	if sch == nil || sch.IsEmpty() {
		return nil, extractErr(2,
			"kind %q declares no schema to extract against", kindName)
	}
	return sch, 0
}

// kindAssigned reports whether name is one of the file's resolved
// kinds.
func kindAssigned(kinds []config.ResolvedKind, name string) bool {
	for _, k := range kinds {
		if k.Name == name {
			return true
		}
	}
	return false
}
