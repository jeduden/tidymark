package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectCodeBlockLines_FencedCodeBlock(t *testing.T) {
	src := []byte("# Heading\n\n```\ncode line\n```\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{3, 4, 5} {
		assert.True(t, lines[ln], "expected line %d to be in code block lines", ln)
	}
	for _, ln := range []int{1, 2} {
		assert.False(t, lines[ln], "expected line %d to NOT be in code block lines", ln)
	}
}

func TestCollectCodeBlockLines_FencedWithInfoString(t *testing.T) {
	src := []byte("# Heading\n\n```go\nfmt.Println()\n```\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{3, 4, 5} {
		assert.True(t, lines[ln], "expected line %d to be in code block lines", ln)
	}
}

func TestCollectCodeBlockLines_IndentedCodeBlock(t *testing.T) {
	src := []byte("Some paragraph.\n\n    indented code\n    more code\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{3, 4} {
		assert.True(t, lines[ln], "expected line %d to be in code block lines", ln)
	}
	assert.False(t, lines[1], "expected line 1 to NOT be in code block lines")
}

func TestCollectCodeBlockLines_NoCodeBlocks(t *testing.T) {
	src := []byte("# Title\n\nJust a paragraph.\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	assert.Empty(t, lines, "expected empty map for document with no code blocks")
}

func TestCollectCodeBlockLines_EmptyFencedCodeBlock(t *testing.T) {
	src := []byte("```\n```\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	assert.Empty(t, lines, "expected empty map for empty fenced code block without info string")
}

func TestCollectCodeBlockLines_EmptyFencedCodeBlockWithInfo(t *testing.T) {
	src := []byte("```go\n```\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{1, 2} {
		assert.True(t, lines[ln], "expected line %d to be in code block lines", ln)
	}
}

func TestCollectCodeBlockLines_MultipleFencedCodeBlocks(t *testing.T) {
	src := []byte("```\nfirst\n```\n\n```\nsecond\n```\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{1, 2, 3, 5, 6, 7} {
		assert.True(t, lines[ln], "expected line %d to be in code block lines", ln)
	}
	assert.False(t, lines[4], "expected line 4 to NOT be in code block lines")
}

func TestCollectCodeBlockLines_TildeFence(t *testing.T) {
	src := []byte("~~~\ncode\n~~~\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{1, 2, 3} {
		assert.True(t, lines[ln], "expected line %d to be in code block lines", ln)
	}
}

func TestCollectCodeBlockLines_FencedWithMultipleContentLines(t *testing.T) {
	src := []byte("```\nline1\nline2\nline3\n```\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{1, 2, 3, 4, 5} {
		assert.True(t, lines[ln], "expected line %d to be in code block lines", ln)
	}
}

func TestCollectCodeBlockLines_EmptyDocument(t *testing.T) {
	src := []byte("")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	assert.Empty(t, lines, "expected empty map for empty document")
}

func TestCollectCodeBlockLines_TabIndentedLine(t *testing.T) {
	src := []byte("\thello\nworld\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	assert.True(t, lines[1], "expected line 1 to be in code block lines (tab-indented = indented code block)")
	assert.False(t, lines[2], "expected line 2 to NOT be in code block lines")
}

func TestCollectCodeBlockLines_FencedWithBlankLinesInside(t *testing.T) {
	src := []byte("```\ncode\n\n\nmore code\n```\n")
	f, err := NewFile("test.md", src)
	require.NoError(t, err)
	lines := CollectCodeBlockLines(f)
	for _, ln := range []int{1, 2, 3, 4, 5, 6} {
		assert.True(t, lines[ln], "expected line %d to be in code block lines", ln)
	}
}
