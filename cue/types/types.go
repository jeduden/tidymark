// Package types embeds the canonical mdsmith
// field-type-shortcut library so schema parsing can
// resolve `created: date` and friends without
// touching the network. The library source itself
// lives in `types.cue` next to this file.
//
// The intended import path for external CUE consumers
// is `github.com/jeduden/mdsmith/types`. The literal
// CUE import syntax is not yet implemented; today's
// surface is the bare-name YAML scalar handled by
// internal/schema's frontmatterExpr.
package types

import _ "embed"

//go:embed types.cue
var source string

// Source returns the embedded `types.cue` contents
// verbatim. internal/schema reads this once to seed
// its runtime registry and to drive the drift test
// that pins registry entries to the documented CUE.
func Source() string { return source }
