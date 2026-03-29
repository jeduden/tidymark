package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/jeduden/mdsmith/internal/lint"
)

// TextFormatter outputs diagnostics in human-readable text format.
// When Color is true, the file location is printed in cyan and the rule ID in yellow.
type TextFormatter struct {
	Color bool
}

// Format writes each diagnostic as a header line followed by an optional
// source snippet with line-number gutter and caret marker.
func (f *TextFormatter) Format(w io.Writer, diagnostics []lint.Diagnostic) error {
	for _, d := range diagnostics {
		var err error
		if f.Color {
			_, err = fmt.Fprintf(w, "\033[36m%s:%d:%d\033[0m \033[33m%s\033[0m %s\n",
				d.File, d.Line, d.Column, d.RuleID, d.Message)
		} else {
			_, err = fmt.Fprintf(w, "%s:%d:%d %s %s\n",
				d.File, d.Line, d.Column, d.RuleID, d.Message)
		}
		if err != nil {
			return err
		}

		if err := f.formatSnippet(w, d); err != nil {
			return err
		}
	}
	return nil
}

// formatSnippet writes the source context lines with a line-number gutter
// and either a > marker (whole-line, Column≤1) or a dot-leader caret
// (exact position, Column>1) for the diagnostic line.
func (f *TextFormatter) formatSnippet(w io.Writer, d lint.Diagnostic) error {
	if len(d.SourceLines) == 0 {
		return nil
	}

	maxLineNum := d.SourceStartLine + len(d.SourceLines) - 1
	gutterWidth := len(fmt.Sprintf("%d", maxLineNum))
	if gutterWidth < 2 {
		gutterWidth = 2
	}

	for i, line := range d.SourceLines {
		lineNum := d.SourceStartLine + i
		isDiagLine := lineNum == d.Line

		if err := f.writeSourceLine(w, gutterWidth, lineNum, line, isDiagLine); err != nil {
			return err
		}

		if isDiagLine && d.Column > 1 {
			if err := f.writeCaretLine(w, gutterWidth, d.Column); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeSourceLine writes a single source line with line-number gutter.
// Diagnostic lines get a > prefix; context lines are dimmed when color is on.
func (f *TextFormatter) writeSourceLine(w io.Writer, gutterWidth, lineNum int, line string, isDiag bool) error {
	if isDiag {
		format := ">%*d | %s\n"
		if f.Color {
			format = "\033[31m>\033[0m%*d | %s\n"
		}
		_, err := fmt.Fprintf(w, format, gutterWidth-1, lineNum, line)
		return err
	}
	format := "%*d | %s\n"
	if f.Color {
		format = "\033[2m%*d | %s\033[0m\n"
	}
	_, err := fmt.Fprintf(w, format, gutterWidth, lineNum, line)
	return err
}

// writeCaretLine writes the dot-leader caret line under the diagnostic column.
func (f *TextFormatter) writeCaretLine(w io.Writer, gutterWidth, column int) error {
	caretPad := strings.Repeat("·", column-1)
	gutterPad := strings.Repeat(" ", gutterWidth)
	format := "%s | %s^\n"
	if f.Color {
		format = "%s | %s\033[31m^\033[0m\n"
	}
	_, err := fmt.Fprintf(w, format, gutterPad, caretPad)
	return err
}
