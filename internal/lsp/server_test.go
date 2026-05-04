package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jeduden/mdsmith/internal/rule"

	// Register rule packages so rule.All() returns the production set.
	_ "github.com/jeduden/mdsmith/internal/rules/linelength"
	_ "github.com/jeduden/mdsmith/internal/rules/notrailingspaces"
)

// testHarness wires a Server to a pair of in-memory pipes plus a
// dedicated reader goroutine that demultiplexes incoming frames so
// tests can wait for specific notifications without racing on the
// underlying bufio.Reader.
type testHarness struct {
	t            *testing.T
	srv          *Server
	clientWriter io.WriteCloser
	srvDone      chan error
	cancel       context.CancelFunc

	writeMu sync.Mutex
	nextID  atomic.Int64

	// Channels populated by the reader goroutine.
	notifications chan parsedNotification
	responses     chan parsedResponse

	// serverRequests holds method names of server-initiated requests
	// the reader has already auto-acked. Tests can assert their
	// presence without driving the read loop themselves.
	seenMu     sync.Mutex
	seenServer map[string]int
}

type parsedNotification struct {
	Method string
	Params json.RawMessage
}

type parsedResponse struct {
	ID   string
	Resp rpcResponse
}

func newHarness(t *testing.T) *testHarness {
	return newHarnessWithDebounce(t, -1)
}

func newDebouncedHarness(t *testing.T, debounce time.Duration) *testHarness {
	return newHarnessWithDebounce(t, debounce)
}

func newHarnessWithDebounce(t *testing.T, debounce time.Duration) *testHarness {
	t.Helper()
	srvIn, clientWriter := io.Pipe()
	clientRawReader, srvOut := io.Pipe()
	srv := New(Options{
		Rules:    rule.All(),
		Reader:   srvIn,
		Writer:   srvOut,
		Debounce: debounce,
	})
	// Tests want lint-on-didChange to fire so they can verify the
	// pipeline. Production defaults to onSave (skipping didChange);
	// flipping to onType keeps the existing tests deterministic.
	srv.settingsMu.Lock()
	srv.settings.Run = runOnType
	srv.settingsMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Run(ctx)
		_ = srvOut.Close()
	}()

	h := &testHarness{
		t:             t,
		srv:           srv,
		clientWriter:  clientWriter,
		srvDone:       done,
		cancel:        cancel,
		notifications: make(chan parsedNotification, 64),
		responses:     make(chan parsedResponse, 64),
		seenServer:    make(map[string]int),
	}

	go h.readPump(bufio.NewReader(clientRawReader))

	t.Cleanup(func() {
		_ = clientWriter.Close()
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Logf("server did not exit cleanly")
		}
	})
	return h
}

// readPump is the only goroutine that reads from the pipe. It demuxes
// frames into the notifications / responses channels, and auto-acks
// server-initiated requests so the dispatch loop never blocks.
func (h *testHarness) readPump(r *bufio.Reader) {
	for {
		raw, err := readFrame(r)
		if err != nil {
			return
		}
		var probe struct {
			ID     json.RawMessage `json:"id,omitempty"`
			Method string          `json:"method,omitempty"`
			Params json.RawMessage `json:"params,omitempty"`
			Result json.RawMessage `json:"result,omitempty"`
			Error  *responseError  `json:"error,omitempty"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			continue
		}

		// Server-initiated request: auto-ack with nil result so the
		// server's dispatch loop can move on, then record that we saw it.
		if probe.Method != "" && len(probe.ID) > 0 {
			h.write(struct {
				JSONRPC string          `json:"jsonrpc"`
				ID      json.RawMessage `json:"id"`
				Result  any             `json:"result"`
			}{JSONRPC: "2.0", ID: probe.ID, Result: nil})
			h.seenMu.Lock()
			h.seenServer[probe.Method]++
			h.seenMu.Unlock()
			continue
		}

		// Notification (no id, has method).
		if probe.Method != "" {
			select {
			case h.notifications <- parsedNotification{Method: probe.Method, Params: probe.Params}:
			default:
				// Drop on overflow — tests only care about a small
				// suffix of notifications.
			}
			continue
		}

		// Response to a client request.
		if len(probe.ID) > 0 {
			select {
			case h.responses <- parsedResponse{
				ID:   string(probe.ID),
				Resp: rpcResponse{Result: probe.Result, Error: probe.Error},
			}:
			default:
			}
		}
	}
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	tp := textproto.NewReader(r)
	hdr, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	cl := hdr.Get("Content-Length")
	if cl == "" {
		return nil, fmt.Errorf("missing Content-Length")
	}
	n, err := strconv.Atoi(cl)
	if err != nil {
		return nil, err
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (h *testHarness) write(v any) {
	body, err := json.Marshal(v)
	require.NoError(h.t, err)
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	if _, err := fmt.Fprintf(h.clientWriter, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return
	}
	_, _ = h.clientWriter.Write(body)
}

func (h *testHarness) request(method string, params any) (json.RawMessage, *responseError) {
	h.t.Helper()
	id := h.nextID.Add(1)
	idJSON, err := json.Marshal(id)
	require.NoError(h.t, err)
	h.write(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  any             `json:"params,omitempty"`
	}{JSONRPC: "2.0", ID: idJSON, Method: method, Params: params})

	deadline := time.After(5 * time.Second)
	for {
		select {
		case r := <-h.responses:
			if r.ID == string(idJSON) {
				return r.Resp.Result, r.Resp.Error
			}
		case <-deadline:
			h.t.Fatalf("timeout waiting for response to %s", method)
			return nil, nil
		}
	}
}

func (h *testHarness) notify(method string, params any) {
	h.t.Helper()
	h.write(notificationMessage{JSONRPC: "2.0", Method: method, Params: params})
}

// awaitNotification consumes from the notifications channel until it
// finds one matching method or hits the timeout.
func (h *testHarness) awaitNotification(method string, timeout time.Duration) json.RawMessage {
	h.t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case n := <-h.notifications:
			if n.Method == method {
				return n.Params
			}
		case <-deadline:
			h.t.Fatalf("timeout waiting for notification %s", method)
			return nil
		}
	}
}

// serverRequestCount returns how many times the server sent the named
// method as a request. Useful for asserting the server registered
// watchers / pulled configuration during initialization.
func (h *testHarness) serverRequestCount(method string) int {
	h.seenMu.Lock()
	defer h.seenMu.Unlock()
	return h.seenServer[method]
}

func TestInitializeAdvertisesCapabilities(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	resultRaw, errResp := h.request("initialize", initializeParams{
		Capabilities: clientCapabilities{Workspace: &workspaceClientCapabilities{Configuration: true}},
	})
	require.Nil(t, errResp)

	var res initializeResult
	require.NoError(t, json.Unmarshal(resultRaw, &res))
	assert.True(t, res.Capabilities.TextDocumentSync.OpenClose)
	assert.Equal(t, syncFull, res.Capabilities.TextDocumentSync.Change)
	assert.Contains(t, res.Capabilities.CodeActionProvider.CodeActionKinds, kindQuickFix)
	assert.Contains(t, res.Capabilities.CodeActionProvider.CodeActionKinds, kindSourceFixAll)
	assert.Equal(t, "mdsmith", res.ServerInfo.Name)
}

func TestDidOpenPublishesDiagnostics(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	doc := didOpenTextDocumentParams{
		TextDocument: textDocumentItem{
			URI:        "file:///workspace/sample.md",
			LanguageID: "markdown",
			Version:    1,
			Text:       "# Title\n\nThis line has trailing spaces   \n",
		},
	}
	h.notify("textDocument/didOpen", doc)

	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &p))
	assert.Equal(t, doc.TextDocument.URI, p.URI)

	var found bool
	for _, d := range p.Diagnostics {
		if d.Code == "MDS006" {
			found = true
			assert.NotZero(t, int(d.Severity))
			assert.Equal(t, "mdsmith", d.Source)
			require.NotNil(t, d.Data)
			assert.Equal(t, "no-trailing-spaces", d.Data.RuleName)
			break
		}
	}
	assert.True(t, found, "expected MDS006 diagnostic in %+v", p.Diagnostics)
}

func TestDidCloseClearsDiagnostics(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/clear.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: "# Hi\n\ntrailing  \n"},
	})
	first := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(first, &p))
	assert.NotEmpty(t, p.Diagnostics)

	h.notify("textDocument/didClose", didCloseTextDocumentParams{
		TextDocument: textDocumentIdentifier{URI: uri},
	})
	second := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var cleared publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(second, &cleared))
	assert.Equal(t, uri, cleared.URI)
	assert.Empty(t, cleared.Diagnostics)
}

func TestShutdownReturnsNullResult(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)
	resultRaw, errResp := h.request("shutdown", nil)
	require.Nil(t, errResp)
	assert.Equal(t, "null", string(resultRaw))
}

func TestDidChangeUpdatesDiagnostics(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/change.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{
			URI: uri, LanguageID: "markdown", Version: 1,
			Text: "# Hi\n\nclean line\n",
		},
	})
	first := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p1 publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(first, &p1))
	assert.Empty(t, p1.Diagnostics)

	h.notify("textDocument/didChange", didChangeTextDocumentParams{
		TextDocument: versionedTextDocumentIdentifier{URI: uri, Version: 2},
		ContentChanges: []textDocumentContentChangeEvent{
			{Text: "# Hi\n\ndirty line   \n"},
		},
	})
	second := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p2 publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(second, &p2))
	var saw006 bool
	for _, d := range p2.Diagnostics {
		if d.Code == "MDS006" {
			saw006 = true
			break
		}
	}
	assert.True(t, saw006, "expected MDS006 after didChange, got %+v", p2.Diagnostics)
}

func TestCodeActionQuickFix(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/qf.md"
	dirty := "# Hi\n\ndirty line   \n"
	clean := "# Hi\n\ndirty line\n"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: dirty},
	})
	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &p))

	var trailing Diagnostic
	for _, d := range p.Diagnostics {
		if d.Code == "MDS006" {
			trailing = d
			break
		}
	}
	require.NotEmpty(t, trailing.Code, "no MDS006 in %+v", p.Diagnostics)

	resultRaw, errResp := h.request("textDocument/codeAction", codeActionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Range:        trailing.Range,
		Context:      codeActionContext{Diagnostics: []Diagnostic{trailing}},
	})
	require.Nil(t, errResp)

	var actions []codeAction
	require.NoError(t, json.Unmarshal(resultRaw, &actions))
	require.NotEmpty(t, actions)

	var qf *codeAction
	for i, a := range actions {
		if a.Kind == kindQuickFix {
			qf = &actions[i]
			break
		}
	}
	require.NotNil(t, qf)
	require.NotNil(t, qf.Edit)
	edits, ok := qf.Edit.Changes[uri]
	require.True(t, ok)
	require.Len(t, edits, 1)
	assert.Equal(t, clean, edits[0].NewText)
}

func TestCodeActionSourceFixAll(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/all.md"
	dirty := "# Hi\n\ndirty line   \n"
	clean := "# Hi\n\ndirty line\n"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: dirty},
	})
	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &p))
	require.NotEmpty(t, p.Diagnostics)

	resultRaw, errResp := h.request("textDocument/codeAction", codeActionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Range:        Range{},
		Context:      codeActionContext{Diagnostics: p.Diagnostics, Only: []string{kindSourceFixAll}},
	})
	require.Nil(t, errResp)
	var actions []codeAction
	require.NoError(t, json.Unmarshal(resultRaw, &actions))
	var fixAll *codeAction
	for i, a := range actions {
		if a.Kind == kindSourceFixAll {
			fixAll = &actions[i]
			break
		}
	}
	require.NotNil(t, fixAll, "expected source.fixAll.mdsmith action; got %+v", actions)
	edits := fixAll.Edit.Changes[uri]
	require.Len(t, edits, 1)
	assert.Equal(t, clean, edits[0].NewText)
}

func TestCodeActionOnlyFiltersOutQuickFix(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/only.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: "# Hi\n\ndirty   \n"},
	})
	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &p))
	require.NotEmpty(t, p.Diagnostics)

	// Ask only for source.fixAll — quickfix actions must not be returned.
	resultRaw, errResp := h.request("textDocument/codeAction", codeActionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Range:        Range{},
		Context:      codeActionContext{Diagnostics: p.Diagnostics, Only: []string{kindSourceFixAll}},
	})
	require.Nil(t, errResp)
	var actions []codeAction
	require.NoError(t, json.Unmarshal(resultRaw, &actions))
	for _, a := range actions {
		assert.NotEqual(t, kindQuickFix, a.Kind, "Only=source.fixAll filter should suppress quickfix actions")
	}
}

func TestUnknownMethodReturnsError(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)
	_, errResp = h.request("textDocument/bogus", nil)
	require.NotNil(t, errResp)
	assert.Equal(t, codeMethodNotFound, errResp.Code)
}

func TestInitializedTriggersRegistration(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	h.notify("initialized", map[string]any{})
	// The reader pump auto-acks server requests and tracks them.
	// Wait briefly for the server to send both expected requests.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h.serverRequestCount("workspace/configuration") > 0 &&
			h.serverRequestCount("client/registerCapability") > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Positive(t, h.serverRequestCount("workspace/configuration"))
	assert.Positive(t, h.serverRequestCount("client/registerCapability"))
}

func TestDidChangeWatchedFilesRelintsOpenDocs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/watched.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: "# Hi\n\ndirty   \n"},
	})
	_ = h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)

	h.notify("workspace/didChangeWatchedFiles", didChangeWatchedFilesParams{
		Changes: []fileEvent{{URI: "file:///workspace/.mdsmith.yml", Type: 2}},
	})

	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &p))
	assert.Equal(t, uri, p.URI)
}

func TestDidChangeWatchedFilesIgnoresUnrelatedFiles(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)
	// Unrelated file: must not crash, no notification expected.
	h.notify("workspace/didChangeWatchedFiles", didChangeWatchedFilesParams{
		Changes: []fileEvent{{URI: "file:///workspace/other.txt", Type: 2}},
	})
	// Give the server a moment to process. We assert no diagnostics arrive.
	select {
	case n := <-h.notifications:
		t.Fatalf("unexpected notification: %s", n.Method)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDidChangeConfigurationRelintsOpenDocs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/cfg.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: "# Hi\n\ndirty   \n"},
	})
	_ = h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)

	h.notify("workspace/didChangeConfiguration", map[string]any{"settings": map[string]any{}})
	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &p))
	assert.Equal(t, uri, p.URI)
}

func TestDebouncedLintCollapsesRapidChanges(t *testing.T) {
	t.Parallel()
	h := newDebouncedHarness(t, 50*time.Millisecond)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/debounce.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: "# Hi\n\nclean\n"},
	})
	_ = h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)

	for i, txt := range []string{"# Hi\n\ndirty1   \n", "# Hi\n\ndirty2   \n", "# Hi\n\nfinal   \n"} {
		h.notify("textDocument/didChange", didChangeTextDocumentParams{
			TextDocument:   versionedTextDocumentIdentifier{URI: uri, Version: i + 2},
			ContentChanges: []textDocumentContentChangeEvent{{Text: txt}},
		})
	}
	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &p))
	assert.Equal(t, uri, p.URI)
	var saw006 bool
	for _, d := range p.Diagnostics {
		if d.Code == "MDS006" {
			saw006 = true
			break
		}
	}
	assert.True(t, saw006)
}

// TestDidSaveLintsWhenRunOnSave verifies that save events still
// produce diagnostics when run=onSave.
func TestDidSaveLintsWhenRunOnSave(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	// Reset to onSave to exercise the save-only path.
	h.srv.settingsMu.Lock()
	h.srv.settings.Run = runOnSave
	h.srv.settingsMu.Unlock()

	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/save.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: "# Hi\n\ndirty   \n"},
	})
	_ = h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)

	h.notify("textDocument/didSave", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
	raw := h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)
	var p publishDiagnosticsParams
	require.NoError(t, json.Unmarshal(raw, &p))
	assert.Equal(t, uri, p.URI)
}

// TestRunOffSuppressesLint verifies run=off skips lint passes from
// didChange but didOpen still produces an initial snapshot.
func TestRunOffSuppressesLint(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	_, errResp := h.request("initialize", initializeParams{})
	require.Nil(t, errResp)

	uri := "file:///workspace/off.md"
	h.notify("textDocument/didOpen", didOpenTextDocumentParams{
		TextDocument: textDocumentItem{URI: uri, LanguageID: "markdown", Version: 1, Text: "# Hi\n\nclean\n"},
	})
	_ = h.awaitNotification("textDocument/publishDiagnostics", 5*time.Second)

	// Switch run mode to off.
	h.srv.settingsMu.Lock()
	h.srv.settings.Run = runOff
	h.srv.settingsMu.Unlock()

	h.notify("textDocument/didChange", didChangeTextDocumentParams{
		TextDocument:   versionedTextDocumentIdentifier{URI: uri, Version: 2},
		ContentChanges: []textDocumentContentChangeEvent{{Text: "# Hi\n\ndirty   \n"}},
	})
	select {
	case n := <-h.notifications:
		t.Fatalf("unexpected diagnostic notification when run=off: %s", n.Method)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestUriToPathRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		uri  string
		want string
	}{
		{"file:///tmp/foo.md", "/tmp/foo.md"},
		{"file:///C:/Users/x/foo.md", "C:/Users/x/foo.md"},
		{"https://example.com", ""},
		{"untitled:Untitled-1", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, uriToPath(tc.uri), "uri=%s", tc.uri)
	}
}

func TestIsWholeFileOnly(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"catalog", "toc", "toc-directive", "include"} {
		assert.True(t, isWholeFileOnly(name), "expected %s to be whole-file-only", name)
	}
	assert.False(t, isWholeFileOnly("line-length"))
}

func TestIsFixableUsesRegistry(t *testing.T) {
	t.Parallel()
	rules := rule.All()
	assert.True(t, isFixable(rules, "no-trailing-spaces"))
	assert.False(t, isFixable(rules, "no-such-rule"))
}

func TestDocumentStoreOpenURIs(t *testing.T) {
	t.Parallel()
	s := newDocumentStore()
	s.set("file:///a", &document{uri: "file:///a", path: "/a"})
	s.set("file:///b", &document{uri: "file:///b", path: "/b"})
	got := s.openURIs()
	assert.ElementsMatch(t, []string{"file:///a", "file:///b"}, got)
}

func TestPickRootPrefersWorkspaceFolder(t *testing.T) {
	t.Parallel()
	got := pickRoot(initializeParams{
		WorkspaceFolders: []workspaceFolder{{URI: "file:///ws/root", Name: "ws"}},
		RootURI:          "file:///fallback",
	})
	assert.Equal(t, "/ws/root", got)

	got = pickRoot(initializeParams{RootURI: "file:///legacy"})
	assert.Equal(t, "/legacy", got)

	got = pickRoot(initializeParams{})
	assert.Equal(t, "", got)
}

func TestWantsKind(t *testing.T) {
	t.Parallel()
	assert.True(t, wantsKind(nil, "quickfix"))
	assert.True(t, wantsKind([]string{"quickfix"}, "quickfix"))
	assert.True(t, wantsKind([]string{"source"}, "source.fixAll.mdsmith"))
	assert.False(t, wantsKind([]string{"refactor"}, "quickfix"))
}

func TestDocumentEndPositionTrailingNewline(t *testing.T) {
	t.Parallel()
	endLine, endChar := documentEndPosition([]byte("hello\nworld\n"))
	assert.Equal(t, 2, endLine)
	assert.Equal(t, 0, endChar)
}

func TestDocumentEndPositionNoTrailingNewline(t *testing.T) {
	t.Parallel()
	endLine, endChar := documentEndPosition([]byte("hello\nworld"))
	assert.Equal(t, 1, endLine)
	assert.Equal(t, 5, endChar)
}

func TestDocumentEndPositionEmpty(t *testing.T) {
	t.Parallel()
	endLine, endChar := documentEndPosition(nil)
	assert.Equal(t, 0, endLine)
	assert.Equal(t, 0, endChar)
}

func TestReloadConfigEmptyRoot(t *testing.T) {
	t.Parallel()
	s := New(Options{Reader: nil, Writer: io.Discard})
	s.reloadConfig()
	cfg, _, _ := s.snapshotConfig()
	assert.NotNil(t, cfg, "expected default config when no root is set")
}

func TestReloadConfigDiscoverInTempDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, writeFile(dir+"/.mdsmith.yml", "rules:\n  line-length: false\n"))
	s := New(Options{Reader: nil, Writer: io.Discard})
	s.configMu.Lock()
	s.rootDir = dir
	s.configMu.Unlock()
	s.reloadConfig()
	cfg, path, _ := s.snapshotConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, dir+"/.mdsmith.yml", path)
}

func TestReloadConfigOverridePath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := dir + "/custom.yml"
	require.NoError(t, writeFile(cfgPath, "rules:\n  line-length: false\n"))

	s := New(Options{Reader: nil, Writer: io.Discard})
	s.settingsMu.Lock()
	s.settings.ConfigPath = cfgPath
	s.settingsMu.Unlock()
	s.reloadConfig()
	cfg, path, _ := s.snapshotConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, cfgPath, path)
}

func TestReloadConfigOverridePathInvalidUsesDefaults(t *testing.T) {
	t.Parallel()
	s := New(Options{Reader: nil, Writer: io.Discard})
	s.settingsMu.Lock()
	s.settings.ConfigPath = "/no/such/path.yml"
	s.settingsMu.Unlock()
	s.reloadConfig()
	cfg, path, _ := s.snapshotConfig()
	require.NotNil(t, cfg, "expected fallback to defaults on missing override")
	assert.Empty(t, path)
}

// writeFile is a tiny test helper that writes a file.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
