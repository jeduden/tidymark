package linkgraph

import (
	"net/url"
	"path"
	"strings"
)

// ResolveRelTarget joins srcFile's directory with linkPath and
// returns the workspace-relative result. Absolute paths and ones
// that escape the workspace root after normalization return the
// empty string — callers must treat "" as "no in-workspace target"
// rather than as a valid path.
//
// The function is strict about its inputs:
//
//   - srcFile must already be workspace-relative (no leading `/`,
//     no drive letter, no UNC `\\` prefix). Callers that hold
//     absolute paths must convert them first; otherwise a
//     `../../etc/passwd`-style linkPath could escape via
//     path.Join's absolute-path semantics.
//   - linkPath has both `\` and `/` translated to `/` before joining
//     so a Windows-authored `sub\x.md` resolves the same way on Linux.
//     (filepath.ToSlash is OS-dependent and a no-op on POSIX hosts;
//     this helper translates explicitly via strings.ReplaceAll.)
//   - Absolute inputs are rejected up-front; path.Join of two relative
//     paths never produces an absolute result, so the only escape vector
//     is a leading `../` in the cleaned output (caught below).
func ResolveRelTarget(srcFile, linkPath string) string {
	srcFile = strings.ReplaceAll(srcFile, `\`, `/`)
	linkPath = strings.ReplaceAll(linkPath, `\`, `/`)
	if path.IsAbs(srcFile) || path.IsAbs(linkPath) {
		return ""
	}
	if isDriveOrUNC(srcFile) || isDriveOrUNC(linkPath) {
		return ""
	}
	dir := path.Dir(srcFile)
	cleaned := path.Clean(path.Join(dir, linkPath))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

// isDriveOrUNC reports whether p starts with a Windows drive letter
// (e.g. `C:`) or a UNC prefix (`//server`). Used by ResolveRelTarget
// to refuse non-relative inputs even when running on POSIX hosts,
// where filepath.IsAbs wouldn't flag them.
func isDriveOrUNC(p string) bool {
	if len(p) >= 2 && p[1] == ':' {
		c := p[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	return strings.HasPrefix(p, "//")
}

// DecodeAnchor URL-decodes raw and returns the decoded form. On
// decode failure (e.g. a stray `%` not followed by hex) the input
// is returned unchanged.
//
// Use NormalizeAnchor when comparing against CollectAnchors output —
// NormalizeAnchor combines DecodeAnchor with Slugify so callers see
// one normalised form. DecodeAnchor is exposed for code paths that
// store the decoded anchor as a distinct field from the slugified
// one (the LSP locator), where the slugify step happens later.
func DecodeAnchor(raw string) string {
	if d, err := url.PathUnescape(raw); err == nil {
		return d
	}
	return raw
}
