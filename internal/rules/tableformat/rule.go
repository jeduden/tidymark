// Package tableformat implements MDS025, the single table rule.
// It owns table parsing, the three structural checks ported from the
// retired MDS060 (MD055 table-pipe-style, MD056 table-column-count,
// MD058 blanks-around-tables), and the prettier-style alignment pass
// that gives the rule its name.
package tableformat

import (
	"fmt"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/jeduden/mdsmith/internal/rules/settings"
	"github.com/jeduden/mdsmith/internal/rules/tablefmt"
)

func init() {
	rule.Register(&Rule{Pad: 1, Style: StyleConsistent})
}

// Rule gates table well-formedness: edge-pipe style (MD055), column
// count vs the header (MD056), surrounding blank lines (MD058), and
// the column-alignment / padding pass that owns prettier-style
// table formatting.
type Rule struct {
	Pad   int    // spaces on each side of cell content
	Style string // edge-pipe style: one of the Style* constants
}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS025" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "table-format" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "table" }

// GetPad returns the current pad setting.
func (r *Rule) GetPad() int { return r.Pad }

// ApplySettings implements rule.Configurable.
func (r *Rule) ApplySettings(s map[string]any) error {
	for k, v := range s {
		switch k {
		case "pad":
			n, ok := settings.ToInt(v)
			if !ok {
				return fmt.Errorf("table-format: pad must be an integer, got %T", v)
			}
			if n < 0 {
				return fmt.Errorf("table-format: pad must be non-negative, got %d", n)
			}
			r.Pad = n
		case "style":
			str, ok := v.(string)
			if !ok {
				return fmt.Errorf("table-format: style must be a string, got %T", v)
			}
			switch str {
			case StyleConsistent, StyleLeadingAndTrailing, StyleNoLeadingOrTrailing:
				r.Style = str
			default:
				return fmt.Errorf(
					"table-format: invalid style %q (valid: %s, %s, %s)",
					str, StyleConsistent, StyleLeadingAndTrailing, StyleNoLeadingOrTrailing)
			}
		default:
			return fmt.Errorf("table-format: unknown setting %q", k)
		}
	}
	return nil
}

// DefaultSettings implements rule.Configurable.
func (r *Rule) DefaultSettings() map[string]any {
	return map[string]any{
		"pad":   1,
		"style": StyleConsistent,
	}
}

// Check implements rule.Rule. It emits both the structural diagnostics
// (MD055/056/058) and the alignment diagnostics produced by the
// prettier-style format pass.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	skipLines := formatSkipLines(f)
	var diags []lint.Diagnostic
	for _, v := range tablefmt.Violations(f.Lines, skipLines, r.Pad) {
		diags = append(diags, lint.Diagnostic{
			File:     f.Path,
			Line:     v.StartLine,
			Column:   1,
			RuleID:   r.ID(),
			RuleName: r.Name(),
			Severity: lint.Warning,
			Message:  v.Message,
		})
	}
	diags = append(diags, structureDiagnostics(f, r.style(), r.ID(), r.Name())...)
	return diags
}

// Fix implements rule.FixableRule. The structure pass runs first
// (edge normalization for MD055, blank-line insertion for MD058) so
// the alignment pass then sees the structurally-normalized bytes and
// canonicalizes the remaining bordered tables.
func (r *Rule) Fix(f *lint.File) []byte {
	intermediate := applyStructureFix(f, r.style())
	parsed, _ := lint.NewFile(f.Path, intermediate) // NewFile never errors today
	parsed.GeneratedRanges = f.GeneratedRanges
	skipLines := formatSkipLines(parsed)
	return tablefmt.FormatLines(parsed.Source, parsed.Lines, skipLines, r.Pad)
}

// formatSkipLines returns the line numbers the alignment pass must
// ignore: fenced/indented code, processing-instruction blocks, and
// generated-section bodies. Tablefmt previously honored only code
// blocks; bringing it into line with the structure pass keeps both
// passes from touching content the source file does not own. The
// returned map is freshly allocated — `lint.Collect*BlockLines`
// return shared, read-only caches that must not be mutated.
func formatSkipLines(f *lint.File) map[int]bool {
	skip := map[int]bool{}
	for n := range lint.CollectCodeBlockLines(f) {
		skip[n] = true
	}
	for n := range lint.CollectPIBlockLines(f) {
		skip[n] = true
	}
	for _, gr := range f.GeneratedRanges {
		for n := gr.From; n <= gr.To; n++ {
			skip[n] = true
		}
	}
	return skip
}

// style returns the configured pipe style, defaulting to the
// consistent style so a Rule literal without an explicit Style
// (legacy callers and tests) keeps working.
func (r *Rule) style() string {
	if r.Style == "" {
		return StyleConsistent
	}
	return r.Style
}

var (
	_ rule.FixableRule  = (*Rule)(nil)
	_ rule.Configurable = (*Rule)(nil)
)
