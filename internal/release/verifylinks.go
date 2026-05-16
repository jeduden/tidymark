package release

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// VerifyWebsiteLinks runs a fixed set of probes against the
// rendered HTML produced by `hugo --minify ...`. Each probe
// targets a behavior the render-link hook is responsible for:
// `.md` → permalink resolution, `index.md` → section URL,
// no `README.md` hrefs in rule pages, javascript:/data:
// hrefs neutralized by html/template, and the baseURL prefix
// appearing on site-absolute hrefs when one was supplied at
// build time. None of these are visible to the synced-tree
// mdsmith check (it walks the markdown filesystem
// pre-render), so without these probes a regression in the
// render-link hook ships silently.
//
// htmlDir is the Hugo output root (`public/` under
// website/). baseURL is the URL Hugo was built with; the
// path portion (e.g. `/mdsmith` for project-pages, empty
// for root deploys) is treated as the expected path prefix
// on every site-absolute href. Each probe's regex accepts
// both `href="value"` (Hugo's default) and `href=value`
// (--minify), so the function is robust whether or not the
// caller passed `--minify`.
//
// Probes fail closed with a single returned error naming
// the probe and the file that failed; subsequent probes
// are not run.
func VerifyWebsiteLinks(htmlDir, baseURL string) error {
	prefix, err := pathPrefixFromBaseURL(baseURL)
	if err != nil {
		return fmt.Errorf("parse base url: %w", err)
	}
	for _, p := range websiteLinkProbes(prefix) {
		if err := runWebsiteLinkProbe(htmlDir, p); err != nil {
			return err
		}
	}
	return nil
}

// linkProbe describes one rendered-HTML assertion. Either
// wantMatch or wantNoMatch is set, never both; whichever is
// non-nil drives the check at that path. recursive=true
// scans every .html file under path; recursive=false reads
// the single file at path.
type linkProbe struct {
	name        string
	path        string
	wantMatch   *regexp.Regexp
	wantNoMatch *regexp.Regexp
	recursive   bool
}

// websiteLinkProbes returns the probes that VerifyWebsiteLinks
// runs against the rendered output. Built fresh per call so
// the captured prefix lands in each regex; prefix is the
// site-path component of the build's baseURL (`/mdsmith`
// for a project-pages deploy, "" for a root deploy).
func websiteLinkProbes(prefix string) []linkProbe {
	q := regexp.QuoteMeta
	hrefEq := `href="?` // allow both quoted (default) and unquoted (minified) emission
	return []linkProbe{
		{
			name: "sibling .md resolves to target permalink",
			path: "docs/development/merge-queue/index.html",
			wantMatch: regexp.MustCompile(
				hrefEq + q(prefix) + `/docs/development/pr-fixup-workflow/`),
		},
		{
			name: "index.md drop resolves to section URL on leaf page",
			path: "docs/development/architecture-audit/index.html",
			wantMatch: regexp.MustCompile(
				hrefEq + q(prefix) + `/docs/development/architecture/`),
		},
		{
			name: "no README.md hrefs leaked into rule pages",
			path: "docs/rules",
			wantNoMatch: regexp.MustCompile(
				`href=(?:"[^"]*README\.md|[^ ">]*README\.md)`),
			recursive: true,
		},
		{
			name:        "no javascript: hrefs reached rendered HTML",
			path:        ".",
			wantNoMatch: regexp.MustCompile(`href=(?:"javascript:|javascript:)`),
			recursive:   true,
		},
		{
			name:        "no data: hrefs reached rendered HTML",
			path:        ".",
			wantNoMatch: regexp.MustCompile(`href=(?:"data:|data:)`),
			recursive:   true,
		},
	}
}

// runWebsiteLinkProbe evaluates one probe. For
// non-recursive probes it reads a single file. For
// recursive probes it walks the subtree and either reports
// the first file matching wantNoMatch or aggregates all
// files; wantMatch is not used with recursive=true today.
func runWebsiteLinkProbe(root string, p linkProbe) error {
	target := filepath.Join(root, p.path)
	if !p.recursive {
		data, err := readHTMLFile(target)
		if err != nil {
			return fmt.Errorf("verify %q: %w", p.name, err)
		}
		if p.wantMatch != nil && !p.wantMatch.Match(data) {
			return fmt.Errorf("verify %q: no match for %s in %s",
				p.name, p.wantMatch, target)
		}
		if p.wantNoMatch != nil && p.wantNoMatch.Match(data) {
			return fmt.Errorf("verify %q: unwanted match for %s in %s",
				p.name, p.wantNoMatch, target)
		}
		return nil
	}
	if p.wantNoMatch == nil {
		return fmt.Errorf("verify %q: recursive probe needs wantNoMatch", p.name)
	}
	return filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		data, err := readHTMLFile(path)
		if err != nil {
			return err
		}
		if p.wantNoMatch.Match(data) {
			return fmt.Errorf("verify %q: unwanted match for %s in %s",
				p.name, p.wantNoMatch, path)
		}
		return nil
	})
}

// readHTMLFile reads an HTML file via the package's fs seam
// for symmetry with the other release subcommands.
// filepath.WalkDir bypasses the seam, so this routes through
// it consistently for unit-test injection points.
func readHTMLFile(path string) ([]byte, error) {
	t := New()
	data, err := t.fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("rendered HTML not found: %s", path)
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

// pathPrefixFromBaseURL returns the URL path component of
// baseURL, with a trailing slash trimmed so the caller can
// concatenate `/docs/...` without producing `//docs/...`.
// An empty baseURL or a root-deploy baseURL
// (`https://example.com/`) yields "". A project-pages
// baseURL (`https://example.com/repo/`) yields `/repo`.
func pathPrefixFromBaseURL(baseURL string) (string, error) {
	if baseURL == "" {
		return "", nil
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(u.Path, "/"), nil
}
