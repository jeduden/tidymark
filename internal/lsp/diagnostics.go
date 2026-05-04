package lsp

import (
	"strings"
	"unicode/utf16"

	"github.com/jeduden/mdsmith/internal/lint"
)

// toLSP converts an mdsmith diagnostic to the LSP wire shape.
//
// Coordinates flip from 1-based (mdsmith) to 0-based (LSP). The end
// column is derived from the source so the squiggle covers the
// remainder of the line; rules can later widen this with their own
// per-rule end column once LSP-aware spans land in the engine.
//
// LSP positions count UTF-16 code units, not bytes. mdsmith
// columns are 1-based rune offsets, so we convert by counting
// UTF-16 units up to the rune at the diagnostic column.
func toLSP(d lint.Diagnostic, lines [][]byte) Diagnostic {
	startLine := d.Line - 1
	if startLine < 0 {
		startLine = 0
	}
	line := currentLine(lines, d.Line)
	startCol := utf16Column(line, d.Column-1)
	// End at the line's UTF-16 length so empty lines produce a
	// zero-width range (start == end) instead of a range whose end
	// character (1) lies past the actual line length. Clients
	// typically clamp out-of-range positions, but emitting a valid
	// range is cheaper and avoids surprising downstream tooling.
	endCol := utf16Column(line, runeLen(line))
	if endCol < startCol {
		endCol = startCol
	}
	return Diagnostic{
		Range: Range{
			Start: Position{Line: startLine, Character: startCol},
			End:   Position{Line: startLine, Character: endCol},
		},
		Severity: severityFor(d.Severity),
		Code:     d.RuleID,
		Source:   "mdsmith",
		Message:  d.Message,
		Data:     &diagnosticData{RuleName: d.RuleName},
	}
}

// toLSPAll maps a slice. Returns an empty (non-nil) slice for empty
// input so the JSON wire form is `[]`, never `null`.
func toLSPAll(diags []lint.Diagnostic, source []byte) []Diagnostic {
	out := make([]Diagnostic, 0, len(diags))
	lines := splitLines(source)
	for _, d := range diags {
		out = append(out, toLSP(d, lines))
	}
	return out
}

func severityFor(s lint.Severity) DiagnosticSeverity {
	if s == lint.Warning {
		return severityWarning
	}
	return severityError
}

// splitLines splits source into per-line byte slices, preserving
// trailing empty lines so the indexing matches lint.File.Lines (which
// uses bytes.Split). Rules such as single-trailing-newline emit
// diagnostics anchored at len(f.Lines) for trailing whitespace runs;
// trimming the trailing newlines here would make currentLine() return
// "" and toLSP would clamp to a position past the document. Each line
// has its trailing CR stripped so Windows-style line endings produce
// matching positions on the wire.
func splitLines(source []byte) [][]byte {
	if len(source) == 0 {
		return nil
	}
	parts := strings.Split(string(source), "\n")
	out := make([][]byte, len(parts))
	for i, p := range parts {
		out[i] = []byte(strings.TrimRight(p, "\r"))
	}
	return out
}

// currentLine returns the content of 1-based line number n as a
// string, or "" when out of range.
func currentLine(lines [][]byte, n int) string {
	if n < 1 || n > len(lines) {
		return ""
	}
	return string(lines[n-1])
}

// runeLen returns the number of runes in s.
func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// utf16Column returns the UTF-16 code-unit offset that corresponds to
// rune offset col in s. Clamps to the string's UTF-16 length.
//
// utf16.RuneLen returns -1 for unpaired surrogates and other invalid
// code points; we treat those as a single UTF-16 unit so positions
// stay non-negative even when the document contains adversarial
// input.
func utf16Column(s string, col int) int {
	if col <= 0 {
		return 0
	}
	units := 0
	consumed := 0
	for _, r := range s {
		if consumed >= col {
			break
		}
		w := utf16.RuneLen(r)
		if w < 0 {
			w = 1
		}
		units += w
		consumed++
	}
	return units
}
