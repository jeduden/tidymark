package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jeduden/mdsmith/internal/rules"
)

// diagCache stores the last-published diagnostics for each open URI.
// The hover handler reads this to find which rule covers the cursor
// position without re-running lint.
type diagCache struct {
	mu sync.RWMutex
	m  map[string][]Diagnostic
}

func newDiagCache() *diagCache {
	return &diagCache{m: make(map[string][]Diagnostic)}
}

func (c *diagCache) set(uri string, diags []Diagnostic) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[uri] = diags
}

func (c *diagCache) get(uri string) []Diagnostic {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.m[uri]
}

func (c *diagCache) delete(uri string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, uri)
}

// handleHover implements textDocument/hover.
//
// Resolution order (per plan 133):
//  1. Diagnostic-first: if the cursor falls inside an active
//     diagnostic range, return the rule docs for that diagnostic.
//  2. Directive fallback: if no diagnostic covers the cursor, check
//     whether the cursor is inside a <?directive …?> block; if so,
//     return the directive's guide docs.
//  3. Plain prose: return null (no hover).
func (s *Server) handleHover(msg *requestMessage) {
	var p hoverParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		_ = s.t.writeError(msg.ID, codeInvalidParams, "invalid hover params")
		return
	}

	doc, ok := s.docs.get(p.TextDocument.URI)
	if !ok {
		_ = s.t.writeResponse(msg.ID, nil)
		return
	}

	pos := p.Position

	// --- Pass 1: diagnostic-first ---
	diags := s.diagStore.get(p.TextDocument.URI)
	for _, d := range diags {
		if posInRange(pos, d.Range) {
			result := hoverForDiagnostic(d)
			if result != nil {
				_ = s.t.writeResponse(msg.ID, result)
				return
			}
		}
	}

	// --- Pass 2: directive fallback ---
	_, _, root := s.snapshotConfig()
	if result := hoverForDirective(pos, doc.text, root); result != nil {
		_ = s.t.writeResponse(msg.ID, result)
		return
	}

	// --- Pass 3: no match ---
	_ = s.t.writeResponse(msg.ID, nil)
}

// posInRange reports whether pos falls within r (inclusive start,
// exclusive end per LSP convention).
func posInRange(pos Position, r Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character >= r.End.Character {
		return false
	}
	return true
}

// hoverForDiagnostic returns a hoverResult for a diagnostic, loading
// the rule docs from the embedded rule READMEs. Returns nil only when
// the rule code is empty (which should not occur in practice).
func hoverForDiagnostic(d Diagnostic) *hoverResult {
	if d.Code == "" {
		return nil
	}

	body, err := rules.LookupRule(d.Code)
	if err != nil {
		// Unknown rule: surface a generic pointer to the CLI help.
		body = fmt.Sprintf("See `mdsmith help rule %s` for documentation.", d.Code)
	}

	value := fmt.Sprintf("**%s** — %s\n\n---\n\n%s", d.Code, d.Message, body)
	return &hoverResult{
		Contents: markupContent{Kind: "markdown", Value: value},
		Range:    &d.Range,
	}
}

// hoverForDirective checks whether pos falls inside a <?name …?>
// block in source and, if so, returns hover content loaded from the
// directive guide docs in the workspace root. Returns nil when no
// directive covers pos.
func hoverForDirective(pos Position, source []byte, rootDir string) *hoverResult {
	block, ok := findDirectiveAtPos(pos, source)
	if !ok {
		return nil
	}

	docContent := loadDirectiveDocs(block.name, rootDir)
	if docContent == "" {
		// Directive exists but we have no doc file; return a short note.
		docContent = fmt.Sprintf(
			"See the [directives guide](docs/guides/directives/) for `<?%s?>` documentation.",
			block.name,
		)
	}

	return &hoverResult{
		Contents: markupContent{Kind: "markdown", Value: docContent},
		Range:    &block.blockRange,
	}
}

// directiveToFile maps a canonical directive name to its guide doc
// file path relative to docs/guides/directives/.
var directiveToFile = map[string]string{
	"catalog":             "generating-content.md",
	"include":             "generating-content.md",
	"build":               "build.md",
	"require":             "enforcing-structure.md",
	"allow-empty-section": "enforcing-structure.md",
}

// loadDirectiveDocs reads the guide doc for a directive from the
// workspace root. Returns "" when no doc file exists or the file
// cannot be read.
func loadDirectiveDocs(name, rootDir string) string {
	file, ok := directiveToFile[name]
	if !ok {
		return ""
	}
	if rootDir == "" {
		return ""
	}
	//nolint:gosec // path is constructed from the directive map, not user input
	data, err := os.ReadFile(filepath.Join(rootDir, "docs", "guides", "directives", file))
	if err != nil {
		return ""
	}
	return stripDocFrontMatter(string(data))
}

// stripDocFrontMatter removes the leading YAML front matter block
// (--- … ---) from a guide doc file so the hover popup shows the
// Markdown body, not raw YAML metadata.
func stripDocFrontMatter(s string) string {
	if !strings.HasPrefix(s, "---\n") {
		return s
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return s
	}
	body := s[4+end+5:]
	return strings.TrimLeft(body, "\n")
}

// directiveBlock holds the parsed directive name and its LSP range.
type directiveBlock struct {
	name       string
	blockRange Range
}

// findDirectiveAtPos scans source for a <?name …?> block that
// contains pos (0-based LSP line/char). Returns the block and true
// when found.
//
// The scan is line-by-line and does not use goldmark: the hover
// handler needs to work on the raw buffer as the editor has it,
// potentially before a full re-parse, and the goldmark AST does not
// expose 0-based LSP line numbers directly. We mirror the pi_parser
// logic: <?name starts a block, ?> (on its own line) closes it.
func findDirectiveAtPos(pos Position, source []byte) (directiveBlock, bool) {
	lines := splitLines(source)
	type openBlock struct {
		name      string
		startLine int
	}
	var open *openBlock
	for lineIdx, line := range lines {
		trimmed := strings.TrimLeft(string(line), " ")
		if open == nil {
			db, matched, isPI, closed := tryOpenDirective(trimmed, line, lineIdx, pos)
			if matched {
				return db, true
			}
			if isPI && !closed {
				open = &openBlock{name: db.name, startLine: lineIdx}
			}
			continue
		}
		// Inside an open block: check for the closing ?> line.
		if strings.TrimSpace(string(line)) == "?>" {
			r := blockRange(open.startLine, lineIdx, line)
			if pos.Line >= open.startLine && pos.Line <= lineIdx {
				return directiveBlock{name: open.name, blockRange: r}, true
			}
			open = nil
		}
	}
	// Unclosed block at EOF.
	if open != nil {
		endLine := len(lines) - 1
		if endLine < 0 {
			endLine = 0
		}
		r := blockRange(open.startLine, endLine, lines[endLine])
		if pos.Line >= open.startLine && pos.Line <= endLine {
			return directiveBlock{name: open.name, blockRange: r}, true
		}
	}
	return directiveBlock{}, false
}

// tryOpenDirective attempts to parse an opening <?name …?> tag on trimmed.
// Returns (block, matched, isPI, closed) where:
//   - isPI=false means the line is not a PI opener; caller should skip.
//   - matched=true means the cursor is inside this single-line directive.
//   - closed=true means this is a single-line PI (regardless of match).
//   - When isPI=true, matched=false, closed=false: multi-line PI opener;
//     caller should open a block with block.name.
func tryOpenDirective(trimmed string, line []byte, lineIdx int, pos Position) (directiveBlock, bool, bool, bool) {
	if !strings.HasPrefix(trimmed, "<?") {
		return directiveBlock{}, false, false, false
	}
	name := extractPIName(trimmed[2:])
	if name == "" || strings.HasPrefix(name, "/") {
		return directiveBlock{}, false, false, false
	}
	// Single-line PI: contains ?> after the opening tag.
	if strings.Contains(strings.TrimRight(trimmed, " \t\r\n")[2:], "?>") {
		r := Range{
			Start: Position{Line: lineIdx, Character: 0},
			End:   Position{Line: lineIdx, Character: utf16Length(line)},
		}
		if pos.Line == lineIdx {
			return directiveBlock{name: name, blockRange: r}, true, true, true
		}
		return directiveBlock{name: name}, false, true, true
	}
	// Multi-line PI opener.
	return directiveBlock{name: name}, false, true, false
}

// blockRange returns an LSP Range covering startLine through endLine
// (inclusive). The end character is set to the full UTF-16 length of
// endLineBytes so the range covers the closing ?> marker.
func blockRange(startLine, endLine int, endLineBytes []byte) Range {
	return Range{
		Start: Position{Line: startLine, Character: 0},
		End:   Position{Line: endLine, Character: utf16Length(endLineBytes)},
	}
}

// extractPIName extracts the directive name from the bytes after "<?".
// Returns "" if no valid name can be parsed. Mirrors extractPINameBytes
// from pi_parser.go but works on strings.
func extractPIName(rest string) string {
	rest = strings.TrimRight(rest, "\r\n")
	for i, c := range rest {
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			return rest[:i]
		}
		if c == '?' && i+1 < len(rest) && rest[i+1] == '>' {
			return rest[:i]
		}
	}
	return rest
}
