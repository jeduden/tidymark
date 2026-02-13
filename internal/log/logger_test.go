package log

import (
	"bytes"
	"testing"
)

func TestPrintf_Enabled(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{Enabled: true, W: &buf}

	l.Printf("config: %s", ".mdsmith.yml")

	want := "config: .mdsmith.yml\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrintf_Disabled(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{Enabled: false, W: &buf}

	l.Printf("config: %s", ".mdsmith.yml")

	if got := buf.String(); got != "" {
		t.Errorf("expected no output, got %q", got)
	}
}

func TestPrintf_MultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{Enabled: true, W: &buf}

	l.Printf("file: %s", "README.md")
	l.Printf("rule: %s %s", "MDS001", "line-length")

	want := "file: README.md\nrule: MDS001 line-length\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
