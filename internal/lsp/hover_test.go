package lsp

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Register the production rule set so rule.All() returns what an
	// editor would actually see.
	_ "github.com/jeduden/mdsmith/internal/rules/all"
)

// findRepoRoot returns the repository root via `git rev-parse`.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}

// TestHoverProviderAdvertised verifies that hoverProvider is included
// in the initialize capabilities response.
func TestHoverProviderAdvertised(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	resultRaw, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	var res initializeResult
	require.NoError(t, json.Unmarshal(resultRaw, &res))
	assert.True(t, res.Capabilities.HoverProvider, "hoverProvider should be true in initialize response")
}

// TestHoverOnDiagnosticReturnsRuleDocs verifies that hovering over a
// diagnostic squiggle returns rule documentation. Uses MDS006
// (no-trailing-spaces) which fires reliably on trailing-space input.
func TestHoverOnDiagnosticReturnsRuleDocs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/hover-diag.md"
	// trailing spaces trigger MDS006
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{
			URI:        uri,
			LanguageID: "markdown",
			Version:    1,
			Text:       "# Title\n\ntrailing spaces   \n",
		},
	})

	// Wait for the diagnostic to be published.
	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var pub publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &pub))

	// Find the MDS006 diagnostic.
	var diag *Diagnostic
	for i := range pub.Diagnostics {
		if pub.Diagnostics[i].Code == "MDS006" {
			diag = &pub.Diagnostics[i]
			break
		}
	}
	require.NotNil(t, diag, "expected MDS006 diagnostic, got %+v", pub.Diagnostics)

	// Hover at the start of the MDS006 range.
	hoverRaw, errResp := h.request("textDocument/hover", hoverParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     diag.Range.Start,
	})
	require.Nil(t, errResp)
	require.NotNil(t, hoverRaw)
	require.NotEqual(t, "null", string(hoverRaw), "expected non-null hover result over diagnostic")

	var result hoverResult
	require.NoError(t, json.Unmarshal(hoverRaw, &result))
	assert.Equal(t, "markdown", result.Contents.Kind)
	// The body should mention MDS006 and the rule description.
	assert.Contains(t, result.Contents.Value, "MDS006")
	// The rule docs body should contain content from the rule README.
	assert.Contains(t, result.Contents.Value, "trailing")
	require.NotNil(t, result.Range)
}

// TestHoverOnDirectiveReturnsDirectiveDocs verifies that hovering
// inside a <?catalog?> block returns catalog docs when no diagnostic
// covers the cursor.
func TestHoverOnDirectiveReturnsDirectiveDocs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	// Initialize with the repo root as the workspace folder so the server
	// can find the directive guide docs under docs/guides/directives/.
	repoRoot := findRepoRoot(t)
	_, errResp := h.request("initialize", initializeParams{
		WorkspaceFolders: []workspaceFolder{{URI: "file://" + repoRoot}},
	})
	require.Nil(t, errResp)

	uri := "file://" + repoRoot + "/hover-directive.md"
	// A catalog directive with no violations on the line itself.
	text := "# Title\n\n<?catalog\nglob: \"*.md\"\n?>\n\nsome content\n"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{
			URI:        uri,
			LanguageID: "markdown",
			Version:    1,
			Text:       text,
		},
	})

	// Wait for initial diagnostics publish so the doc is fully loaded.
	h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)

	// Hover on the `glob: "*.md"` line (line 3, col 0 in the directive body).
	hoverRaw, errResp := h.request("textDocument/hover", hoverParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: 3, Character: 0},
	})
	require.Nil(t, errResp)
	require.NotNil(t, hoverRaw)
	require.NotEqual(t, "null", string(hoverRaw), "expected non-null hover result inside directive block")

	var result hoverResult
	require.NoError(t, json.Unmarshal(hoverRaw, &result))
	assert.Equal(t, "markdown", result.Contents.Kind)
	// The directive docs body should mention catalog.
	assert.Contains(t, result.Contents.Value, "catalog")
	require.NotNil(t, result.Range)
}

// TestHoverOnProseReturnsNull verifies that hovering on plain prose
// with no diagnostic or directive returns null.
func TestHoverOnProseReturnsNull(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/hover-prose.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{
			URI:        uri,
			LanguageID: "markdown",
			Version:    1,
			Text:       "# Title\n\nPlain prose text here.\n",
		},
	})

	// Wait for initial diagnostics.
	h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)

	// Hover on the plain prose line.
	hoverRaw, errResp := h.request("textDocument/hover", hoverParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Position:     Position{Line: 2, Character: 5},
	})
	require.Nil(t, errResp)
	// null is the expected JSON for no hover.
	assert.Equal(t, "null", string(hoverRaw), "expected null hover result on plain prose")
}

// TestHoverOnUnknownURI verifies that hovering on an URI that has no
// open document returns null without crashing.
func TestHoverOnUnknownURI(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	hoverRaw, errResp := h.request("textDocument/hover", hoverParams{
		TextDocument: textDocumentIdentifier{URI: "file:///workspace/notopen.md"},
		Position:     Position{Line: 0, Character: 0},
	})
	require.Nil(t, errResp)
	assert.Equal(t, "null", string(hoverRaw))
}

// --- Unit tests for hover helpers ---

func TestPosInRange(t *testing.T) {
	t.Parallel()
	r := Range{
		Start: Position{Line: 2, Character: 5},
		End:   Position{Line: 2, Character: 20},
	}
	// Inside
	assert.True(t, posInRange(Position{Line: 2, Character: 10}, r))
	// At start (inclusive)
	assert.True(t, posInRange(Position{Line: 2, Character: 5}, r))
	// At end (exclusive)
	assert.False(t, posInRange(Position{Line: 2, Character: 20}, r))
	// Before start character
	assert.False(t, posInRange(Position{Line: 2, Character: 4}, r))
	// Wrong line
	assert.False(t, posInRange(Position{Line: 1, Character: 10}, r))
	assert.False(t, posInRange(Position{Line: 3, Character: 10}, r))
}

func TestPosInRangeMultiLine(t *testing.T) {
	t.Parallel()
	r := Range{
		Start: Position{Line: 1, Character: 3},
		End:   Position{Line: 3, Character: 7},
	}
	// On start line, before character
	assert.False(t, posInRange(Position{Line: 1, Character: 2}, r))
	// On start line, at character
	assert.True(t, posInRange(Position{Line: 1, Character: 3}, r))
	// Middle line
	assert.True(t, posInRange(Position{Line: 2, Character: 0}, r))
	// On end line, before end char
	assert.True(t, posInRange(Position{Line: 3, Character: 6}, r))
	// On end line, at end char (exclusive)
	assert.False(t, posInRange(Position{Line: 3, Character: 7}, r))
}

func TestFindDirectiveAtPos(t *testing.T) {
	t.Parallel()

	// Multi-line directive
	source := []byte("# Title\n\n<?catalog\nglob: \"*.md\"\n?>\n\nprose\n")
	// Line 2 = "<?catalog", line 3 = "glob: ...", line 4 = "?>"

	block, ok := findDirectiveAtPos(Position{Line: 2, Character: 5}, source)
	assert.True(t, ok)
	assert.Equal(t, "catalog", block.name)
	assert.Equal(t, 2, block.blockRange.Start.Line)
	assert.Equal(t, 4, block.blockRange.End.Line)

	// Inside the body
	block, ok = findDirectiveAtPos(Position{Line: 3, Character: 0}, source)
	assert.True(t, ok)
	assert.Equal(t, "catalog", block.name)

	// On the closing line
	block, ok = findDirectiveAtPos(Position{Line: 4, Character: 0}, source)
	assert.True(t, ok)
	assert.Equal(t, "catalog", block.name)

	// Before the directive
	_, ok = findDirectiveAtPos(Position{Line: 0, Character: 0}, source)
	assert.False(t, ok)

	// After the directive (prose)
	_, ok = findDirectiveAtPos(Position{Line: 6, Character: 0}, source)
	assert.False(t, ok)
}

func TestFindDirectiveAtPosSingleLine(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\n\n<?catalog glob: \"*.md\"?>\n\nprose\n")
	// Line 2 = "<?catalog glob: \"*.md\"?>"

	block, ok := findDirectiveAtPos(Position{Line: 2, Character: 5}, source)
	assert.True(t, ok)
	assert.Equal(t, "catalog", block.name)
	assert.Equal(t, 2, block.blockRange.Start.Line)
	assert.Equal(t, 2, block.blockRange.End.Line)

	// Before
	_, ok = findDirectiveAtPos(Position{Line: 1, Character: 0}, source)
	assert.False(t, ok)

	// After
	_, ok = findDirectiveAtPos(Position{Line: 4, Character: 0}, source)
	assert.False(t, ok)
}

func TestFindDirectiveAtPosClosureDirective(t *testing.T) {
	t.Parallel()

	// Closure directive (<?/catalog?>) should not match as an open block.
	source := []byte("<?catalog\nglob: \"*.md\"\n?>\n\n<?/catalog?>\n")
	// Line 4 = "<?/catalog?>"

	_, ok := findDirectiveAtPos(Position{Line: 4, Character: 3}, source)
	assert.False(t, ok, "closure directive should not match")
}

func TestExtractPIName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"catalog\n", "catalog"},
		{"catalog glob: \"*.md\"\n", "catalog"},
		{"catalog?>", "catalog"},
		{"include\nfile: x.md\n", "include"},
		{"", ""},
		{"?>\n", ""},
	}
	for _, tt := range tests {
		got := extractPIName(tt.input)
		assert.Equal(t, tt.want, got, "input: %q", tt.input)
	}
}

func TestHoverForDiagnosticKnownRule(t *testing.T) {
	t.Parallel()
	d := Diagnostic{
		Code:    "MDS001",
		Message: "line too long (200 > 100)",
		Range:   Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 10}},
	}
	result := hoverForDiagnostic(d)
	require.NotNil(t, result)
	assert.Equal(t, "markdown", result.Contents.Kind)
	assert.Contains(t, result.Contents.Value, "MDS001")
	assert.Contains(t, result.Contents.Value, "line too long (200 > 100)")
	// The rule doc body should contain content from the rule README.
	assert.Contains(t, result.Contents.Value, "line")
	require.NotNil(t, result.Range)
}

func TestHoverForDiagnosticUnknownRule(t *testing.T) {
	t.Parallel()
	d := Diagnostic{
		Code:    "MDS999",
		Message: "some message",
		Range:   Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 1, Character: 5}},
	}
	result := hoverForDiagnostic(d)
	require.NotNil(t, result)
	assert.Equal(t, "markdown", result.Contents.Kind)
	assert.Contains(t, result.Contents.Value, "MDS999")
	// Falls back to generic help text.
	assert.Contains(t, result.Contents.Value, "mdsmith help rule")
}

func TestHoverForDiagnosticEmptyCode(t *testing.T) {
	t.Parallel()
	d := Diagnostic{
		Code:    "",
		Message: "no code",
	}
	result := hoverForDiagnostic(d)
	assert.Nil(t, result, "should return nil for empty code")
}
