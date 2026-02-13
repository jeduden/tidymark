package log

import (
	"fmt"
	"io"
)

// Logger writes verbose diagnostic messages when Enabled is true.
// Output goes to the configured writer (typically stderr).
type Logger struct {
	Enabled bool
	W       io.Writer
}

// Printf writes a formatted message to W when Enabled is true.
// It is a no-op when Enabled is false.
func (l *Logger) Printf(format string, args ...any) {
	if !l.Enabled {
		return
	}
	_, _ = fmt.Fprintf(l.W, format+"\n", args...)
}
