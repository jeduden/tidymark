// Package mcp implements a Model Context Protocol server over stdio.
// It exposes mdsmith's check and fix capabilities as MCP tools so
// Claude Code and other MCP clients can lint and fix Markdown content
// without invoking a separate CLI process.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// protocolVersion is the MCP protocol version this server advertises.
const protocolVersion = "2024-11-05"

// serverName is the name reported in the initialize response.
const serverName = "mdsmith"

// jsonrpcVersion is the JSON-RPC version string.
const jsonrpcVersion = "2.0"

// request is a JSON-RPC 2.0 request or notification.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// response is a JSON-RPC 2.0 success response.
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result"`
}

// errorResponse is a JSON-RPC 2.0 error response.
type errorResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   rpcError        `json:"error"`
}

// rpcError carries a JSON-RPC 2.0 error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSON-RPC error codes.
const (
	codeParseError     = -32700
	codeMethodNotFound = -32601
	codeInternalError  = -32603
)

// toolsHandler is implemented by objects that handle MCP tool calls.
type toolsHandler interface {
	list() []toolDef
	call(name string, args json.RawMessage) (any, error)
}

// Server handles the MCP stdio transport.
type Server struct {
	tools toolsHandler
}

// NewServer creates an MCP server using the given tools handler.
func NewServer(tools toolsHandler) *Server {
	return &Server{tools: tools}
}

// Serve reads JSON-RPC 2.0 messages from r and writes responses to w
// until r is exhausted or an unrecoverable error occurs.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	enc := json.NewEncoder(w)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(errorResponse{
				JSONRPC: jsonrpcVersion,
				Error:   rpcError{Code: codeParseError, Message: "parse error: " + err.Error()},
			})
			continue
		}

		// Notifications (no ID) are processed but get no response.
		isNotification := req.ID == nil || string(req.ID) == "null"

		res, rpcErr := s.dispatch(&req)
		if isNotification {
			continue
		}

		if rpcErr != nil {
			_ = enc.Encode(errorResponse{
				JSONRPC: jsonrpcVersion,
				ID:      req.ID,
				Error:   *rpcErr,
			})
			continue
		}
		_ = enc.Encode(response{
			JSONRPC: jsonrpcVersion,
			ID:      req.ID,
			Result:  res,
		})
	}
	return scanner.Err()
}

// dispatch routes a request to the appropriate handler.
func (s *Server) dispatch(req *request) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params)
	case "notifications/initialized":
		return nil, nil
	case "tools/list":
		return s.handleToolsList()
	case "tools/call":
		return s.handleToolsCall(req.Params)
	case "shutdown":
		return struct{}{}, nil
	case "notifications/cancelled":
		return nil, nil
	default:
		return nil, &rpcError{
			Code:    codeMethodNotFound,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		}
	}
}

// initializeParams holds fields from the initialize request we care about.
type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// initializeResult is the response to initialize.
type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      serverInfo   `json:"serverInfo"`
	Capabilities    capabilities `json:"capabilities"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type capabilities struct {
	Tools toolsCapability `json:"tools"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

func (s *Server) handleInitialize(_ json.RawMessage) (any, *rpcError) {
	return initializeResult{
		ProtocolVersion: protocolVersion,
		ServerInfo:      serverInfo{Name: serverName, Version: ""},
		Capabilities:    capabilities{Tools: toolsCapability{ListChanged: false}},
	}, nil
}

// toolDef describes one MCP tool.
type toolDef struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	InputSchema inputSch  `json:"inputSchema"`
}

// inputSch is the JSON Schema for a tool's input.
type inputSch struct {
	Type       string              `json:"type"`
	Properties map[string]schemaProp `json:"properties"`
	Required   []string            `json:"required"`
}

type schemaProp struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type toolsListResult struct {
	Tools []toolDef `json:"tools"`
}

func (s *Server) handleToolsList() (any, *rpcError) {
	return toolsListResult{Tools: s.tools.list()}, nil
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolsCallResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) handleToolsCall(raw json.RawMessage) (any, *rpcError) {
	var p toolsCallParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpcError{Code: codeParseError, Message: "invalid tools/call params: " + err.Error()}
	}

	result, err := s.tools.call(p.Name, p.Arguments)
	if err != nil {
		text, _ := json.Marshal(map[string]string{"error": err.Error()})
		return toolsCallResult{
			Content: []contentBlock{{Type: "text", Text: string(text)}},
			IsError: true,
		}, nil
	}

	text, merr := json.Marshal(result)
	if merr != nil {
		return nil, &rpcError{Code: codeInternalError, Message: "marshal error: " + merr.Error()}
	}
	return toolsCallResult{
		Content: []contentBlock{{Type: "text", Text: string(text)}},
	}, nil
}
