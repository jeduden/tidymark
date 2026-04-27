package main

import (
	"os"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/lint"
)

// attachExplanations populates Diagnostic.Explanation for each
// diagnostic. For each unique file, it parses front-matter kinds,
// resolves provenance, and looks up the diagnostic's rule. The
// "winning source" reported is the source that wrote the rule's
// "enabled" leaf — under block-replace today, every leaf in a rule
// shares the same source, so this matches what plan 95 promises:
// "the source of the setting that triggered it".
//
// Files that fail to read or parse degrade gracefully — the
// diagnostic keeps its current Explanation (nil).
func attachExplanations(cfg *config.Config, diags []lint.Diagnostic) {
	type fileCtx struct {
		res *config.Resolution
	}
	cache := make(map[string]fileCtx)

	for i := range diags {
		d := &diags[i]
		if d.RuleName == "" {
			continue
		}
		ctx, ok := cache[d.File]
		if !ok {
			ctx = fileCtx{res: resolveFileForExplain(cfg, d.File)}
			cache[d.File] = ctx
		}
		if ctx.res == nil {
			continue
		}
		rp, ok := ctx.res.Rules[d.RuleName]
		if !ok {
			continue
		}
		exp := &lint.Explanation{
			Rule:        d.RuleName,
			Source:      winningSource(rp),
			Kinds:       append([]string(nil), ctx.res.EffectiveKinds...),
			LeafSources: leafSources(rp),
		}
		d.Explanation = exp
	}
}

// resolveFileForExplain reads file front-matter (best effort) and
// returns the per-file Resolution. Returns nil if the file cannot be
// read; callers fall back to leaving the diagnostic un-explained.
func resolveFileForExplain(cfg *config.Config, file string) *config.Resolution {
	var fmKinds []string
	data, err := os.ReadFile(file) // #nosec G304 — file path comes from the user's lint args
	if err == nil {
		prefix, _ := lint.StripFrontMatter(data)
		fmKinds, _ = lint.ParseFrontMatterKinds(prefix)
	}
	return config.ResolveWithProvenance(cfg, file, fmKinds)
}

// winningSource picks the layer that wrote the rule's enabled bit. If
// the rule has no leaves, fall back to "default".
func winningSource(rp config.RuleProvenance) string {
	if leaf, ok := rp.Leaves["enabled"]; ok && leaf.WinningSource != "" {
		return leaf.WinningSource
	}
	// Fall back to any leaf's winning source.
	for _, leaf := range rp.Leaves {
		if leaf.WinningSource != "" {
			return leaf.WinningSource
		}
	}
	return "default"
}

// leafSources returns a copy of rp's leaf -> winning-source mapping
// suitable for embedding in JSON output.
func leafSources(rp config.RuleProvenance) map[string]string {
	if len(rp.Leaves) == 0 {
		return nil
	}
	out := make(map[string]string, len(rp.Leaves))
	for k, leaf := range rp.Leaves {
		out[k] = leaf.WinningSource
	}
	return out
}
