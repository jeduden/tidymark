package main_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mcpRoundTrip sends newline-separated JSON-RPC messages to mdsmith mcp
// and returns the parsed response objects (one per response line).
func mcpRoundTrip(t *testing.T, messages ...string) []map[string]json.RawMessage {
	t.Helper()
	input := strings.Join(messages, "\n") + "\n"
	stdout, _, code := runBinary(t, input, "mcp")
	require.Equal(t, 0, code, "expected exit 0 from mcp server; stdout=%q", stdout)

	var results []map[string]json.RawMessage
	dec := json.NewDecoder(strings.NewReader(stdout))
	for dec.More() {
		var m map[string]json.RawMessage
		require.NoError(t, dec.Decode(&m), "decoding mcp response line")
		results = append(results, m)
	}
	return results
}

func TestMCP_Initialize(t *testing.T) {
	results := mcpRoundTrip(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
	)
	require.Len(t, results, 1)
	resultStr := string(results[0]["result"])
	assert.Contains(t, resultStr, "protocolVersion")
	assert.Contains(t, resultStr, "mdsmith")
}

func TestMCP_ToolsList(t *testing.T) {
	results := mcpRoundTrip(t,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
	)
	require.Len(t, results, 1)
	resultStr := string(results[0]["result"])
	assert.Contains(t, resultStr, "mdsmith_check")
	assert.Contains(t, resultStr, "mdsmith_fix")
}

func TestMCP_CheckDetectsDiagnostic(t *testing.T) {
	// A line longer than 80 chars triggers MDS001.
	longLine := "This line is intentionally very long to exceed the eighty character line-length limit set by default."
	content := "# Title\n\n" + longLine + "\n"

	callMsg, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "mdsmith_check",
			"arguments": map[string]any{"content": content},
		},
	})

	results := mcpRoundTrip(t, string(callMsg))
	require.Len(t, results, 1)

	// Result should be a tools/call result with content containing MDS001.
	resultStr := string(results[0]["result"])
	assert.Contains(t, resultStr, "MDS001", "expected line-length diagnostic")
	assert.Nil(t, results[0]["error"], "expected no JSON-RPC error")
}

func TestMCP_CheckCleanContent(t *testing.T) {
	content := "# Title\n\nShort clean paragraph.\n"

	callMsg, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "mdsmith_check",
			"arguments": map[string]any{"content": content},
		},
	})

	results := mcpRoundTrip(t, string(callMsg))
	require.Len(t, results, 1)

	// The text field should be a JSON array — empty when there are no diagnostics.
	resultStr := string(results[0]["result"])
	assert.Contains(t, resultStr, "[]", "expected empty diagnostic array for clean content")
}

func TestMCP_FixTrailingSpaces(t *testing.T) {
	content := "# Title  \n\nsome text   \n"

	callMsg, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "mdsmith_fix",
			"arguments": map[string]any{"content": content},
		},
	})

	results := mcpRoundTrip(t, string(callMsg))
	require.Len(t, results, 1)

	// The result is a toolsCallResult; the actual payload is in content[0].text,
	// which is itself a JSON string. Parse two levels deep.
	var callResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	require.NoError(t, json.Unmarshal(results[0]["result"], &callResult))
	require.NotEmpty(t, callResult.Content)

	var fixResult struct {
		Content   string `json:"content"`
		Changed   bool   `json:"changed"`
		Remaining []any  `json:"remaining"`
	}
	require.NoError(t, json.Unmarshal([]byte(callResult.Content[0].Text), &fixResult))
	assert.True(t, fixResult.Changed, "expected trailing spaces to be fixed")
	assert.Empty(t, fixResult.Remaining)
}

func TestMCP_FullRoundTrip(t *testing.T) {
	// Full initialize → tools/list → tools/call → shutdown sequence.
	content := "# Title\n\nClean paragraph.\n"
	callMsg, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "mdsmith_check",
			"arguments": map[string]any{"content": content},
		},
	})

	results := mcpRoundTrip(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		string(callMsg),
		`{"jsonrpc":"2.0","id":4,"method":"shutdown"}`,
	)

	// notifications/initialized has no id → no response; others get responses.
	require.Len(t, results, 4, "expected responses for initialize, tools/list, tools/call, shutdown")
}
