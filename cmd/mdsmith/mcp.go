package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/jeduden/mdsmith/internal/mcp"
)

func runMCP(args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mdsmith mcp\n\n"+
			"Run an MCP (Model Context Protocol) server on stdio.\n\n"+
			"The server exposes two tools:\n"+
			"  mdsmith_check  Lint Markdown content and return diagnostics as JSON.\n"+
			"  mdsmith_fix    Auto-fix Markdown content and return corrected text.\n\n"+
			"Add mdsmith as an MCP server in .claude/settings.json:\n\n"+
			`  {`+"\n"+
			`    "mcpServers": {`+"\n"+
			`      "mdsmith": {`+"\n"+
			`        "type": "stdio",`+"\n"+
			`        "command": "mdsmith",`+"\n"+
			`        "args": ["mcp"]`+"\n"+
			`      }`+"\n"+
			`    }`+"\n"+
			`  }`+"\n\n"+
			"Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}

	srv := mcp.NewServer(mcp.MdsmithTools{})
	if err := srv.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "mdsmith mcp: %v\n", err)
		return 2
	}
	return 0
}
