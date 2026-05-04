package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
	"sync"
)

// transport reads and writes LSP-framed JSON-RPC messages over the
// supplied reader and writer. Frames use the HTTP-style header form:
//
//	Content-Length: N\r\n
//	\r\n
//	<N bytes of UTF-8 JSON>
//
// Concurrent readers are not supported; concurrent writers are
// serialized via writeMu so the server can publish notifications from
// any goroutine.
type transport struct {
	r       *bufio.Reader
	w       io.Writer
	writeMu sync.Mutex
}

func newTransport(r io.Reader, w io.Writer) *transport {
	return &transport{r: bufio.NewReader(r), w: w}
}

// readMessage reads one framed JSON-RPC message. Returns io.EOF when
// the stream closes cleanly between frames.
func (t *transport) readMessage() (*requestMessage, error) {
	tp := textproto.NewReader(t.r)
	header, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	clRaw := header.Get("Content-Length")
	if clRaw == "" {
		return nil, errors.New("lsp: missing Content-Length header")
	}
	cl, err := strconv.Atoi(clRaw)
	if err != nil {
		return nil, fmt.Errorf("lsp: invalid Content-Length %q: %w", clRaw, err)
	}
	if cl < 0 || cl > 64*1024*1024 {
		return nil, fmt.Errorf("lsp: Content-Length %d out of bounds", cl)
	}
	body := make([]byte, cl)
	if _, err := io.ReadFull(t.r, body); err != nil {
		return nil, fmt.Errorf("lsp: reading body: %w", err)
	}
	var msg requestMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("lsp: parsing JSON: %w", err)
	}
	return &msg, nil
}

// writeJSON marshals v and emits a framed message.
func (t *transport) writeJSON(v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("lsp: encoding JSON: %w", err)
	}
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	if _, err := fmt.Fprintf(t.w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return fmt.Errorf("lsp: writing header: %w", err)
	}
	if _, err := t.w.Write(body); err != nil {
		return fmt.Errorf("lsp: writing body: %w", err)
	}
	return nil
}

// writeResponse writes a successful response. A nil result is
// serialized as JSON null (LSP requires the result field on success).
func (t *transport) writeResponse(id json.RawMessage, result any) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("lsp: encoding result: %w", err)
	}
	return t.writeJSON(responseMessage{JSONRPC: "2.0", ID: id, Result: raw})
}

// writeError writes an error response.
func (t *transport) writeError(id json.RawMessage, code int, msg string) error {
	return t.writeJSON(responseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &responseError{Code: code, Message: msg},
	})
}

// writeNotification writes a server-to-client notification.
func (t *transport) writeNotification(method string, params any) error {
	return t.writeJSON(notificationMessage{JSONRPC: "2.0", Method: method, Params: params})
}

// writeRequest writes a server-to-client request. The id MUST be
// pre-encoded so callers can match the response.
func (t *transport) writeRequest(id json.RawMessage, method string, params any) error {
	return t.writeJSON(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  any             `json:"params,omitempty"`
	}{JSONRPC: "2.0", ID: id, Method: method, Params: params})
}
