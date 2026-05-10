package mcp

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTools is a minimal toolsHandler used in server-level unit tests.
type stubTools struct{}

func (stubTools) list() []toolDef {
	return []toolDef{
		{
			Name:        "test_tool",
			Description: "a test tool",
			InputSchema: inputSch{
				Type:       "object",
				Properties: map[string]schemaProp{"x": {Type: "string", Description: "x"}},
				Required:   []string{"x"},
			},
		},
	}
}

func (stubTools) call(name string, args json.RawMessage) (any, error) {
	if name == "test_tool" {
		var a struct{ X string }
		_ = json.Unmarshal(args, &a)
		return map[string]string{"echo": a.X}, nil
	}
	return nil, nil
}

func roundTrip(t *testing.T, msgs ...string) []map[string]json.RawMessage {
	t.Helper()
	var in bytes.Buffer
	for _, m := range msgs {
		in.WriteString(m + "\n")
	}
	var out bytes.Buffer
	srv := NewServer(stubTools{})
	err := srv.Serve(&in, &out)
	require.NoError(t, err)

	var results []map[string]json.RawMessage
	dec := json.NewDecoder(&out)
	for dec.More() {
		var m map[string]json.RawMessage
		require.NoError(t, dec.Decode(&m))
		results = append(results, m)
	}
	return results
}

func TestServer_Initialize(t *testing.T) {
	results := roundTrip(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
	)
	require.Len(t, results, 1)
	assert.Equal(t, `"2.0"`, string(results[0]["jsonrpc"]))
	assert.Equal(t, `1`, string(results[0]["id"]))
	assert.Contains(t, string(results[0]["result"]), "protocolVersion")
}

func TestServer_ToolsList(t *testing.T) {
	results := roundTrip(t,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	require.Len(t, results, 1)
	assert.Contains(t, string(results[0]["result"]), "test_tool")
}

func TestServer_ToolsCall(t *testing.T) {
	results := roundTrip(t,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test_tool","arguments":{"x":"hello"}}}`,
	)
	require.Len(t, results, 1)
	result := string(results[0]["result"])
	assert.Contains(t, result, "hello")
}

func TestServer_NotificationNoResponse(t *testing.T) {
	// Notifications have no id — server must not emit a response.
	results := roundTrip(t,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
	)
	// Only the tools/list request should produce a response.
	require.Len(t, results, 1)
	assert.Contains(t, string(results[0]["result"]), "test_tool")
}

func TestServer_UnknownMethod(t *testing.T) {
	results := roundTrip(t,
		`{"jsonrpc":"2.0","id":4,"method":"unknown/method"}`,
	)
	require.Len(t, results, 1)
	assert.NotNil(t, results[0]["error"], "expected error for unknown method")
}

func TestServer_ParseError(t *testing.T) {
	results := roundTrip(t,
		`{invalid json`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/list"}`,
	)
	// Two responses: one error for bad JSON, one success for tools/list.
	require.Len(t, results, 2)
	assert.NotNil(t, results[0]["error"])
	assert.NotNil(t, results[1]["result"])
}

func TestServer_Shutdown(t *testing.T) {
	results := roundTrip(t,
		`{"jsonrpc":"2.0","id":6,"method":"shutdown"}`,
	)
	require.Len(t, results, 1)
	assert.Nil(t, results[0]["error"])
}
