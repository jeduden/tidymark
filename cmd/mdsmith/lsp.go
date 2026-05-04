package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	flag "github.com/spf13/pflag"

	"github.com/jeduden/mdsmith/internal/lsp"
	"github.com/jeduden/mdsmith/internal/rule"
)

// runLSP implements the "lsp" subcommand: speak Language Server
// Protocol over stdio. The server lives in internal/lsp; this file is
// only the CLI entry point.
func runLSP(args []string) int {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mdsmith lsp\n\n"+
			"Run the mdsmith Language Server Protocol server over stdio.\n"+
			"Designed to be spawned by an LSP client (VS Code, Neovim,\n"+
			"Helix, JetBrains LSP plugin). Reads JSON-RPC frames on\n"+
			"stdin and writes responses and notifications on stdout.\n\n"+
			"The server reuses the same lint and fix pipelines as\n"+
			"`mdsmith check` and `mdsmith fix`. See\n"+
			"docs/guides/editors/vscode.md for client setup.\n")
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "mdsmith: lsp takes no positional arguments\n")
		return 2
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv := lsp.New(lsp.Options{
		Rules:  rule.All(),
		Reader: os.Stdin,
		Writer: os.Stdout,
	})
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "mdsmith: lsp: %v\n", err)
		return 2
	}
	return 0
}
