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
		// The `Implementation: [source](./)` meta line self-links the
		// generated page on the site; repoint it at the rule's
		// GitHub source directory.
		data = bytes.ReplaceAll(data,
			[]byte("[source](./)"),
			[]byte("[source]("+ruleSourceTreeBase+e.Name()+"/)"))
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
