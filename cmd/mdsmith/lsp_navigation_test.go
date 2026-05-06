package main_test

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLSPNavigationE2E spawns the shared mdsmith binary, drives a
// full symbol-navigation round-trip
// (initialize → didOpen → documentSymbol → definition → references
// → prepareCallHierarchy → incomingCalls → shutdown → exit) over
// stdio, and asserts the headline plan-131 acceptance criteria
// against a real workspace.
func TestLSPNavigationE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping LSP navigation subprocess test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tmp, srcA, srcB := writeNavigationCorpus(t)
	pipe := startLSPSubprocess(t, ctx, binaryPath)
	rootURI := pathToFileURIE2E(t, tmp)
	uriA, uriB := rootURI+"/a.md", rootURI+"/b.md"

	initE2ENavigation(t, pipe, rootURI)
	pipe.openDocument(uriA, srcA)
	_ = pipe.awaitDiagnostics(t, uriA, time.Now().Add(15*time.Second))
	pipe.openDocument(uriB, srcB)
	_ = pipe.awaitDiagnostics(t, uriB, time.Now().Add(15*time.Second))

	assertDocumentSymbolOutline(t, pipe, uriA)
	assertDefinitionJumpsToHeading(t, pipe, uriA, uriB)
	assertReferencesFromB(t, pipe, uriA, uriB)
	assertCallHierarchyIncoming(t, pipe, uriA)

	pipe.shutdown(t)
}

// writeNavigationCorpus writes a two-file workspace where a.md is
// the navigation target and b.md links into it.
func writeNavigationCorpus(t *testing.T) (root, srcA, srcB string) {
	t.Helper()
	root = t.TempDir()
	srcA = "# Alpha\n\n## Inner\n\nbody\n"
	srcB = "# Beta\n\n[link](./a.md#inner)\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.md"), []byte(srcA), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.md"), []byte(srcB), 0o644))
	return root, srcA, srcB
}

func initE2ENavigation(t *testing.T, pipe *lspPipe, rootURI string) {
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
	for _, want := range []string{
		"documentSymbolProvider",
		"definitionProvider",
		"referencesProvider",
		"callHierarchyProvider",
	} {
		assert.Contains(t, caps, want)
	}
	pipe.notify("initialized", map[string]any{})
}

func assertDocumentSymbolOutline(t *testing.T, pipe *lspPipe, uriA string) {
	t.Helper()
	syms := pipe.requestPickResult(t, "textDocument/documentSymbol", 100, map[string]any{
		"textDocument": map[string]any{"uri": uriA},
	}).([]any)
	require.NotEmpty(t, syms)
	root := syms[0].(map[string]any)
	assert.Equal(t, "Alpha", root["name"])
	require.Contains(t, root, "children")
}

func assertDefinitionJumpsToHeading(t *testing.T, pipe *lspPipe, uriA, uriB string) {
	t.Helper()
	defLoc := pipe.requestPickResult(t, "textDocument/definition", 101, map[string]any{
		"textDocument": map[string]any{"uri": uriB},
		"position":     map[string]any{"line": 2, "character": 12},
	})
	defObj, ok := defLoc.(map[string]any)
	require.True(t, ok, "definition result: %v", defLoc)
	assert.Equal(t, uriA, defObj["uri"])
}

func assertReferencesFromB(t *testing.T, pipe *lspPipe, uriA, uriB string) {
	t.Helper()
	refsRaw := pipe.requestPickResult(t, "textDocument/references", 102, map[string]any{
		"textDocument": map[string]any{"uri": uriA},
		"position":     map[string]any{"line": 2, "character": 3},
		"context":      map[string]any{"includeDeclaration": false},
	})
	refs, ok := refsRaw.([]any)
	require.True(t, ok, "references result: %v", refsRaw)
	require.Len(t, refs, 1)
	first := refs[0].(map[string]any)
	assert.Equal(t, uriB, first["uri"])
}

func assertCallHierarchyIncoming(t *testing.T, pipe *lspPipe, uriA string) {
	t.Helper()
	preparedRaw := pipe.requestPickResult(t, "textDocument/prepareCallHierarchy", 103, map[string]any{
		"textDocument": map[string]any{"uri": uriA},
		"position":     map[string]any{"line": 0, "character": 0},
	})
	prepared, ok := preparedRaw.([]any)
	require.True(t, ok)
	require.Len(t, prepared, 1)

	incomingRaw := pipe.requestPickResult(t, "callHierarchy/incomingCalls", 104, map[string]any{
		"item": prepared[0],
	})
	incoming, ok := incomingRaw.([]any)
	require.True(t, ok)
	require.Len(t, incoming, 1, "expected one incoming call from b.md")
}

// requestPickResult issues a request and returns the value at
// `result`. It also handles server-initiated requests interleaved
// with the response, so workspace/configuration replies don't stall
// the dispatch loop on the other side.
func (p *lspPipe) requestPickResult(t *testing.T, method string, id int, params any) any {
	t.Helper()
	p.writeFrame(map[string]any{
		"jsonrpc": "2.0", "id": id, "method": method, "params": params,
	})
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		m := p.readFrame()
		if mid, ok := m["id"].(float64); ok && int(mid) == id {
			return m["result"]
		}
		// Server-initiated request? Auto-ack.
		if method, _ := m["method"].(string); method != "" {
			if mid, ok := m["id"]; ok && mid != nil {
				p.writeFrame(map[string]any{
					"jsonrpc": "2.0", "id": mid, "result": nil,
				})
			}
		}
	}
	t.Fatalf("timed out waiting for response to %s", method)
	return nil
}

// pathToFileURIE2E mirrors the production server's pathToURI: it
// emits RFC 8089-compliant file URIs so the helper produces the
// same shape on every host OS (including Windows drive letters and
// UNC paths). Without that the E2E test would send a non-standard
// `file://C:/...` rootUri on Windows that the server's URI parser
// would treat as a UNC host.
func pathToFileURIE2E(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	if isWindowsDrivePathE2E(abs) {
		u := url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(abs)}
		return u.String()
	}
	if strings.HasPrefix(abs, `\\`) {
		rest := strings.TrimPrefix(filepath.ToSlash(abs), "//")
		host, tail, _ := strings.Cut(rest, "/")
		u := url.URL{Scheme: "file", Host: host, Path: "/" + tail}
		return u.String()
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	return u.String()
}

func isWindowsDrivePathE2E(p string) bool {
	if len(p) < 2 || p[1] != ':' {
		return false
	}
	c := p[0]
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// silence unused import in some build paths
var _ = json.Marshal
