package frontmattervalidation

import (
	"strings"
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
)

func testFile(
	t *testing.T, path, source string, stripFrontMatter bool,
) *lint.File {
	t.Helper()
	f, err := lint.NewFileFromSource(
		path, []byte(source), stripFrontMatter,
	)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func expectDiagnosticCount(
	t *testing.T, diags []lint.Diagnostic, want int,
) {
	t.Helper()
	if len(diags) != want {
		t.Fatalf("expected %d diagnostics, got %d", want, len(diags))
	}
}

func expectMessageContains(
	t *testing.T, diags []lint.Diagnostic, want string,
) {
	t.Helper()
	for _, d := range diags {
		if strings.Contains(d.Message, want) {
			return
		}
	}
	t.Fatalf("expected a diagnostic containing %q, got %#v", want, diags)
}

func TestRuleMetadata(t *testing.T) {
	r := &Rule{}
	if r.ID() != "MDS027" {
		t.Errorf("ID() = %q, want MDS027", r.ID())
	}
	if r.Name() != "front-matter-validation" {
		t.Errorf(
			"Name() = %q, want front-matter-validation",
			r.Name(),
		)
	}
	if r.Category() != "meta" {
		t.Errorf("Category() = %q, want meta", r.Category())
	}
}

func TestDefaultSettings(t *testing.T) {
	r := &Rule{}
	settings := r.DefaultSettings()
	if settings == nil {
		t.Fatal("DefaultSettings() returned nil")
	}
	if _, ok := settings["required"]; !ok {
		t.Fatal(`DefaultSettings() missing "required"`)
	}
	if _, ok := settings["fields"]; !ok {
		t.Fatal(`DefaultSettings() missing "fields"`)
	}
}

func TestApplySettingsValid(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"required": []any{"id", "status"},
		"fields": map[string]any{
			"id": "string",
			"status": map[string]any{
				"type": "string",
				"enum": []any{"draft", "ready"},
			},
			"priority": map[string]any{
				"type": "int",
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}
	if len(r.Required) != 2 {
		t.Fatalf("len(Required) = %d, want 2", len(r.Required))
	}
	if got := r.Fields["id"].Type; got != "string" {
		t.Errorf(`Fields["id"].Type = %q, want "string"`, got)
	}
	if got := len(r.Fields["status"].Enum); got != 2 {
		t.Errorf(`len(Fields["status"].Enum) = %d, want 2`, got)
	}
}

func TestApplySettingsUnknownSetting(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"nope": true})
	if err == nil {
		t.Fatal("expected error for unknown setting")
	}
}

func TestApplySettingsInvalidRequired(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"required": "id",
	})
	if err == nil {
		t.Fatal("expected error for invalid required")
	}
}

func TestApplySettingsInvalidFieldType(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"fields": map[string]any{
			"id": map[string]any{
				"type": "uuid",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown field type")
	}
}

func TestApplySettingsEnumTypeMismatch(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"fields": map[string]any{
			"id": map[string]any{
				"type": "int",
				"enum": []any{1, "2"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for enum/type mismatch")
	}
}

func TestApplySettingsArrayItemsWithoutArrayType(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"fields": map[string]any{
			"tags": map[string]any{
				"type":  "string",
				"items": "string",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for array settings on non-array type")
	}
}

func TestApplySettingsArrayBoundsInvalid(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"fields": map[string]any{
			"tags": map[string]any{
				"type":      "array",
				"min-items": 3,
				"max-items": 1,
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for min-items > max-items")
	}
}

func TestApplySettingsArrayItemsValid(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"fields": map[string]any{
			"tags": map[string]any{
				"type":      "array",
				"min-items": 1,
				"max-items": 3,
				"items": map[string]any{
					"type": "string",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}
	schema := r.Fields["tags"]
	if schema.Items == nil {
		t.Fatal(`Fields["tags"].Items = nil, want non-nil`)
	}
	if schema.MinItems == nil || *schema.MinItems != 1 {
		t.Fatalf("min-items = %v, want 1", schema.MinItems)
	}
	if schema.MaxItems == nil || *schema.MaxItems != 3 {
		t.Fatalf("max-items = %v, want 3", schema.MaxItems)
	}
	if schema.Items.Type != "string" {
		t.Fatalf("items.type = %q, want string", schema.Items.Type)
	}
}

func TestCheckNoSchemaIsNoop(t *testing.T) {
	r := &Rule{}
	f := testFile(t, "doc.md", "# Title\n", true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 0)
}

func TestCheckMissingRequiredField(t *testing.T) {
	r := &Rule{
		Required: []string{"id", "title"},
	}
	f := testFile(t, "doc.md", "# Title\n", true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 2)
	expectMessageContains(t, diags, `required field "id"`)
	expectMessageContains(t, diags, `required field "title"`)
}

func TestCheckTypeMismatch(t *testing.T) {
	r := &Rule{
		Fields: map[string]FieldSchema{
			"id": {Type: "string"},
		},
	}
	source := "---\nid: 42\n---\n# Title\n"
	f := testFile(t, "doc.md", source, true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 1)
	expectMessageContains(t, diags, `field "id"`)
	expectMessageContains(t, diags, `must be string`)
}

func TestCheckEnumMismatch(t *testing.T) {
	r := &Rule{
		Fields: map[string]FieldSchema{
			"status": {
				Type: "string",
				Enum: []any{"draft", "ready"},
			},
		},
	}
	source := "---\nstatus: done\n---\n# Title\n"
	f := testFile(t, "doc.md", source, true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 1)
	expectMessageContains(t, diags, `field "status"`)
	expectMessageContains(t, diags, `invalid value "done"`)
}

func TestCheckValidFrontMatter(t *testing.T) {
	r := &Rule{
		Required: []string{"id", "status"},
		Fields: map[string]FieldSchema{
			"id": {
				Type: "string",
			},
			"status": {
				Type: "string",
				Enum: []any{"draft", "ready"},
			},
		},
	}
	source := "---\nid: MDS027\nstatus: ready\n---\n# Title\n"
	f := testFile(t, "doc.md", source, true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 0)
}

func TestCheckInvalidYAML(t *testing.T) {
	r := &Rule{
		Required: []string{"id"},
	}
	source := "---\nid: [\n---\n# Title\n"
	f := testFile(t, "doc.md", source, true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 1)
	expectMessageContains(t, diags, "not valid YAML")
}

func TestCheckParsesFrontMatterWhenNotStripped(t *testing.T) {
	r := &Rule{
		Required: []string{"id"},
	}
	source := "---\nid: MDS027\n---\n# Title\n"
	f, err := lint.NewFile("doc.md", []byte(source))
	if err != nil {
		t.Fatal(err)
	}

	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 0)
}

func TestCheckDiagnosticIncludesPathAndField(t *testing.T) {
	r := &Rule{
		Required: []string{"id"},
	}
	f := testFile(t, "plans/doc.md", "# Title\n", true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 1)
	if diags[0].File != "plans/doc.md" {
		t.Fatalf("diag file = %q, want plans/doc.md", diags[0].File)
	}
	if !strings.Contains(diags[0].Message, `"id"`) {
		t.Fatalf(
			"diag message missing field name: %q",
			diags[0].Message,
		)
	}
}

func TestCheckArrayItemTypeMismatch(t *testing.T) {
	r := &Rule{
		Fields: map[string]FieldSchema{
			"tags": {
				Type: "array",
				Items: &FieldSchema{
					Type: "string",
				},
			},
		},
	}
	source := "---\ntags:\n  - alpha\n  - 2\n---\n# Title\n"
	f := testFile(t, "doc.md", source, true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 1)
	expectMessageContains(t, diags, `field "tags[1]"`)
	expectMessageContains(t, diags, `must be string`)
}

func TestCheckArrayMinItems(t *testing.T) {
	min := 2
	r := &Rule{
		Fields: map[string]FieldSchema{
			"tags": {
				Type:     "array",
				MinItems: &min,
			},
		},
	}
	source := "---\ntags:\n  - alpha\n---\n# Title\n"
	f := testFile(t, "doc.md", source, true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 1)
	expectMessageContains(t, diags, `field "tags"`)
	expectMessageContains(t, diags, "at least 2 items")
}

func TestCheckArrayMaxItems(t *testing.T) {
	max := 1
	r := &Rule{
		Fields: map[string]FieldSchema{
			"tags": {
				Type:     "array",
				MaxItems: &max,
			},
		},
	}
	source := "---\ntags:\n  - alpha\n  - beta\n---\n# Title\n"
	f := testFile(t, "doc.md", source, true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 1)
	expectMessageContains(t, diags, `field "tags"`)
	expectMessageContains(t, diags, "at most 1 items")
}

func TestCheckArrayItemsValid(t *testing.T) {
	r := &Rule{
		Fields: map[string]FieldSchema{
			"tags": {
				Type: "array",
				Items: &FieldSchema{
					Type: "string",
				},
			},
		},
	}
	source := "---\ntags:\n  - alpha\n  - beta\n---\n# Title\n"
	f := testFile(t, "doc.md", source, true)
	diags := r.Check(f)
	expectDiagnosticCount(t, diags, 0)
}
