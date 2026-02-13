package include

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jeduden/tidymark/internal/lint"
)

func newTestFile(
	t *testing.T, path, source string, fs ...fstest.MapFS,
) *lint.File {
	t.Helper()
	f, err := lint.NewFile(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) > 0 {
		f.FS = fs[0]
	}
	return f
}

func expectDiags(
	t *testing.T, diags []lint.Diagnostic, count int,
) {
	t.Helper()
	if len(diags) != count {
		msgs := make([]string, len(diags))
		for i, d := range diags {
			msgs[i] = d.Message
		}
		t.Fatalf(
			"expected %d diagnostic(s), got %d: %v",
			count, len(diags), msgs,
		)
	}
}

func expectDiagMsg(
	t *testing.T, diags []lint.Diagnostic, msg string,
) {
	t.Helper()
	if len(diags) == 0 {
		t.Fatalf(
			"expected diagnostic with message %q, got none", msg,
		)
	}
	for _, d := range diags {
		if strings.Contains(d.Message, msg) {
			return
		}
	}
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	t.Errorf(
		"no diagnostic contains %q, got: %v", msg, msgs,
	)
}

// =====================================================================
// Rule metadata
// =====================================================================

func TestRule_ID(t *testing.T) {
	r := &Rule{}
	if r.ID() != "TM021" {
		t.Errorf("expected ID TM021, got %s", r.ID())
	}
}

func TestRule_Name(t *testing.T) {
	r := &Rule{}
	if r.Name() != "include" {
		t.Errorf("expected Name include, got %s", r.Name())
	}
}

// =====================================================================
// Basic include
// =====================================================================

func TestCheck_IncludeUpToDate(t *testing.T) {
	fsys := fstest.MapFS{
		"data.md": {Data: []byte("Hello world\n")},
	}
	src := "# Doc\n\n<!-- include\nfile: data.md\n-->\nHello world\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiags(t, diags, 0)
}

func TestCheck_IncludeOutOfDate(t *testing.T) {
	fsys := fstest.MapFS{
		"data.md": {Data: []byte("Updated content\n")},
	}
	src := "# Doc\n\n<!-- include\nfile: data.md\n-->\nOld content\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiagMsg(t, diags, "generated section is out of date")
}

// =====================================================================
// Strip frontmatter
// =====================================================================

func TestCheck_StripFrontmatterDefault(t *testing.T) {
	fsys := fstest.MapFS{
		"data.md": {
			Data: []byte(
				"---\nid: TM001\n---\nHello world\n",
			),
		},
	}
	src := "# Doc\n\n<!-- include\nfile: data.md\n-->\nHello world\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiags(t, diags, 0)
}

func TestCheck_StripFrontmatterOff(t *testing.T) {
	fsys := fstest.MapFS{
		"data.md": {
			Data: []byte(
				"---\nid: TM001\n---\nHello world\n",
			),
		},
	}
	src := "# Doc\n\n<!-- include\nfile: data.md\n" +
		"strip-frontmatter: \"false\"\n-->\n" +
		"---\nid: TM001\n---\nHello world\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiags(t, diags, 0)
}

// =====================================================================
// Wrap in code fence
// =====================================================================

func TestCheck_WrapInCodeFence(t *testing.T) {
	fsys := fstest.MapFS{
		"example.md": {Data: []byte("# Hello\n")},
	}
	src := "# Doc\n\n<!-- include\nfile: example.md\n" +
		"wrap: markdown\n-->\n```markdown\n# Hello\n```\n" +
		"<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiags(t, diags, 0)
}

// =====================================================================
// Missing file
// =====================================================================

func TestCheck_MissingFile(t *testing.T) {
	fsys := fstest.MapFS{}
	src := "# Doc\n\n<!-- include\nfile: missing.md\n-->\nold\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiagMsg(t, diags, "not found")
}

// =====================================================================
// Validation errors
// =====================================================================

func TestCheck_MissingFileParam(t *testing.T) {
	fsys := fstest.MapFS{}
	src := "# Doc\n\n<!-- include\n-->\nold\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiagMsg(t, diags, `missing required "file" parameter`)
}

func TestCheck_AbsoluteFilePath(t *testing.T) {
	fsys := fstest.MapFS{}
	src := "# Doc\n\n<!-- include\nfile: /etc/passwd\n-->\nold\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiagMsg(t, diags, "absolute file path")
}

func TestCheck_DotDotTraversal(t *testing.T) {
	fsys := fstest.MapFS{}
	src := "# Doc\n\n<!-- include\nfile: ../secret.md\n-->\nold\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	diags := r.Check(f)
	expectDiagMsg(t, diags, `".." traversal`)
}

// =====================================================================
// Fix
// =====================================================================

func TestFix_UpdatesContent(t *testing.T) {
	fsys := fstest.MapFS{
		"data.md": {Data: []byte("New content\n")},
	}
	src := "# Doc\n\n<!-- include\nfile: data.md\n-->\nOld content\n<!-- /include -->\n"
	f := newTestFile(t, "doc.md", src, fsys)
	r := &Rule{}
	got := string(r.Fix(f))
	want := "# Doc\n\n<!-- include\nfile: data.md\n-->\nNew content\n<!-- /include -->\n"
	if got != want {
		t.Errorf(
			"Fix output mismatch\ngot:\n%s\nwant:\n%s", got, want,
		)
	}
}

// =====================================================================
// No FS
// =====================================================================

func TestCheck_NoFS(t *testing.T) {
	f, err := lint.NewFile("test.md", []byte("# Hello\n"))
	if err != nil {
		t.Fatal(err)
	}
	r := &Rule{}
	diags := r.Check(f)
	expectDiags(t, diags, 0)
}
