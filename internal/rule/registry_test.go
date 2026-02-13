package rule

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
)

// stubRule is a minimal Rule implementation for testing.
type stubRule struct {
	id   string
	name string
}

func (r *stubRule) ID() string                           { return r.id }
func (r *stubRule) Name() string                         { return r.name }
func (r *stubRule) Category() string                     { return "test" }
func (r *stubRule) Check(_ *lint.File) []lint.Diagnostic { return nil }

func resetRegistry() {
	registry = nil
}

func TestRegisterAndAll(t *testing.T) {
	resetRegistry()

	r1 := &stubRule{id: "MDS001", name: "line-length"}
	r2 := &stubRule{id: "MDS002", name: "first-heading"}

	Register(r1)
	Register(r2)

	all := All()
	if len(all) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(all))
	}
	if all[0].ID() != "MDS001" {
		t.Errorf("expected first rule ID %q, got %q", "MDS001", all[0].ID())
	}
	if all[1].ID() != "MDS002" {
		t.Errorf("expected second rule ID %q, got %q", "MDS002", all[1].ID())
	}
}

func TestAllReturnsCopy(t *testing.T) {
	resetRegistry()

	Register(&stubRule{id: "MDS001", name: "line-length"})

	all := All()
	all[0] = nil // Mutate the returned slice.

	// The registry should be unaffected.
	original := All()
	if original[0] == nil {
		t.Error("All() should return a copy; mutating the result affected the registry")
	}
}

func TestByID_Found(t *testing.T) {
	resetRegistry()

	r := &stubRule{id: "MDS003", name: "heading-style"}
	Register(r)

	found := ByID("MDS003")
	if found == nil {
		t.Fatal("expected to find rule MDS003")
	}
	if found.ID() != "MDS003" {
		t.Errorf("expected ID %q, got %q", "MDS003", found.ID())
	}
	if found.Name() != "heading-style" {
		t.Errorf("expected Name %q, got %q", "heading-style", found.Name())
	}
}

func TestByID_NotFound(t *testing.T) {
	resetRegistry()

	Register(&stubRule{id: "MDS001", name: "line-length"})

	found := ByID("MDS999")
	if found != nil {
		t.Errorf("expected nil for unknown rule ID, got %v", found)
	}
}
