package main_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLSPHoverE2E spawns the shared mdsmith binary, drives a
// textDocument/hover round-trip, and asserts the plan-133 acceptance
// criteria over a real LSP pipe.
//
// Three sub-cases are exercised:
//  1. Hover over an MDS001 diagnostic → returns MarkupContent with
//     line-length rule docs.
//  2. Hover inside a <?catalog?> directive (no diagnostic at cursor) →
//     returns catalog directive guide docs.
//  3. Hover on plain prose (no diagnostic, no directive) → returns null.
func TestLSPHoverE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping LSP hover subprocess test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tmp, longLineSrc, catalogSrc := writeHoverCorpus(t)
	pipe := startLSPSubprocess(t, ctx, binaryPath)
	rootURI := pathToFileURIE2E(t, tmp)
	initHoverE2E(t, pipe, rootURI)

	uriLong := rootURI + "/longline.md"
	uriCatalog := rootURI + "/catalog.md"

	pipe.openDocument(uriLong, longLineSrc)
	diags := pipe.awaitDiagnostics(t, uriLong, time.Now().Add(20*time.Second))
	require.NotNil(t, findDiag(diags, "MDS001"),
		"expected MDS001 diagnostic in longline.md, got %+v", diags.Diagnostics)

	t.Run("HoverOnDiagnostic", func(t *testing.T) {
		assertHoverOnDiagnostic(t, pipe, uriLong)
	})

	pipe.openDocument(uriCatalog, catalogSrc)
	_ = pipe.awaitDiagnostics(t, uriCatalog, time.Now().Add(20*time.Second))

	t.Run("HoverOnDirective", func(t *testing.T) {
		assertHoverOnDirective(t, pipe, uriCatalog)
	})

	t.Run("HoverOnProse", func(t *testing.T) {
		result := hoverRequest(t, pipe, uriCatalog, 6, 0)
		assert.Nil(t, result, "expected null hover on plain prose")
	})

	pipe.shutdown(t)
}

// writeHoverCorpus creates the two test Markdown files in a temp dir
// and returns the dir path plus the file contents.
func writeHoverCorpus(t *testing.T) (tmp, longLineSrc, catalogSrc string) {
	t.Helper()
	tmp = t.TempDir()
	longLine := strings.Repeat("x", 120)
	longLineSrc = "# Title\n\n" + longLine + "\n"
	catalogSrc = "# Index\n\n<?catalog\nglob: \"*.md\"\n?>\n\nPlain prose here.\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "longline.md"), []byte(longLineSrc), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "catalog.md"), []byte(catalogSrc), 0o644))
	return tmp, longLineSrc, catalogSrc
}

// findDiag returns a pointer to the first diagnostic with the given
// code, or nil if none is found.
func findDiag(diags publishDiagnosticsParams, code string) *publishedDiagItem {
	for i := range diags.Diagnostics {
		if diags.Diagnostics[i].Code == code {
			return &diags.Diagnostics[i]
		}
	}
	return nil
}

// assertHoverOnDiagnostic asserts that hovering within the MDS001
// diagnostic range returns MarkupContent containing rule docs.
//
// MDS001 fires at column 101 (default 100-char limit). Hover at
// char 110 which is within the range [100, 120].
func assertHoverOnDiagnostic(t *testing.T, pipe *lspPipe, uri string) {
	t.Helper()
	result := hoverRequest(t, pipe, uri, 2, 110)
	require.NotNil(t, result, "expected non-null hover on MDS001 diagnostic")
	assert.Equal(t, "markdown", result["kind"])
	value, _ := result["value"].(string)
	assert.True(t, strings.Contains(value, "MDS001"),
		"hover body should mention MDS001, got: %s", value)
	assert.True(t, strings.Contains(strings.ToLower(value), "line"),
		"hover body should contain line-length docs, got: %s", value)
}

// assertHoverOnDirective asserts that hovering inside a catalog
// directive returns docs mentioning "catalog".
func assertHoverOnDirective(t *testing.T, pipe *lspPipe, uri string) {
	t.Helper()
	// Line 3 = "glob: \"*.md\"", inside the catalog directive body.
	result := hoverRequest(t, pipe, uri, 3, 0)
	require.NotNil(t, result, "expected non-null hover inside catalog directive")
	assert.Equal(t, "markdown", result["kind"])
	value, _ := result["value"].(string)
	assert.True(t, strings.Contains(strings.ToLower(value), "catalog"),
		"hover body should mention catalog, got: %s", value)
}

// initHoverE2E sends the initialize handshake and asserts the server
// advertises hoverProvider.
func initHoverE2E(t *testing.T, pipe *lspPipe, rootURI string) {
	t.Helper()
	resp := pipe.request("initialize", 1, map[string]any{
		"rootUri":      rootURI,
		"capabilities": fullClientCapabilities(),
	})
	require.Equal(t, float64(1), resp["id"])
	res, ok := resp["result"].(map[string]any)
	require.True(t, ok)
	caps, ok := res["capabilities"].(map[string]any)
	require.True(t, ok)
	hoverProv, _ := caps["hoverProvider"].(bool)
	assert.True(t, hoverProv, "expected hoverProvider in initialize capabilities")
	pipe.notify("initialized", map[string]any{})
}

// hoverRequest sends a textDocument/hover request and returns the
// `contents` field of the result, or nil if the result is JSON null.
func hoverRequest(t *testing.T, pipe *lspPipe, uri string, line, char int) map[string]any {
	t.Helper()
	idNum := 200
	pipe.writeFrame(map[string]any{
		"jsonrpc": "2.0",
		"id":      idNum,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": line, "character": char},
		},
	})
	return awaitHoverResponse(t, pipe, idNum)
}

// awaitHoverResponse reads frames until it receives a response for
// idNum, auto-handling server-initiated requests along the way.
func awaitHoverResponse(t *testing.T, pipe *lspPipe, idNum int) map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		m := pipe.readFrame()
		id, _ := m["id"].(float64)
		method, _ := m["method"].(string)
		if method == "workspace/configuration" {
			pipe.writeFrame(map[string]any{
				"jsonrpc": "2.0", "id": m["id"],
				"result": []map[string]any{{"run": "onSave"}},
			})
			continue
		}
		if method == "client/registerCapability" {
			pipe.writeFrame(map[string]any{"jsonrpc": "2.0", "id": m["id"], "result": nil})
			continue
		}
		if method != "" {
			continue
		}
		if id != float64(idNum) {
			continue
		}
		result := m["result"]
		if result == nil {
			return nil
		}
		if resultMap, ok := result.(map[string]any); ok {
			if contents, ok := resultMap["contents"].(map[string]any); ok {
				return contents
			}
		}
		return nil
	}
	t.Fatalf("timed out waiting for hover response")
	return nil
}
