package query

import "testing"

func TestMatch_MatchingField(t *testing.T) {
	fm := map[string]any{"status": "✅", "id": 42}
	got, err := Match(`status: "✅"`, fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected match")
	}
}

func TestMatch_NonMatchingField(t *testing.T) {
	fm := map[string]any{"status": "🔲", "id": 42}
	got, err := Match(`status: "✅"`, fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected no match")
	}
}

func TestMatch_MissingField(t *testing.T) {
	fm := map[string]any{"id": 42}
	got, err := Match(`status: "✅"`, fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected no match for missing field")
	}
}

func TestMatch_NilFrontMatter(t *testing.T) {
	got, err := Match(`status: "✅"`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected no match for nil front matter")
	}
}

func TestMatch_SchemaStringProto(t *testing.T) {
	// Proto/template files have CUE schema strings as values,
	// not concrete values. These should not match.
	fm := map[string]any{"status": `"🔲" | "🔳" | "✅"`}
	got, err := Match(`status: "✅"`, fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected no match for schema-string front matter")
	}
}

func TestMatch_CompoundExpression(t *testing.T) {
	fm := map[string]any{"status": "✅", "id": 60}
	got, err := Match(`status: "✅", id: >50`, fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected match for compound expression")
	}
}

func TestMatch_CompoundExpression_PartialFail(t *testing.T) {
	fm := map[string]any{"status": "✅", "id": 30}
	got, err := Match(`status: "✅", id: >50`, fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected no match when id constraint fails")
	}
}

func TestMatch_InvalidCUE(t *testing.T) {
	fm := map[string]any{"status": "✅"}
	_, err := Match(`status: [[[`, fm)
	if err == nil {
		t.Fatal("expected error for invalid CUE expression")
	}
}

func TestCompile_Valid(t *testing.T) {
	m, err := Compile(`status: "✅"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := m.Match(map[string]any{"status": "✅"})
	if !got {
		t.Fatal("expected match")
	}
}

func TestCompile_Invalid(t *testing.T) {
	_, err := Compile(`status: [[[`)
	if err == nil {
		t.Fatal("expected error for invalid CUE expression")
	}
}
