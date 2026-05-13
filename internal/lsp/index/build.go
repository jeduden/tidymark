package index

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/jeduden/mdsmith/internal/linkgraph"
	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/mdtext"
	"github.com/jeduden/mdsmith/internal/yamlutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v3"
)

// buildFileEntry parses source under filePath (workspace-relative) and
// extracts the symbol/edge tables for that file.
func buildFileEntry(filePath string, source []byte) *FileEntry {
	fe := &FileEntry{
		Path:      NormalizePath(filePath),
		LineCount: countLines(source),
	}

	// Front matter is parsed first because it carries the file's
	// title / kinds — both surfaced as workspace-symbol matches.
	fmBytes, body := lint.StripFrontMatter(source)
	fmOffset := countLines(fmBytes)
	fe.Symbols = append(fe.Symbols, frontMatterSymbols(fe.Path, fmBytes)...)
	if title, ok := frontMatterScalar(fmBytes, "title"); ok {
		fe.Title = title
	}
	if kinds, ok := frontMatterStringList(fmBytes, "kinds"); ok {
		fe.Kinds = kinds
	}

	// Parse the body with the same goldmark configuration the lint
	// pipeline uses, so processing-instructions surface as our
	// custom AST node. The parser context carries the resolved
	// reference-definition map that collectLinkRefDefs needs.
	ctx := parser.NewContext()
	root := lint.NewParser().Parse(text.NewReader(body), parser.WithContext(ctx))
	lines := bytes.Split(body, []byte("\n"))

	// Wrap the parsed body in a *lint.File so the linkgraph extractor
	// (which is the single source of truth for Markdown link parsing)
	// can walk the same AST without re-parsing. f.LineOffset carries
	// the front-matter row count so callers add it back when they
	// need file-relative coordinates.
	f := &lint.File{
		Path:       fe.Path,
		Source:     body,
		AST:        root,
		LineOffset: fmOffset,
	}

	// Headings drive the outline.
	headingSyms := collectHeadings(fe.Path, root, body, lines, fmOffset, fe.LineCount)
	fe.Symbols = append(fe.Symbols, headingSyms...)

	// Link reference definitions (parsed by goldmark) flatten alongside.
	fe.Symbols = append(fe.Symbols, collectLinkRefDefs(fe.Path, ctx, body, lines, fmOffset)...)

	// Directives (PIs) at the document root.
	fe.Symbols = append(fe.Symbols, collectDirectives(fe.Path, root, body, fmOffset)...)

	// Edges: anchor / file / ref-style links plus directive targets.
	// Both link and directive extraction go through linkgraph so the
	// LSP index, MDS027, and `mdsmith list backlinks` walk the same
	// bytes through the same parser.
	fe.Outgoing = append(fe.Outgoing, collectLinkEdges(f)...)
	fe.Outgoing = append(fe.Outgoing, collectDirectiveEdges(f)...)

	return fe
}

func countLines(source []byte) int {
	if len(source) == 0 {
		return 0
	}
	n := bytes.Count(source, []byte{'\n'})
	if source[len(source)-1] != '\n' {
		n++
	}
	return n
}

// collectHeadings returns one Symbol per heading. Range extends to
// the line before the next heading at the same or lower level — that
// matches how outline UIs (VS Code's symbol picker, Helix's
// jump-to-symbol) shade the section.
func collectHeadings(filePath string, root ast.Node, source []byte, lines [][]byte, fmOffset, totalLines int) []Symbol {
	heads, headStart := walkHeadings(root, source)
	syms := make([]Symbol, 0, len(heads))
	usedAnchors := make(map[string]bool)
	slugCounts := make(map[string]int)
	for i, h := range heads {
		txt := mdtext.ExtractPlainText(h, source)
		anchor := uniqueAnchor(mdtext.Slugify(txt), usedAnchors, slugCounts)
		startLine := headStart[i] + fmOffset
		endLine := headingEndLine(heads, headStart, i, fmOffset, totalLines)
		col := columnOfLine(lines, headStart[i]-1, h.Lines().At(0).Start, source)
		syms = append(syms, Symbol{
			File:          filePath,
			Kind:          SymbolHeading,
			Name:          txt,
			Anchor:        anchor,
			Level:         h.Level,
			StartLine:     startLine,
			EndLine:       endLine,
			SelectionLine: startLine,
			SelectionCol:  col,
		})
	}
	return syms
}

// walkHeadings collects every ast.Heading in document order along
// with its 1-based source line. Goldmark guarantees a parsed
// heading has at least one source line; setext-style headings
// also produce non-empty Lines().
func walkHeadings(root ast.Node, source []byte) ([]*ast.Heading, []int) {
	var heads []*ast.Heading
	var headStart []int
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok || h.Lines().Len() == 0 {
			return ast.WalkContinue, nil
		}
		heads = append(heads, h)
		headStart = append(headStart, lineOfOffset(source, h.Lines().At(0).Start))
		return ast.WalkContinue, nil
	})
	return heads, headStart
}

// uniqueAnchor returns a slug suffixed with -1, -2, … when the bare
// slug is already used, mirroring CommonMark / GitHub disambiguation.
func uniqueAnchor(slug string, used map[string]bool, counts map[string]int) string {
	if slug == "" {
		return ""
	}
	anchor := slug
	if used[anchor] {
		c := counts[slug]
		for {
			c++
			anchor = fmt.Sprintf("%s-%d", slug, c)
			if !used[anchor] {
				break
			}
		}
		counts[slug] = c
	}
	used[anchor] = true
	return anchor
}

// headingEndLine returns the 1-based last line that belongs to
// heads[i]'s section: the line before the next heading at the same
// or lower level, clamped to totalLines.
func headingEndLine(heads []*ast.Heading, headStart []int, i, fmOffset, totalLines int) int {
	startLine := headStart[i] + fmOffset
	endLine := totalLines
	for j := i + 1; j < len(heads); j++ {
		if heads[j].Level <= heads[i].Level {
			endLine = headStart[j] - 1 + fmOffset
			break
		}
	}
	if endLine < startLine {
		endLine = startLine
	}
	return endLine
}

// columnOfLine returns the 1-based column of an absolute byte offset
// within a body parsed without the front matter. lines is bytes.Split
// of the same body.
func columnOfLine(lines [][]byte, lineIdx int, absOffset int, source []byte) int {
	if lineIdx < 0 || lineIdx >= len(lines) {
		return 1
	}
	// Compute cumulative offset to start of lineIdx.
	cum := 0
	for i := 0; i < lineIdx; i++ {
		cum += len(lines[i]) + 1 // +1 for the \n
	}
	if absOffset < cum {
		return 1
	}
	if absOffset > cum+len(lines[lineIdx]) {
		absOffset = cum + len(lines[lineIdx])
	}
	return absOffset - cum + 1
}

// lineOfOffset is a 1-based line index for a byte offset in source.
func lineOfOffset(source []byte, offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(source) {
		offset = len(source)
	}
	line := 1
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return line
}

// frontMatterSymbols extracts top-level YAML keys from the front
// matter prefix and returns one Symbol per key. Lines are 1-based
// from the start of the file. Parsing goes through yamlutil so the
// index never expands a YAML alias on user-controlled content — the
// rest of mdsmith treats every front-matter parse as a potential
// alias-bomb vector and the symbol index has to match.
func frontMatterSymbols(filePath string, fm []byte) []Symbol {
	if len(fm) == 0 {
		return nil
	}
	node, err := yamlutil.UnmarshalNodeSafe(stripDelimiters(fm))
	if err != nil || len(node.Content) == 0 {
		return nil
	}
	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	out := make([]Symbol, 0, len(mapping.Content)/2)
	// yaml.v3 line numbers are 1-based within the parsed buffer; the
	// stripped buffer drops the leading "---" line so add 1.
	// Non-scalar keys (mapping or sequence keys per YAML spec) and
	// empty key values are skipped — they don't produce a sensible
	// outline entry and an empty Symbol.Name would render as a
	// blank row in the editor's outline.
	for i := 0; i < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		if k.Kind != yaml.ScalarNode || k.Value == "" {
			continue
		}
		out = append(out, Symbol{
			File:          filePath,
			Kind:          SymbolFrontMatter,
			Name:          k.Value,
			StartLine:     k.Line + 1,
			EndLine:       k.Line + 1,
			SelectionLine: k.Line + 1,
			SelectionCol:  k.Column,
		})
	}
	return out
}

// stripDelimiters removes the leading and trailing `---\n` lines
// from a front-matter prefix as returned by lint.StripFrontMatter.
// The trailing strip uses TrimSuffix with the exact `---\n`
// pattern (or `---` without a trailing newline as a fallback for
// truncated input) rather than scanning for the last occurrence
// of `---`. The previous LastIndex approach could match `---`
// inside YAML content (e.g. inside a multi-line quoted string),
// which would over-truncate the front matter.
func stripDelimiters(fm []byte) []byte {
	body := fm
	body = bytes.TrimPrefix(body, []byte("---\n"))
	if t := bytes.TrimSuffix(body, []byte("---\n")); len(t) != len(body) {
		return t
	}
	return bytes.TrimSuffix(body, []byte("---"))
}

// frontMatterScalar returns a top-level scalar key from front matter
// as a string. Empty string + false when absent or non-scalar.
// yamlutil.UnmarshalSafe rejects anchors/aliases so a malicious file
// can't trigger expansion during the symbol-index build. Non-string
// scalars (numbers, bools) are formatted via fmt.Sprintf so callers
// always get a stable string form.
func frontMatterScalar(fm []byte, key string) (string, bool) {
	if len(fm) == 0 {
		return "", false
	}
	var m map[string]any
	if err := yamlutil.UnmarshalSafe(stripDelimiters(fm), &m); err != nil {
		return "", false
	}
	v, ok := m[key]
	if !ok {
		return "", false
	}
	if s, ok := v.(string); ok {
		return s, true
	}
	return fmt.Sprintf("%v", v), true
}

// frontMatterStringList returns a top-level YAML list of strings.
// Parses via yamlutil so YAML aliases are rejected before any
// expansion can happen on the user's input.
func frontMatterStringList(fm []byte, key string) ([]string, bool) {
	if len(fm) == 0 {
		return nil, false
	}
	var m map[string]any
	if err := yamlutil.UnmarshalSafe(stripDelimiters(fm), &m); err != nil {
		return nil, false
	}
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	list, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok {
			continue
		}
		out = append(out, s)
	}
	return out, true
}

// refDefRE matches a CommonMark reference definition at the start of
// a line. Cribbed from internal/rules/nounusedlinkdefinitions.
var refDefRE = regexp.MustCompile(`(?m)^[ ]{0,3}\[([^\]\n]+)\]:[ \t]*\S+.*$`)

// RefDefRegexpMatches returns the same submatch indices
// refDefRE.FindAllSubmatchIndex produces for body. Exported so the
// LSP rename surface can iterate every reference definition without
// duplicating the regex pattern (and without giving callers a way
// to mutate the package-level pattern).
func RefDefRegexpMatches(body []byte) [][]int {
	return refDefRE.FindAllSubmatchIndex(body, -1)
}

// collectLinkRefDefs finds `[label]: url` lines in body. The CommonMark
// reference definition map is stored in parser.Context (`ctx.References`)
// — we use it to confirm a regex match really is a definition (not a
// link inside a paragraph), then read the line/col from the source.
func collectLinkRefDefs(filePath string, ctx parser.Context, body []byte, lines [][]byte, fmOffset int) []Symbol {
	wanted := map[string]bool{}
	for _, ref := range ctx.References() {
		wanted[string(ref.Label())] = true
	}
	if len(wanted) == 0 {
		return nil
	}
	// Track normalized labels we've already emitted: goldmark
	// resolves only the first definition for any label, so duplicate
	// regex matches must not produce extra outline entries that
	// would confuse the symbol picker.
	seen := map[string]bool{}
	var out []Symbol
	for _, m := range refDefRE.FindAllSubmatchIndex(body, -1) {
		raw := body[m[2]:m[3]]
		label := string(raw)
		anchor := string(util.ToLinkReference(raw))
		if !wanted[anchor] || seen[anchor] {
			continue
		}
		seen[anchor] = true
		// m[2]-1 is the offset of `[`; m[2] is the offset of the
		// label's first byte. Use the label position so "go to
		// definition" highlights the label, not the bracket.
		labelOffset := m[2]
		line := lineOfOffset(body, labelOffset) + fmOffset
		col := columnOfLine(lines, lineOfOffset(body, labelOffset)-1, labelOffset, body)
		out = append(out, Symbol{
			File:          filePath,
			Kind:          SymbolLinkRef,
			Name:          label,
			Anchor:        anchor,
			StartLine:     line,
			EndLine:       line,
			SelectionLine: line,
			SelectionCol:  col,
		})
	}
	return out
}

// collectDirectives returns one Symbol per processing-instruction
// block at the document root. Closing markers (<?/name?>) are
// skipped; only the opener is treated as a symbol.
func collectDirectives(filePath string, root ast.Node, source []byte, fmOffset int) []Symbol {
	var out []Symbol
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		pi, ok := n.(*lint.ProcessingInstruction)
		if !ok {
			continue
		}
		if strings.HasPrefix(pi.Name, "/") {
			continue
		}
		startLine, endLine := piLineRange(pi, source, fmOffset)
		out = append(out, Symbol{
			File:          filePath,
			Kind:          SymbolDirective,
			Name:          pi.Name,
			StartLine:     startLine,
			EndLine:       endLine,
			SelectionLine: startLine,
			SelectionCol:  1,
		})
	}
	return out
}

// piLineRange returns the 1-based [start, end] source lines for a
// processing-instruction block. The PI parser guarantees Lines() is
// non-empty for any parsed PI, so the helper does not handle that
// case. The closing-marker offset (`?>`) on a continuation line
// gives the end; goldmark emits HasClosure() == true for every
// well-formed PI, so the branch where Lines() spans multiple
// segments without a closure is unreachable in practice.
func piLineRange(pi *lint.ProcessingInstruction, source []byte, fmOffset int) (int, int) {
	startSeg := pi.Lines().At(0)
	startLine := lineOfOffset(source, startSeg.Start) + fmOffset
	endLine := startLine
	if pi.HasClosure() && pi.ClosureLine.Start > startSeg.Start {
		endLine = lineOfOffset(source, pi.ClosureLine.Start) + fmOffset
	}
	return startLine, endLine
}

// collectLinkEdges maps linkgraph's per-file link extraction onto
// the index's Edge type. Anchor-only links (`#fragment`) produce
// EdgeAnchorLink, file-targeting links produce EdgeFileLink (with
// any anchor slugified for cross-file lookups), and reference-style
// links (`[text][label]`) produce EdgeRefLink.
//
// File-link targets are resolved against f.Path through
// linkgraph.ResolveRelTarget; targets that escape the workspace
// root return "" and are dropped — recording them with an empty
// TargetFile would let IncomingEdges treat the edge as a
// self-reference.
//
// All edge source-line numbers are file-relative (link.Line +
// f.LineOffset), matching the rest of the index's coordinate system.
func collectLinkEdges(f *lint.File) []Edge {
	links := linkgraph.ExtractLinks(f)
	refs := linkgraph.ExtractLinkRefs(f)
	out := make([]Edge, 0, len(links)+len(refs))

	for _, l := range links {
		t := l.Target
		line := l.Line + f.LineOffset
		if t.LocalAnchor {
			out = append(out, Edge{
				SourceFile:   f.Path,
				SourceLine:   line,
				SourceCol:    l.Column,
				TargetAnchor: linkgraph.NormalizeAnchor(t.Anchor),
				Kind:         EdgeAnchorLink,
			})
			continue
		}
		tgt := linkgraph.ResolveRelTarget(f.Path, t.Path)
		if tgt == "" {
			continue
		}
		out = append(out, Edge{
			SourceFile:   f.Path,
			SourceLine:   line,
			SourceCol:    l.Column,
			TargetFile:   tgt,
			TargetAnchor: linkgraph.NormalizeAnchor(t.Anchor),
			Kind:         EdgeFileLink,
		})
	}

	for _, r := range refs {
		out = append(out, Edge{
			SourceFile:  f.Path,
			SourceLine:  r.Line + f.LineOffset,
			SourceCol:   r.Column,
			TargetLabel: r.Label,
			Kind:        EdgeRefLink,
		})
	}
	return out
}

// ResolveRelTarget is a thin re-export of linkgraph.ResolveRelTarget
// so external callers (the LSP server's directive-arg navigation
// paths) can keep using `index.ResolveRelTarget` and pick up the
// unified escape rules.
func ResolveRelTarget(srcFile, linkPath string) string {
	return linkgraph.ResolveRelTarget(srcFile, linkPath)
}

// collectDirectiveEdges maps linkgraph's directive extraction onto
// the index's Edge type. <?include?> and <?build?> produce edges
// whose TargetFile is resolved via linkgraph.ResolveRelTarget;
// edges whose target escapes the workspace are dropped.
// <?catalog?> produces an edge with the Unresolved flag and the raw
// glob list — IncomingEdges skips Unresolved edges, so a catalog
// host never shows up as its own backlink.
func collectDirectiveEdges(f *lint.File) []Edge {
	dirs := linkgraph.ExtractDirectives(f)
	out := make([]Edge, 0, len(dirs))
	for _, d := range dirs {
		line := d.Line + f.LineOffset
		switch d.Kind {
		case linkgraph.DirectiveInclude:
			if tgt := linkgraph.ResolveRelTarget(f.Path, d.TargetPath); tgt != "" {
				out = append(out, Edge{
					SourceFile: f.Path,
					SourceLine: line,
					SourceCol:  d.Column,
					TargetFile: tgt,
					Kind:       EdgeInclude,
				})
			}
		case linkgraph.DirectiveBuild:
			if tgt := linkgraph.ResolveRelTarget(f.Path, d.TargetPath); tgt != "" {
				out = append(out, Edge{
					SourceFile: f.Path,
					SourceLine: line,
					SourceCol:  d.Column,
					TargetFile: tgt,
					Kind:       EdgeBuild,
				})
			}
		case linkgraph.DirectiveCatalog:
			out = append(out, Edge{
				SourceFile: f.Path,
				SourceLine: line,
				SourceCol:  d.Column,
				Kind:       EdgeCatalog,
				Unresolved: true,
				Globs:      d.Globs,
			})
		}
	}
	return out
}
