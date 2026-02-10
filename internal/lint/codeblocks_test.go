package lint

import (
	"testing"
)

func TestCollectCodeBlockLines_FencedCodeBlock(t *testing.T) {
	// Lines:
	// 1: # Heading
	// 2: (blank)
	// 3: ```
	// 4: code line
	// 5: ```
	// 6: (blank)
	src := []byte("# Heading\n\n```\ncode line\n```\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	// Lines 3 (open fence), 4 (content), 5 (close fence) should be in set.
	for _, ln := range []int{3, 4, 5} {
		if !lines[ln] {
			t.Errorf("expected line %d to be in code block lines", ln)
		}
	}
	// Lines 1, 2 should NOT be in set.
	for _, ln := range []int{1, 2} {
		if lines[ln] {
			t.Errorf("expected line %d to NOT be in code block lines", ln)
		}
	}
}

func TestCollectCodeBlockLines_FencedWithInfoString(t *testing.T) {
	src := []byte("# Heading\n\n```go\nfmt.Println()\n```\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{3, 4, 5} {
		if !lines[ln] {
			t.Errorf("expected line %d to be in code block lines", ln)
		}
	}
}

func TestCollectCodeBlockLines_IndentedCodeBlock(t *testing.T) {
	// Indented code block: 4 spaces of indentation, preceded by blank line.
	src := []byte("Some paragraph.\n\n    indented code\n    more code\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{3, 4} {
		if !lines[ln] {
			t.Errorf("expected line %d to be in code block lines", ln)
		}
	}
	// Line 1 should not be in set.
	if lines[1] {
		t.Error("expected line 1 to NOT be in code block lines")
	}
}

func TestCollectCodeBlockLines_NoCodeBlocks(t *testing.T) {
	src := []byte("# Title\n\nJust a paragraph.\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	if len(lines) != 0 {
		t.Errorf("expected empty map for document with no code blocks, got %d entries", len(lines))
	}
}

func TestCollectCodeBlockLines_EmptyFencedCodeBlock(t *testing.T) {
	// An empty fenced code block with no info string: goldmark does not
	// expose the opening fence position, so findFencedOpenLine returns 0.
	// The close fence heuristic also falls through. This is a known
	// limitation that does not affect practical use (the fence lines are
	// short and won't trigger line-length checks).
	src := []byte("```\n```\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	// With no info string and no content, the map will be empty.
	if len(lines) != 0 {
		t.Errorf("expected empty map for empty fenced code block without info string, got %d entries", len(lines))
	}
}

func TestCollectCodeBlockLines_EmptyFencedCodeBlockWithInfo(t *testing.T) {
	// An empty fenced code block WITH an info string: the opening fence
	// can be located via the Info segment.
	src := []byte("```go\n```\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	// Line 1 (open fence) and line 2 (close fence) should be in the set.
	for _, ln := range []int{1, 2} {
		if !lines[ln] {
			t.Errorf("expected line %d to be in code block lines", ln)
		}
	}
}

func TestCollectCodeBlockLines_MultipleFencedCodeBlocks(t *testing.T) {
	src := []byte("```\nfirst\n```\n\n```\nsecond\n```\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	// Lines 1,2,3 (first block) and 5,6,7 (second block).
	for _, ln := range []int{1, 2, 3, 5, 6, 7} {
		if !lines[ln] {
			t.Errorf("expected line %d to be in code block lines", ln)
		}
	}
	// Line 4 (blank between blocks) should NOT be.
	if lines[4] {
		t.Error("expected line 4 to NOT be in code block lines")
	}
}

func TestCollectCodeBlockLines_TildeFence(t *testing.T) {
	src := []byte("~~~\ncode\n~~~\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{1, 2, 3} {
		if !lines[ln] {
			t.Errorf("expected line %d to be in code block lines", ln)
		}
	}
}

func TestCollectCodeBlockLines_FencedWithMultipleContentLines(t *testing.T) {
	src := []byte("```\nline1\nline2\nline3\n```\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{1, 2, 3, 4, 5} {
		if !lines[ln] {
			t.Errorf("expected line %d to be in code block lines", ln)
		}
	}
}

func TestCollectCodeBlockLines_EmptyDocument(t *testing.T) {
	src := []byte("")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	if len(lines) != 0 {
		t.Errorf("expected empty map for empty document, got %d entries", len(lines))
	}
}

func TestCollectCodeBlockLines_TabIndentedLine(t *testing.T) {
	// A tab-indented line at document start is parsed as an indented
	// code block by goldmark (tab equals 4+ spaces indentation).
	src := []byte("\thello\nworld\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	if !lines[1] {
		t.Error("expected line 1 to be in code block lines (tab-indented = indented code block)")
	}
	if lines[2] {
		t.Error("expected line 2 to NOT be in code block lines")
	}
}

func TestCollectCodeBlockLines_FencedWithBlankLinesInside(t *testing.T) {
	// Fenced code block with blank lines inside should mark all lines.
	src := []byte("```\ncode\n\n\nmore code\n```\n")
	f, err := NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{1, 2, 3, 4, 5, 6} {
		if !lines[ln] {
			t.Errorf("expected line %d to be in code block lines", ln)
		}
	}
}
