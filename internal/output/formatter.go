package output

import (
	"io"

	"github.com/jeduden/mdsmith/internal/lint"
)

// Formatter defines the interface for outputting diagnostics.
type Formatter interface {
	Format(w io.Writer, diagnostics []lint.Diagnostic) error
}
