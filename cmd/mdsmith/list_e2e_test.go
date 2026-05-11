package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2E_List_NoArgs_PrintsUsage(t *testing.T) {
	_, stderr, exitCode := runBinary(t, "", "list")
	assert.Equal(t, 0, exitCode, "list with no sub prints usage and exits 0")
	assert.Contains(t, stderr, "Usage: mdsmith list <subcommand>")
	assert.Contains(t, stderr, "query")
	assert.Contains(t, stderr, "backlinks")
}

func TestE2E_List_HelpFlag(t *testing.T) {
	_, stderr, exitCode := runBinary(t, "", "list", "--help")
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stderr, "Usage: mdsmith list <subcommand>")
}

func TestE2E_List_UnknownSubcommand_ExitsTwo(t *testing.T) {
	_, stderr, exitCode := runBinary(t, "", "list", "nope")
	assert.Equal(t, 2, exitCode)
	assert.Contains(t, stderr, "unknown subcommand")
	// Usage banner accompanies the error for discoverability.
	assert.Contains(t, stderr, "Usage: mdsmith list <subcommand>")
}
