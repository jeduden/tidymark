package unclosedcodeblock

import (
	"strings"
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
)

func TestCheck_UnclosedBacktick(t *testing.T) {
	src := []byte("# Doc\n\n```go\nfmt.Println(\"hello\")\n")
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
	if d.RuleID != "MDS031" {
		t.Errorf("expected MDS031, got %s", d.RuleID)
	}
	if d.Line != 3 {
		t.Errorf("expected line 3, got %d", d.Line)
	}
	if d.Severity != lint.Error {
		t.Errorf("expected error severity, got %s", d.Severity)
	}
}

func TestCheck_UnclosedTilde(t *testing.T) {
	src := []byte("# Doc\n\n~~~\ncode\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Line != 3 {
		t.Errorf("expected line 3, got %d", diags[0].Line)
	}
}

func TestCheck_ClosedBlock_NoDiagnostic(t *testing.T) {
	src := []byte("# Doc\n\n```go\nfmt.Println(\"hello\")\n```\n")
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

func TestCheck_ClosedTilde_NoDiagnostic(t *testing.T) {
	src := []byte("# Doc\n\n~~~\ncode\n~~~\n")
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

func TestCheck_EmptyUnclosed(t *testing.T) {
	src := []byte("# Doc\n\n```go\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
}

func TestCheck_MultipleBlocks_SecondUnclosed(t *testing.T) {
	src := []byte("```\na\n```\n\n```\nb\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Line != 5 {
		t.Errorf("expected line 5, got %d", diags[0].Line)
	}
}

func TestCheck_MismatchedFenceChar(t *testing.T) {
	// Tilde fence cannot be closed by backtick fence
	src := []byte("~~~\ncode\n```\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
}

func TestCheck_LongerClosingFence(t *testing.T) {
	// Closing fence can be longer than opening
	src := []byte("```\ncode\n`````\n")
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

func TestCheck_ShorterClosingFence_Unclosed(t *testing.T) {
	// Closing fence shorter than opening does not close
	src := []byte("`````\ncode\n```\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
}

func TestCheck_IndentedFence(t *testing.T) {
	// Up to 3 spaces of indentation is valid
	src := []byte("   ```\ncode\n   ```\n")
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

func TestCheck_MessageContainsLine(t *testing.T) {
	src := []byte("# Doc\n\n```go\ncode\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{}
	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if !strings.Contains(diags[0].Message, "line 3") {
		t.Errorf("expected message to contain line number, got: %s", diags[0].Message)
	}
}

func TestID(t *testing.T) {
	r := &Rule{}
	if r.ID() != "MDS031" {
		t.Errorf("expected MDS031, got %s", r.ID())
	}
}

func TestName(t *testing.T) {
	r := &Rule{}
	if r.Name() != "unclosed-code-block" {
		t.Errorf("expected unclosed-code-block, got %s", r.Name())
	}
}

func TestCategory(t *testing.T) {
	r := &Rule{}
	if r.Category() != "code" {
		t.Errorf("expected code, got %s", r.Category())
	}
}
