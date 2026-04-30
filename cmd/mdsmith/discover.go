package main

import "github.com/jeduden/mdsmith/internal/githooks"

// discoverFilesWithGeneratedContent is a thin shim around
// githooks.DiscoverFiles so existing call sites keep working. The
// shared implementation lives in internal/githooks so the CLI and the
// MDS048 rule cannot drift.
func discoverFilesWithGeneratedContent(repoRoot string, maxBytes int64) []string {
	return githooks.DiscoverFiles(repoRoot, maxBytes)
}
