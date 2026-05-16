package release

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

// ruleReadmeLink matches a Markdown link whose target is a rule
// README relative path and captures the rule directory (with an
// optional `../` prefix) separately from the `README.md` filename.
// It covers both link forms in play: the bare
// `MDS001-line-length/README.md` used by the rule index, and the
// sibling `../MDS021-include/README.md` used between per-rule
// READMEs. Rewriting drops `README.md` so the link resolves to the
// published page directory (`MDS001-line-length/`,
// `../MDS021-include/`) rather than an unpublished `README.md`.
var ruleReadmeLink = regexp.MustCompile(`\]\(((?:\.\./)?MDS[0-9A-Za-z._-]+)/README\.md\)`)

// ruleRefDefLink matches a Markdown reference-style link definition
// whose target is a rule README path (with an optional `../`
// prefix). The multiline flag makes `^` match at each line start.
// Example: [mds027]: ../MDS027-cross-file-reference-integrity/README.md
// Rewriting drops `/README.md` and adds a trailing slash so the
// reference resolves to the published rule page directory.
var ruleRefDefLink = regexp.MustCompile(`(?m)^(\[[^\]]+\]: (?:\.\./)?(MDS[0-9A-Za-z._-]+))/README\.md`)

// repoDocsLink matches an inline link from a rule README into the
// docs/ tree (`../../../docs/path/file.md`). The docs tree IS
// published on the site (under /docs/), but Hugo serves each page
// at `/docs/path/file/` (no `.md`), so the raw relative path
// resolves to a 404. Group 1 captures the path without `.md`;
// group 2 captures an optional `#anchor` fragment.
var repoDocsLink = regexp.MustCompile(`\]\(\.\./\.\./\.\./docs/([^)#]*)\.md([^)]*)\)`)

// repoPlanLink matches an inline link from a rule README into the
// plan/ tree (`../../../plan/file.md`). Plan files are not published
// on the site and must be rewritten as absolute GitHub blob URLs.
var repoPlanLink = regexp.MustCompile(`\]\(\.\./\.\./\.\./plan/([^)]+)\)`)

// repoPlanRefDef matches a reference-style link definition whose
// target is a plan/ file. The multiline flag makes `^` line-anchored.
// Example: [plan107]: ../../../plan/107_no-reference-style.md
var repoPlanRefDef = regexp.MustCompile(`(?m)^(\[[^\]]+\]: )\.\./\.\./\.\./plan/(\S+)`)

// repoRuleLink matches an inline link from any docs/ page into
// the internal/rules/ tree (`../…/internal/rules/MDS001-line-length/`
// or `../…/internal/rules/MDS001-line-length/README.md`). Unlike
// ruleReadmeLink (which only matches the bare or single-`../`
// prefix used inside rule pages themselves), this regex accepts
// any depth of `../` segments so plain docs deep in the tree
// (e.g. docs/background/concepts/generated-section.md, 4 levels
// up) get their rule links rewritten too. Group 1 captures the
// rule directory name. The trailing `README.md` is optional —
// docs link to both forms. Group 2 captures an optional
// `#anchor` so a deep-link into a rule README's heading
// (e.g. `MDS020-required-structure/README.md#index-side-output`)
// preserves the fragment after the slash.
var repoRuleLink = regexp.MustCompile(`\]\((?:\.\./)+internal/rules/(MDS[0-9A-Za-z._-]+)/(?:README\.md)?(#[^)]*)?\)`)

// repoRuleRefDef matches a reference-style link definition whose
// target is a repo-relative internal/rules/ path. Multiline flag
// so `^` anchors at each line start. Mirrors repoRuleLink — any
// `../` depth, optional README.md suffix, optional anchor.
// Example: [mds020]: ../../internal/rules/MDS020-required-structure/README.md
var repoRuleRefDef = regexp.MustCompile(`(?m)^(\[[^\]]+\]: )(?:\.\./)+internal/rules/(MDS[0-9A-Za-z._-]+)/(?:README\.md)?(#\S+)?$`)

// rulePageURLBase is the site-absolute URL prefix every rule page
// lives under. Hugo serves website/content/docs/rules/<dir>/index.md
// at /docs/rules/<dir>/, so repo-relative `internal/rules/<dir>/`
// links from any docs page must rewrite to this prefix to resolve.
const rulePageURLBase = "/docs/rules/"

// repoNonPublishedLink matches an inline link whose target is a
// repo-relative path that the website does NOT publish — every
// tree under the repo that lives outside docs/ and outside
// internal/rules/MDS*/ falls here. On the source tree the link
// resolves to a real file (so mdsmith fix and humans reading the
// source see it fine); on the synced Hugo tree there is no such
// path, so the link 404s. Rewriting to an absolute GitHub blob
// URL keeps the reference clickable on the live site while still
// pointing at the canonical source on github.com.
//
// Group 1 captures the repo-relative path (everything past the
// last `../`). The alternative order matters: every `internal/…`
// prefix that is NOT `internal/rules/MDS…/` falls here, but
// `internal/rules/MDS…/` itself is already rewritten by
// rewriteRuleLinks (run first) and never reaches this regex.
// The `internal/rules/<non-MDS>/` case (e.g.
// `internal/rules/markdownflavor/`) is intentionally caught
// here because only MDS-prefixed directories carry a published
// README.
//
// Root-level files (PLAN.md, README.md, LICENSE, …) are listed
// explicitly. They live at the repo root with no enclosing
// directory, so a `(?:\.\./)+` prefix is the only signal that
// the link target is repo-relative rather than sibling.
var repoNonPublishedLink = regexp.MustCompile(
	`\]\((?:\.\./)+(` +
		`plan/[^)]+|` +
		`cmd/[^)]+|` +
		`editors/[^)]+|` +
		`cue/[^)]+|` +
		`npm/[^)]+|` +
		`python/[^)]+|` +
		`\.claude/[^)]+|` +
		`\.github/[^)]+|` +
		`internal/[^)]+|` +
		`PLAN\.md|README\.md|LICENSE|SECURITY\.md|CLAUDE\.md|AGENTS\.md` +
		`)\)`)

// repoNonPublishedRefDef is the reference-style sibling of
// repoNonPublishedLink for definitions like
// `[plan107]: ../../../plan/107.md`. Multiline anchor; `\S+`
// captures the target so trailing whitespace or comments after
// the URL are not eaten.
var repoNonPublishedRefDef = regexp.MustCompile(
	`(?m)^(\[[^\]]+\]: )(?:\.\./)+(` +
		`plan/\S+|` +
		`cmd/\S+|` +
		`editors/\S+|` +
		`cue/\S+|` +
		`npm/\S+|` +
		`python/\S+|` +
		`\.claude/\S+|` +
		`\.github/\S+|` +
		`internal/\S+|` +
		`PLAN\.md|README\.md|LICENSE|SECURITY\.md|CLAUDE\.md|AGENTS\.md` +
		`)`)

// indexMdLink matches a sibling-style inline link whose target
// is `index.md` (the docs/-tree convention for a directory
// overview). syncDocsFile renames every `index.md` to
// `_index.md` when copying into the Hugo tree, so a link that
// kept the source name resolves to nothing on the synced
// filesystem (and MDS027 flags it). Rewrite `index.md` in the
// link target to `_index.md` to follow the rename. Group 1
// captures the path prefix (e.g. `architecture/`); group 2
// captures an optional `#anchor` fragment.
var indexMdLink = regexp.MustCompile(`\]\(((?:[^)/]+/)*)index\.md((?:#[^)]*)?)\)`)

// ruleFixtureLink matches an inline link in a per-rule README
// whose target is a fixture path under the rule's own directory:
// `[good/default.md](good/default.md)`, `[bad/x.md](bad/x.md)`,
// or the `pattern/bad/` / `pattern/good/` directives-rule case.
// Fixtures live in the source tree but are intentionally not
// republished on the site (no Hugo page for raw test data), so a
// repo-relative link 404s. Rewrite to the rule's GitHub source
// tree URL so the example file is still reachable.
var ruleFixtureLink = regexp.MustCompile(`\]\(((?:bad|good|pattern)/[^)]*)\)`)

// ruleSiblingNonMDSLink matches an inline link in a per-rule
// README whose target is a single-`../`-prefixed sibling under
// internal/rules/ that is NOT another rule's page — the rule's
// Go package directory or the shared `proto.md` schema, for
// example. Sibling MDS rule pages (`../MDS021-include/`) ARE
// published, so they are excluded by requiring the first
// character after `../` to be lowercase or a dot (rule names
// start with uppercase `M`).
var ruleSiblingNonMDSLink = regexp.MustCompile(`\]\(\.\./([a-z._][^)]*)\)`)

// ruleSourceTreeBase is the GitHub directory (tree) route for a
// rule's source. Per-rule READMEs carry an
// `Implementation: [source](./)` link that points at the rule's
// own directory; on the published site `./` would self-link the
// generated page, so syncRulePages rewrites it to the rule's
// GitHub tree URL. `/tree/` (not `/blob/`) is the directory route.
const ruleSourceTreeBase = "https://github.com/jeduden/mdsmith/tree/main/internal/rules/"

// githubBlobBase is the GitHub blob (file) URL prefix for files in
// the repository that are not published on the site (plan/, etc.).
const githubBlobBase = "https://github.com/jeduden/mdsmith/blob/main/"

// ruleDirName matches the MDS-prefixed directory names used for
// per-rule subdirectories under internal/rules/. The prefix guard
// stops syncRulePages from copying non-rule directories (fixtures,
// helper files, …) into the Hugo content tree.
var ruleDirName = regexp.MustCompile(`^MDS[0-9]`)

// rewriteRuleLinks rewrites every repo-relative link in a synced
// markdown body so it resolves on the published site. The three
// classes are applied in order: (1) links into internal/rules/MDS…/
// become /docs/rules/<dir>/<#anchor> site URLs; (2) links into any
// other non-published repo path — plan/, cmd/, editors/, .claude/,
// internal/ (other than the rule pages already handled in step 1),
// and root-level files — become absolute GitHub blob URLs;
// (3) sibling links to `index.md` get the `_index.md` rename
// SyncDocs applied to the file itself, so MDS027 still resolves
// the target on the synced filesystem.
//
// The ordering matters because the rule rewrite is the only one
// that has a site-local target — the non-published rewrite would
// otherwise consume `internal/rules/MDS…` and send it to GitHub.
//
// Idempotent: already-rewritten paths (a leading `/docs/`,
// `https://`, or `_index.md`) do not match any of the three
// regexes, so a second pass is a no-op.
func rewriteRuleLinks(b []byte) []byte {
	b = repoRuleLink.ReplaceAll(b, []byte("]("+rulePageURLBase+"$1/$2)"))
	b = repoRuleRefDef.ReplaceAll(b, []byte("${1}"+rulePageURLBase+"$2/$3"))
	b = repoNonPublishedLink.ReplaceAll(b, []byte("]("+githubBlobBase+"$1)"))
	b = repoNonPublishedRefDef.ReplaceAll(b, []byte("${1}"+githubBlobBase+"$2"))
	// `${1}` (braced form) is required: `$1_index.md` would parse
	// as a variable named `1_index` to Go's regexp expander, so
	// the captured directory prefix would silently vanish (and
	// every link become `.md`).
	b = indexMdLink.ReplaceAll(b, []byte("](${1}_index.md$2)"))
	return b
}

// BuildWebsite prepares the Hugo content tree: optionally runs
// `mdsmith fix` against srcDir so every <?catalog?>/<?include?>
// body is current, snapshots srcDir into dstDir via SyncDocs,
// then publishes the rule directory (internal/rules/index.md) as
// the browsable Rules section under dstDir and copies each
// per-rule README as its own website page. It is the canonical
// implementation behind the website sync — both local dev and the
// pages-deploy workflow call this rather than carrying the
// fix+sync sequence as inline shell (see
// docs/development/release-tooling.md: every workflow that needs
// runtime logic goes through this binary).
//
// The fix step shells out to `go run ./cmd/mdsmith` through the
// Runner seam, mirroring how BuildWheels invokes python; it
// expects the caller's working directory to be the repo root
// (true in CI and for the documented local invocation).
func (t *Toolkit) BuildWebsite(srcDir, dstDir string, runFix bool) error {
	if runFix {
		if err := t.runner.RunCommand("", "go", "run", "./cmd/mdsmith", "fix", srcDir); err != nil {
			return fmt.Errorf("mdsmith fix %s: %w", srcDir, err)
		}
	}
	// SyncDocs already contextualizes every failure path
	// (same-path guard, source not found, wipe/mkdir dst,
	// sync src -> dst), so it is returned unwrapped to keep
	// messages from doubling up.
	if err := t.SyncDocs(srcDir, dstDir); err != nil {
		return err
	}
	// The rule directory lives under internal/rules/, a sibling
	// of the docs/ source tree, not inside it — so SyncDocs never
	// sees it. Publish it as its own section after the docs
	// snapshot (SyncDocs wiped dstDir, so this must run last).
	rulesDir := filepath.Join(filepath.Dir(filepath.Clean(srcDir)), "internal", "rules")
	if err := t.syncRuleIndex(rulesDir, dstDir); err != nil {
		return err
	}
	return t.syncRulePages(rulesDir, dstDir)
}

// syncRuleIndex snapshots the rule directory index
// (rulesDir/index.md) into dstDir/rules/_index.md so the full MDS
// catalog is a browsable page on the site. The same docs→Hugo
// transforms SyncDocs applies (title lift, directive-marker strip)
// are applied here, then each rule-README relative link is
// rewritten to the corresponding local site page URL (e.g.
// `MDS001-line-length/README.md` → `MDS001-line-length/`) so the
// links stay on-site rather than bouncing to GitHub.
//
// A cascade front-matter block is injected so Hugo applies the
// `rule` layout type to all child pages, enabling category and
// status metadata to be displayed in the per-rule template.
//
// A missing rule index is not an error: callers that point
// BuildWebsite at a docs tree with no sibling internal/rules/
// (every BuildWebsite unit test, say) simply get no Rules section.
//
// index.md is located via ReadDir and skipped if it is a symlink,
// mirroring SyncDocs' symlink handling: following a link planted
// at internal/rules/index.md would let build-website publish an
// arbitrary runner file into the Hugo tree. A symlinked (or
// absent) index.md is treated the same as no rule index.
func (t *Toolkit) syncRuleIndex(rulesDir, dstDir string) error {
	entries, err := t.fs.ReadDir(rulesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read rule dir %s: %w", rulesDir, err)
	}
	var found bool
	for _, e := range entries {
		if e.Name() != "index.md" {
			continue
		}
		// Skip a symlinked index.md (DirEntry.Type reports
		// ModeSymlink for a link regardless of its target).
		if e.Type()&fs.ModeSymlink == 0 {
			found = true
		}
		break
	}
	if !found {
		return nil
	}
	// ReadDir already confirmed a non-symlink index.md, so a
	// ReadFile failure here is a real error (or a TOCTOU
	// disappearance) — wrap it rather than special-casing
	// not-exist.
	src := filepath.Join(rulesDir, "index.md")
	data, err := t.fs.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read rule index %s: %w", src, err)
	}
	data = transformMarkdown(data)
	// Rewrite `MDS001-line-length/README.md` → `MDS001-line-length/`
	// so links stay on-site rather than pointing at GitHub.
	data = ruleReadmeLink.ReplaceAll(data, []byte("]($1/)"))
	// Cascade the `rule` layout type to all child pages so Hugo
	// picks up the per-rule template for category/status display.
	data = injectFMField(data, "cascade:\n  type: rule")
	dstRulesDir := filepath.Join(dstDir, "rules")
	if err := t.fs.MkdirAll(dstRulesDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dstRulesDir, err)
	}
	dst := filepath.Join(dstRulesDir, "_index.md")
	if err := t.fs.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write rule index %s: %w", dst, err)
	}
	return nil
}

// syncRulePages copies each per-rule README
// (rulesDir/MDS###-name/README.md) into
// dstDir/rules/MDS###-name/index.md as a standalone Hugo page.
// The same docs→Hugo transforms applied by SyncDocs (title lift,
// directive-marker strip, Hugo shortcode escaping) are applied to
// each README. A `github_source` front-matter field recording the
// repo-relative path to the rule's source directory is injected so
// the rule layout template can link back to GitHub.
//
// Symlinked rule directories are skipped (same rationale as
// SyncDocs). Directories whose name does not start with "MDS" are
// skipped (guard against helper or fixture directories). A rule
// directory with no README.md is silently skipped rather than
// failing, so a partially-authored rule does not block the build.
//
// A missing rules directory is not an error — same as
// syncRuleIndex.
func (t *Toolkit) syncRulePages(rulesDir, dstDir string) error {
	entries, err := t.fs.ReadDir(rulesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read rule dir %s: %w", rulesDir, err)
	}
	dstRulesDir := filepath.Join(dstDir, "rules")
	for _, e := range entries {
		if e.Type()&fs.ModeSymlink != 0 || !e.IsDir() {
			continue
		}
		if !ruleDirName.MatchString(e.Name()) {
			continue
		}
		readmeSrc := filepath.Join(rulesDir, e.Name(), "README.md")
		data, err := t.fs.ReadFile(readmeSrc)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("read rule README %s: %w", readmeSrc, err)
		}
		data = transformMarkdown(data)
		// Rewrite inline cross-rule links (`../MDS021-include/README.md` →
		// `../MDS021-include/`) so they resolve to the sibling rule's
		// published page rather than an unpublished README.md.
		data = ruleReadmeLink.ReplaceAll(data, []byte("]($1/)"))
		// Rewrite reference-style link definitions to sibling rule
		// directories (e.g. `[mds027]: ../MDS027-.../README.md`) so the
		// definition URL resolves to the published page, not a raw README.
		data = ruleRefDefLink.ReplaceAll(data, []byte("$1/"))
		// Rewrite inline links to the docs/ tree to site-absolute paths.
		// Hugo serves each doc at /docs/path/ (no .md); the raw relative
		// ../../../docs/path/file.md would resolve to a 404 on the site.
		// An optional #anchor fragment is preserved after the trailing slash.
		data = repoDocsLink.ReplaceAll(data, []byte("](/docs/$1/$2)"))
		// Rewrite inline links to the plan/ tree (not published on the
		// site) to absolute GitHub blob URLs.
		data = repoPlanLink.ReplaceAll(data, []byte("]("+githubBlobBase+"plan/$1)"))
		// Rewrite reference-style link definitions to plan/ files.
		// ${1} (braced form) is required: Go's template engine would
		// otherwise parse "$1https" as a single group name, leaving the
		// expansion empty.
		data = repoPlanRefDef.ReplaceAll(data, []byte("${1}"+githubBlobBase+"plan/$2"))
		// Rewrite fixture references (`good/default.md`,
		// `bad/x.md`, `pattern/good/`) to the rule's GitHub source
		// tree URL. Fixtures live under the rule's source
		// directory but are not republished on the site.
		ruleSourceURL := ruleSourceTreeBase + e.Name() + "/"
		data = ruleFixtureLink.ReplaceAll(data, []byte("]("+ruleSourceURL+"$1)"))
		// Rewrite single-`../`-prefixed sibling references that
		// are NOT MDS rule pages (a sibling Go package, the
		// shared `proto.md`, etc.) to GitHub blob URLs under
		// internal/rules/. The non-MDS guard in the regex
		// preserves working cross-rule links like
		// `../MDS021-include/`.
		data = ruleSiblingNonMDSLink.ReplaceAll(data,
			[]byte("]("+githubBlobBase+"internal/rules/$1)"))
		// The `Implementation: [source](./)` meta line self-links the
		// generated page on the site; repoint it at the rule's
		// GitHub source directory.
		data = bytes.ReplaceAll(data,
			[]byte("[source](./)"),
			[]byte("[source]("+ruleSourceURL+")"))
		// Inject the repo-relative source path so the layout can
		// render a "View source on GitHub" link without hard-coding
		// the repo URL in the Go layer.
		sourcePath := "internal/rules/" + e.Name() + "/"
		data = injectFMField(data, "github_source: "+sourcePath)
		ruleDst := filepath.Join(dstRulesDir, e.Name())
		if err := t.fs.MkdirAll(ruleDst, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", ruleDst, err)
		}
		dst := filepath.Join(ruleDst, "index.md")
		if err := t.fs.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write rule page %s: %w", dst, err)
		}
	}
	return nil
}

// injectFMField inserts a YAML field block into a document's front
// matter. If the document has no front matter, one is synthesized.
// If the front matter is malformed (opening delimiter with no
// close), the document is returned byte-for-byte unchanged.
// field must be valid YAML without a trailing newline.
func injectFMField(data []byte, field string) []byte {
	fm, body, hasFM, ok := splitDocFrontMatter(string(data))
	if !ok {
		return data
	}
	if hasFM {
		return []byte("---\n" + strings.TrimRight(fm, "\n") + "\n" + field + "\n---\n" + body)
	}
	return []byte("---\n" + field + "\n---\n" + body)
}

// BuildWebsite delegates to a default-OS Toolkit (see Stamp).
func BuildWebsite(srcDir, dstDir string, runFix bool) error {
	return New().BuildWebsite(srcDir, dstDir, runFix)
}
