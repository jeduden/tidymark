package notrailingspaces

import (
	"testing"

	"github.com/jeduden/tidymark/internal/lint"
)

func TestCheck_TrailingSpaces(t *testing.T) {
	src := []byte("hello   \nworld\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	d := diags[0]
	if d.Line != 1 {
		t.Errorf("expected line 1, got %d", d.Line)
	}
	if d.Column != 6 {
		t.Errorf("expected column 6, got %d", d.Column)
	}
	if d.RuleID != "TM006" {
		t.Errorf("expected rule ID TM006, got %s", d.RuleID)
	}
}

func TestCheck_TrailingTabs(t *testing.T) {
	src := []byte("hello\t\nworld\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	d := diags[0]
	if d.Line != 1 {
		t.Errorf("expected line 1, got %d", d.Line)
	}
	if d.Column != 6 {
		t.Errorf("expected column 6, got %d", d.Column)
	}
}

func TestCheck_NoViolation(t *testing.T) {
	src := []byte("hello\nworld\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}

func TestCheck_EmptyFile(t *testing.T) {
	src := []byte("")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}

func TestCheck_MultipleViolations(t *testing.T) {
	src := []byte("hello   \nworld  \nfoo\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diags))
	}
	if diags[0].Line != 1 {
		t.Errorf("expected first diagnostic on line 1, got %d", diags[0].Line)
	}
	if diags[1].Line != 2 {
		t.Errorf("expected second diagnostic on line 2, got %d", diags[1].Line)
	}
}

func TestFix_RemovesTrailingWhitespace(t *testing.T) {
	src := []byte("hello   \nworld\t\nfoo  \t \n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	result := r.Fix(f)
	expected := "hello\nworld\nfoo\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestFix_PreservesCleanLines(t *testing.T) {
	src := []byte("hello\nworld\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	result := r.Fix(f)
	if string(result) != string(src) {
		t.Errorf("expected %q, got %q", string(src), string(result))
	}
}

func TestCheck_SkipsFencedCodeBlockLines(t *testing.T) {
	// Trailing spaces inside a fenced code block should NOT fire TM006.
	src := []byte("# Title\n\n```\ncode   \nmore code  \n```\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for trailing spaces inside code block, got %d", len(diags))
	}
}

func TestCheck_TrailingSpacesOutsideCodeBlockStillFlagged(t *testing.T) {
	// Trailing spaces outside code block should still fire.
	src := []byte("hello   \n\n```\ncode   \n```\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Line != 1 {
		t.Errorf("expected diagnostic on line 1, got %d", diags[0].Line)
	}
}

func TestCheck_SkipsIndentedCodeBlockLines(t *testing.T) {
	// Trailing spaces inside an indented code block should NOT fire TM006.
	src := []byte("Some paragraph.\n\n    code   \n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for trailing spaces inside indented code block, got %d", len(diags))
	}
}

func TestFix_PreservesCodeBlockLines(t *testing.T) {
	// Fix should not strip trailing spaces inside code blocks.
	src := []byte("hello   \n\n```\ncode   \n```\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	result := r.Fix(f)
	expected := "hello\n\n```\ncode   \n```\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}
