// Package index builds and maintains the symbol graph that powers
// mdsmith's LSP navigation methods (documentSymbol, definition,
// references, workspace/symbol, callHierarchy).
//
// The graph stores four kinds of symbols — headings, link-reference
// definitions, top-level front-matter keys, and directives — together
// with the inbound/outbound reference edges that connect them across
// files: anchor links, file links, reference-style links, and the
// include / catalog / build directive targets.
//
// Build is workspace-wide; updates are per-file. Callers re-parse one
// buffer with Update on document events and rebuild the whole index
// when the project's `.mdsmith.yml` changes (kind / ignore globs may
// shift scope).
package index

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// SymbolKind enumerates the four symbol shapes the index recognizes.
// Each maps to a specific LSP SymbolKind in the LSP layer; this
// package keeps the spec-level numbers out of its core types.
type SymbolKind int

const (
	// SymbolHeading is a Markdown heading at any level (H1–H6). The
	// Anchor field carries the slug; the Level field carries the
	// heading level.
	SymbolHeading SymbolKind = iota
	// SymbolLinkRef is a `[label]: url` link-reference definition.
	// The Anchor field carries the normalized label.
	SymbolLinkRef
	// SymbolFrontMatter is a top-level YAML front-matter key. The
	// Name field carries the key.
	SymbolFrontMatter
	// SymbolDirective is a processing-instruction block (<?name … ?>).
	// The Name field carries the directive name.
	SymbolDirective
)

// Symbol is one entry in a file's outline.
type Symbol struct {
	// File is the workspace-relative path of the containing file
	// (forward slashes, no leading `./`). Index lookups key on this.
	File string
	// Kind is the symbol category.
	Kind SymbolKind
	// Name is the human-readable label (heading text, key, label,
	// directive name).
	Name string
	// Anchor is the normalized identifier used for cross-document
	// lookups: heading slug, link-ref label, or "" for other kinds.
	Anchor string
	// Level is the heading level (1–6) for SymbolHeading; 0 otherwise.
	Level int
	// StartLine, EndLine are 1-based line numbers covering the
	// symbol's full range. For headings the range extends to the
	// next sibling heading; for other kinds it's the source line.
	StartLine int
	EndLine   int
	// SelectionLine, SelectionCol point to the symbol's name/label
	// (1-based) — what an editor highlights when "go to definition"
	// jumps to it.
	SelectionLine int
	SelectionCol  int
}

// EdgeKind enumerates the kinds of references the index tracks.
type EdgeKind int

const (
	// EdgeAnchorLink is `[text](#anchor)` — same-file heading reference.
	EdgeAnchorLink EdgeKind = iota
	// EdgeFileLink is `[text](./other.md)` (with optional anchor).
	EdgeFileLink
	// EdgeRefLink is `[text][label]` — reference-style link use.
	EdgeRefLink
	// EdgeInclude is a `<?include file: …?>` directive.
	EdgeInclude
	// EdgeCatalog is a `<?catalog?>` directive.
	EdgeCatalog
	// EdgeBuild is a `<?build source: …?>` directive.
	EdgeBuild
)

// Edge records one reference from a source position to a target.
//
// Empty TargetFile means "same file as Source" (used for anchor and
// reference-style links). Empty TargetAnchor means the reference
// targets the file as a whole (e.g. `[text](./other.md)`).
//
// Unresolved is true for edges whose target cannot be expressed as a
// single workspace file at extraction time. Catalog edges
// (<?catalog?>) set this flag because the directive uses a glob and
// resolves to many files at expansion time. IncomingEdges /
// BacklinksFor skip Unresolved edges so a catalog placeholder doesn't
// surface as a phantom self-backlink on the host file.
type Edge struct {
	SourceFile   string
	SourceLine   int // 1-based
	SourceCol    int // 1-based
	TargetFile   string
	TargetAnchor string
	TargetLabel  string
	Kind         EdgeKind
	Unresolved   bool
	// Globs holds the catalog patterns when Unresolved is true. The
	// LSP server's call-hierarchy uses them to render which patterns
	// a catalog directive walks.
	Globs []string
}

// FileEntry is one file's contribution to the index.
type FileEntry struct {
	// Path is the workspace-relative path with forward slashes.
	Path string
	// Symbols are this file's symbols, in document order.
	Symbols []Symbol
	// Outgoing are the references this file emits.
	Outgoing []Edge
	// Title is the front-matter `title:` value if set, "" otherwise.
	Title string
	// Kinds are the front-matter `kinds:` values if set.
	Kinds []string
	// LineCount is the number of source lines (1-based-inclusive
	// upper bound for symbol ranges). Used to bound heading ranges.
	LineCount int
}

// Index is the workspace-wide symbol graph. Methods are safe to call
// concurrently with each other; concurrent Update/Remove on the same
// path is serialized internally.
type Index struct {
	mu    sync.RWMutex
	root  string
	files map[string]*FileEntry
}

// New returns an empty Index rooted at root. Build populates it.
func New(root string) *Index {
	return &Index{
		root:  root,
		files: make(map[string]*FileEntry),
	}
}

// Root returns the workspace root the index was built against.
func (i *Index) Root() string {
	if i == nil {
		return ""
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.root
}

// Files returns a snapshot of the indexed file paths in arbitrary
// order. Callers must not retain the slice across mutations of the
// index.
func (i *Index) Files() []string {
	if i == nil {
		return nil
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]string, 0, len(i.files))
	for path := range i.files {
		out = append(out, path)
	}
	return out
}

// File returns a snapshot of the FileEntry for the given workspace-
// relative path. The pointer is to a copy so callers may read the
// slices without holding the index lock; the slices themselves are
// shared, so callers must not mutate them.
func (i *Index) File(path string) (*FileEntry, bool) {
	if i == nil {
		return nil, false
	}
	path = NormalizePath(path)
	i.mu.RLock()
	defer i.mu.RUnlock()
	fe, ok := i.files[path]
	if !ok {
		return nil, false
	}
	cp := *fe
	return &cp, true
}

// upsert installs or replaces a FileEntry under fe.Path.
func (i *Index) upsert(fe *FileEntry) {
	i.mu.Lock()
	i.files[fe.Path] = fe
	i.mu.Unlock()
}

// Remove drops the entry for path. No-op when absent.
func (i *Index) Remove(path string) {
	if i == nil {
		return
	}
	path = NormalizePath(path)
	i.mu.Lock()
	delete(i.files, path)
	i.mu.Unlock()
}

// Update re-parses source under path and replaces the FileEntry.
// When source is empty the file is removed entirely (matches the
// case where the file was deleted from disk).
//
// path must be workspace-relative. AbsPathToWorkspace is provided as
// a helper for callers that hold an absolute filesystem path.
func (i *Index) Update(path string, source []byte) {
	if i == nil {
		return
	}
	path = NormalizePath(path)
	if path == "" {
		return
	}
	if len(source) == 0 {
		i.Remove(path)
		return
	}
	fe := buildFileEntry(path, source)
	i.upsert(fe)
}

// UpdateWithKinds is Update plus an override for the file's effective
// kinds list. Callers pass the resolved (front-matter ∪ kind-
// assignment) list so workspace-symbol search and `kind:` navigation
// see config-driven assignments, not just front-matter declarations.
// When kinds is nil the result is identical to Update.
func (i *Index) UpdateWithKinds(path string, source []byte, kinds []string) {
	if i == nil {
		return
	}
	path = NormalizePath(path)
	if path == "" {
		return
	}
	if len(source) == 0 {
		i.Remove(path)
		return
	}
	fe := buildFileEntry(path, source)
	if kinds != nil {
		fe.Kinds = append([]string(nil), kinds...)
	}
	i.upsert(fe)
}

// Build walks the workspace and indexes every Markdown file the
// supplied loader yields. The loader is called once per discovered
// path; returning an error skips that file. files is the list of
// workspace-relative paths to index, typically produced by
// discovery.Discover and then made workspace-relative.
//
// Build replaces the entire current index, including evicting any
// entries whose path no longer appears in files.
func (i *Index) Build(files []string, load func(path string) ([]byte, error)) {
	if i == nil {
		return
	}
	next := make(map[string]*FileEntry, len(files))
	for _, p := range files {
		path := NormalizePath(p)
		if path == "" {
			continue
		}
		data, err := load(path)
		if err != nil || len(data) == 0 {
			continue
		}
		next[path] = buildFileEntry(path, data)
	}
	i.mu.Lock()
	i.files = next
	i.mu.Unlock()
}

// BuildParallel is Build with the per-file parse fanned out across
// workers goroutines. workers <= 1 falls back to the sequential
// Build path. The semantics match Build: the index is replaced
// wholesale, and any path that load can't satisfy is dropped.
//
// The parallel path is safe because per-file extraction is pure given
// its (path, source) inputs — linkgraph's extractors touch no global
// state, and buildFileEntry only writes to the FileEntry it returns.
// Workers slice the input file list into contiguous chunks and write
// to per-worker output slices, so the only synchronization is the
// final waitgroup and the index-wide write at the end.
func (i *Index) BuildParallel(files []string, load func(path string) ([]byte, error), workers int) {
	if i == nil {
		return
	}
	if workers <= 1 || len(files) == 0 {
		i.Build(files, load)
		return
	}
	norm := normalizeFileList(files)
	if len(norm) == 0 {
		i.mu.Lock()
		i.files = make(map[string]*FileEntry)
		i.mu.Unlock()
		return
	}
	results := buildEntriesParallel(norm, load, workers)
	next := make(map[string]*FileEntry, len(norm))
	for _, batch := range results {
		for _, fe := range batch {
			next[fe.Path] = fe
		}
	}
	i.mu.Lock()
	i.files = next
	i.mu.Unlock()
}

// normalizeFileList returns the workspace-relative form of every
// non-empty entry in files, dropping the rest.
func normalizeFileList(files []string) []string {
	out := make([]string, 0, len(files))
	for _, p := range files {
		if np := NormalizePath(p); np != "" {
			out = append(out, np)
		}
	}
	return out
}

// buildEntriesParallel splits paths into contiguous chunks and runs
// buildFileEntry concurrently across workers. Each worker writes to
// its own slot in results so the only synchronization is the
// waitgroup; the per-file work itself is pure under the inputs it
// receives.
func buildEntriesParallel(paths []string, load func(path string) ([]byte, error), workers int) [][]*FileEntry {
	chunk := (len(paths) + workers - 1) / workers
	results := make([][]*FileEntry, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= len(paths) {
			break
		}
		end := start + chunk
		if end > len(paths) {
			end = len(paths)
		}
		wg.Add(1)
		go func(slot int, chunkPaths []string) {
			defer wg.Done()
			out := make([]*FileEntry, 0, len(chunkPaths))
			for _, p := range chunkPaths {
				data, err := load(p)
				if err != nil || len(data) == 0 {
					continue
				}
				out = append(out, buildFileEntry(p, data))
			}
			results[slot] = out
		}(w, paths[start:end])
	}
	wg.Wait()
	return results
}

// IncomingEdges returns every workspace edge whose target is the
// given (file, anchor). When anchor is "" matches edges to the file
// at large (no anchor specified by the caller). Edges marked as
// Unresolved (catalog directives, which use globs rather than a fixed
// target) are skipped — they don't cite any specific file, so they
// must not surface as backlinks to the host file.
//
// The returned slice is a fresh copy.
func (i *Index) IncomingEdges(file, anchor string) []Edge {
	if i == nil {
		return nil
	}
	file = NormalizePath(file)
	i.mu.RLock()
	defer i.mu.RUnlock()
	var out []Edge
	for _, fe := range i.files {
		for _, e := range fe.Outgoing {
			if e.Unresolved {
				continue
			}
			tFile := e.TargetFile
			if tFile == "" {
				tFile = fe.Path
			}
			tFile = NormalizePath(tFile)
			if tFile != file {
				continue
			}
			if anchor != "" && e.TargetAnchor != anchor {
				continue
			}
			out = append(out, e)
		}
	}
	return out
}

// BacklinksFor returns every workspace edge whose target is file,
// regardless of anchor. Use this for the "what cites this file?"
// question — IncomingEdges(file, anchor) answers the narrower
// "what targets this specific heading".
//
// Catalog directives are filtered out at the IncomingEdges layer:
// they carry the Unresolved flag because the directive's target is a
// glob, not a fixed file. Without that filter a catalog host would
// surface as its own backlink.
//
// Same-file citations (EdgeAnchorLink, EdgeRefLink) stay in the
// result so callers can filter on SourceFile when they want only
// external citations. The returned slice is freshly allocated and
// sorted by (SourceFile, SourceLine, SourceCol) so callers
// presenting the result to a user — or asserting on it in a
// test — see a stable order regardless of the underlying map
// iteration.
func (i *Index) BacklinksFor(file string) []Edge {
	if i == nil {
		return nil
	}
	edges := i.IncomingEdges(file, "")
	sort.Slice(edges, func(a, b int) bool {
		if edges[a].SourceFile != edges[b].SourceFile {
			return edges[a].SourceFile < edges[b].SourceFile
		}
		if edges[a].SourceLine != edges[b].SourceLine {
			return edges[a].SourceLine < edges[b].SourceLine
		}
		return edges[a].SourceCol < edges[b].SourceCol
	})
	return edges
}

// OutgoingEdges returns the edges originating in file.
func (i *Index) OutgoingEdges(file string) []Edge {
	if i == nil {
		return nil
	}
	fe, ok := i.File(file)
	if !ok {
		return nil
	}
	out := make([]Edge, len(fe.Outgoing))
	copy(out, fe.Outgoing)
	return out
}

// FilesByKind returns workspace files whose front-matter `kinds:`
// list contains kind. Order is undefined.
func (i *Index) FilesByKind(kind string) []string {
	if i == nil || kind == "" {
		return nil
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	var out []string
	for path, fe := range i.files {
		for _, k := range fe.Kinds {
			if k == kind {
				out = append(out, path)
				break
			}
		}
	}
	return out
}

// SearchSymbols returns symbols whose name (case-insensitive)
// contains query. Match scope:
//
//   - heading text
//   - link-ref labels
//   - front-matter title (matched against the file's Title)
//   - kind names from kinds:
//
// Returns at most max entries (0 = unlimited).
func (i *Index) SearchSymbols(query string, max int) []SymbolMatch {
	if i == nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(query))
	i.mu.RLock()
	defer i.mu.RUnlock()
	var out []SymbolMatch
	full := func() bool { return max > 0 && len(out) >= max }
	for path, fe := range i.files {
		out = matchFileSymbols(out, path, fe, q)
		if full() {
			return out[:max]
		}
		out = matchFileTitle(out, path, fe, q)
		if full() {
			return out[:max]
		}
		out = matchFileKinds(out, path, fe, q)
		if full() {
			return out[:max]
		}
	}
	return out
}

// matchFileSymbols appends matches for headings and link refs.
func matchFileSymbols(out []SymbolMatch, path string, fe *FileEntry, q string) []SymbolMatch {
	for _, s := range fe.Symbols {
		if s.Kind != SymbolHeading && s.Kind != SymbolLinkRef {
			continue
		}
		if !nameMatches(s.Name, q) {
			continue
		}
		out = append(out, SymbolMatch{File: path, Symbol: s})
	}
	return out
}

// matchFileTitle appends a synthetic Title symbol when the file's
// front-matter title matches.
func matchFileTitle(out []SymbolMatch, path string, fe *FileEntry, q string) []SymbolMatch {
	if fe.Title == "" || !nameMatches(fe.Title, q) {
		return out
	}
	return append(out, SymbolMatch{
		File: path,
		Symbol: Symbol{
			File:          path,
			Kind:          SymbolFrontMatter,
			Name:          fe.Title,
			StartLine:     1,
			EndLine:       1,
			SelectionLine: 1,
			SelectionCol:  1,
		},
	})
}

// matchFileKinds appends one synthetic symbol per matching kind.
func matchFileKinds(out []SymbolMatch, path string, fe *FileEntry, q string) []SymbolMatch {
	for _, k := range fe.Kinds {
		if !nameMatches(k, q) {
			continue
		}
		out = append(out, SymbolMatch{
			File: path,
			Symbol: Symbol{
				File:          path,
				Kind:          SymbolFrontMatter,
				Name:          "kind:" + k,
				StartLine:     1,
				EndLine:       1,
				SelectionLine: 1,
				SelectionCol:  1,
			},
		})
	}
	return out
}

// nameMatches returns true when q is empty or a case-insensitive
// substring of name.
func nameMatches(name, q string) bool {
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), q)
}

// SymbolMatch pairs a Symbol with the file that contains it. Returned
// from workspace-wide queries so callers can build LSP locations.
type SymbolMatch struct {
	File   string
	Symbol Symbol
}

// NormalizePath returns path with forward slashes and no leading
// `./`. Empty input passes through. Backslashes are translated even
// on platforms where filepath.ToSlash is a no-op so a Windows-style
// path landing in the index from a cross-platform test still keys
// against the same slot as the slashed form.
func NormalizePath(path string) string {
	if path == "" {
		return ""
	}
	p := strings.ReplaceAll(filepath.ToSlash(path), `\`, "/")
	p = strings.TrimPrefix(p, "./")
	return p
}

// AbsPathToWorkspace returns the workspace-relative form of abs given
// the index's root directory. When abs is already relative, or when
// root is empty, the input is returned unchanged.
func (i *Index) AbsPathToWorkspace(abs string) string {
	if i == nil {
		return abs
	}
	i.mu.RLock()
	root := i.root
	i.mu.RUnlock()
	return absToWorkspace(root, abs)
}

func absToWorkspace(root, abs string) string {
	if root == "" || !filepath.IsAbs(abs) {
		return NormalizePath(abs)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return NormalizePath(abs)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return NormalizePath(abs)
	}
	return NormalizePath(rel)
}
