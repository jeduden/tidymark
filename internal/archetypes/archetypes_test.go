package archetypes

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fsWith(files map[string]string) fs.FS {
	m := fstest.MapFS{}
	for p, body := range files {
		m[p] = &fstest.MapFile{Data: []byte(body)}
	}
	return m
}

func TestResolver_ListDefaultRoot(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/story.md": "# ?",
		"archetypes/prd.md":   "# ?",
	})}
	entries := r.List()
	require.Len(t, entries, 2)
	assert.Equal(t, "prd", entries[0].Name)
	assert.Equal(t, "archetypes/prd.md", entries[0].Path)
	assert.Equal(t, "story", entries[1].Name)
}

func TestResolver_ListMultipleRootsEarlierShadows(t *testing.T) {
	r := &Resolver{
		Roots: []string{"custom", "archetypes"},
		FS: fsWith(map[string]string{
			"archetypes/prd.md":   "# default",
			"custom/prd.md":       "# custom",
			"archetypes/story.md": "# story",
		}),
	}
	entries := r.List()
	require.Len(t, entries, 2)
	assert.Equal(t, "prd", entries[0].Name)
	assert.Equal(t, "custom/prd.md", entries[0].Path)
	assert.Equal(t, "story", entries[1].Name)
	assert.Equal(t, "archetypes/story.md", entries[1].Path)
}

func TestResolver_ListSkipsNonMarkdownAndDirs(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/README.txt":    "notes",
		"archetypes/story.md":      "# story",
		"archetypes/sub/nested.md": "# nested",
	})}
	entries := r.List()
	require.Len(t, entries, 1)
	assert.Equal(t, "story", entries[0].Name)
}

func TestResolver_LookupReturnsEntry(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/story.md": "# story",
	})}
	e, err := r.Lookup("story")
	require.NoError(t, err)
	assert.Equal(t, "story", e.Name)
	assert.Equal(t, "archetypes/story.md", e.Path)
}

func TestResolver_LookupEmptyName(t *testing.T) {
	r := &Resolver{}
	_, err := r.Lookup("")
	require.Error(t, err)
}

func TestResolver_LookupMissingListsRoots(t *testing.T) {
	r := &Resolver{Roots: []string{"a", "b"}, FS: fsWith(map[string]string{})}
	_, err := r.Lookup("none")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no archetypes found")
	assert.Contains(t, err.Error(), "a, b")
}

func TestResolver_LookupMissingListsSiblings(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/story.md": "# story",
		"archetypes/prd.md":   "# prd",
	})}
	_, err := r.Lookup("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "available under roots")
	assert.Contains(t, err.Error(), "prd")
	assert.Contains(t, err.Error(), "story")
}

func TestResolver_Content(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/story.md": "# story body",
	})}
	b, err := r.Content("story")
	require.NoError(t, err)
	assert.Equal(t, "# story body", string(b))
}

func TestResolver_ContentMissing(t *testing.T) {
	r := &Resolver{}
	_, err := r.Content("x")
	require.Error(t, err)
}

func TestResolver_AbsPathJoinsRootDir(t *testing.T) {
	r := &Resolver{
		RootDir: "/home/me/proj",
		FS: fsWith(map[string]string{
			"archetypes/story.md": "# story",
		}),
	}
	p, err := r.AbsPath("story")
	require.NoError(t, err)
	assert.Equal(t, "/home/me/proj/archetypes/story.md", p)
}

func TestResolver_AbsPathNoRootDir(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/story.md": "# story",
	})}
	p, err := r.AbsPath("story")
	require.NoError(t, err)
	assert.Equal(t, "archetypes/story.md", p)
}

func TestResolver_AbsPathMissingName(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{})}
	_, err := r.AbsPath("missing")
	require.Error(t, err)
}

// statErrFS wraps an fs.FS and returns a non-ErrNotExist error from
// Stat for a specific path, exercising the unexpected-error branch of
// Lookup.
type statErrFS struct {
	fs      fs.FS
	errPath string
	err     error
}

func (s statErrFS) Open(name string) (fs.File, error) {
	return s.fs.Open(name)
}

func (s statErrFS) Stat(name string) (fs.FileInfo, error) {
	if name == s.errPath {
		return nil, s.err
	}
	return fs.Stat(s.fs, name)
}

func (s statErrFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(s.fs, name)
}

func TestResolver_LookupPropagatesUnexpectedStatError(t *testing.T) {
	boom := errors.New("io failure")
	r := &Resolver{FS: statErrFS{
		fs:      fsWith(map[string]string{"archetypes/dummy.md": "x"}),
		errPath: "archetypes/boom.md",
		err:     boom,
	}}
	_, err := r.Lookup("boom")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading archetype")
	assert.True(t, errors.Is(err, boom))
}

func TestDefaultRoot(t *testing.T) {
	assert.Equal(t, "archetypes", DefaultRoot)
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/x.md": "# x",
	})}
	entries := r.List()
	require.Len(t, entries, 1)
}

func TestResolver_ListFiltersReservedNames(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/README.md":       "docs",
		"archetypes/License.md":      "text",
		"archetypes/CONTRIBUTING.md": "text",
		"archetypes/_scratch.md":     "x",
		"archetypes/.hidden.md":      "x",
		"archetypes/story.md":        "# ?",
		"archetypes/prd.md":          "# ?",
	})}
	entries := r.List()
	require.Len(t, entries, 2)
	assert.Equal(t, "prd", entries[0].Name)
	assert.Equal(t, "story", entries[1].Name)
}

func TestResolver_LookupRejectsReservedNames(t *testing.T) {
	r := &Resolver{FS: fsWith(map[string]string{
		"archetypes/README.md": "docs",
		"archetypes/story.md":  "# ?",
	})}
	for _, name := range []string{"README", "readme", "_draft", ".hidden"} {
		_, err := r.Lookup(name)
		require.Errorf(t, err, "expected error for %q", name)
		assert.Contains(t, err.Error(), "unknown archetype")
	}
}

func TestIsArchetypeName(t *testing.T) {
	cases := map[string]bool{
		"story":              true,
		"prd":                true,
		"agent-def":          true,
		"Story":              true,
		"":                   false,
		"_draft":             false,
		".hidden":            false,
		"README":             false,
		"readme":             false,
		"ReadMe":             false,
		"License":            false,
		"CONTRIBUTING":       false,
		"codeowners":         false,
		"sub/story":          false,
		"a/../../etc/passwd": false,
		"..":                 false,
		"sub\\win":           false,
	}
	for in, want := range cases {
		assert.Equal(t, want, isArchetypeName(in),
			"isArchetypeName(%q)", in)
	}
}

func TestResolver_LookupRejectsPathInjection(t *testing.T) {
	dir := t.TempDir()
	// Place a file outside the archetype root that Lookup must not read.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "secret.md"), []byte("# secret"), 0o644))
	require.NoError(t, os.MkdirAll(
		filepath.Join(dir, "archetypes"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "archetypes", "story.md"),
		[]byte("# ?"), 0o644))

	r := &Resolver{RootDir: dir}
	for _, name := range []string{
		"../secret",
		"sub/story",
		"a/../../etc/passwd",
		"..",
	} {
		_, err := r.Lookup(name)
		require.Errorf(t, err, "expected rejection for %q", name)
		assert.Contains(t, err.Error(), "unknown archetype",
			"got: %v", err)
	}
	// Sanity: the real archetype still resolves.
	entry, err := r.Lookup("story")
	require.NoError(t, err)
	assert.Equal(t, "story", entry.Name)
}

func TestEffectiveRoots_DefaultsAndRootDirJoin(t *testing.T) {
	r := &Resolver{}
	assert.Equal(t, []string{"archetypes"}, r.EffectiveRoots())

	r2 := &Resolver{Roots: []string{"a", "b"}}
	assert.Equal(t, []string{"a", "b"}, r2.EffectiveRoots())

	r3 := &Resolver{RootDir: "/proj"}
	assert.Equal(t, []string{"/proj/archetypes"}, r3.EffectiveRoots())

	r4 := &Resolver{RootDir: "/proj", Roots: []string{"a", "b"}}
	assert.Equal(t, []string{"/proj/a", "/proj/b"}, r4.EffectiveRoots())
}

func TestValidateRoot(t *testing.T) {
	for _, tc := range []struct {
		root    string
		wantErr bool
	}{
		{"", true},
		{"   ", true},
		{"archetypes", false},
		{"./archetypes", false},
		{".", false},
		{"internal/archetypes", false},
		{"/abs", true},
		{"..", true},
		{"../foo", true},
		{"./../foo", true},
	} {
		err := ValidateRoot(tc.root)
		if tc.wantErr {
			assert.Errorf(t, err, "root=%q", tc.root)
		} else {
			assert.NoErrorf(t, err, "root=%q", tc.root)
		}
	}
}

func TestValidateRoots_StopsAtFirstError(t *testing.T) {
	err := ValidateRoots([]string{"ok", "../bad", "alsoOk"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "../bad")
}

func TestValidateRoots_EmptySliceOK(t *testing.T) {
	assert.NoError(t, ValidateRoots(nil))
}

// readDirErrFS returns a non-ErrNotExist error from ReadDir for a
// specific root, exercising ListWithErrors' error-surfacing branch.
type readDirErrFS struct {
	fs      fs.FS
	errRoot string
	err     error
}

func (s readDirErrFS) Open(name string) (fs.File, error) {
	return s.fs.Open(name)
}

func (s readDirErrFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == s.errRoot {
		return nil, s.err
	}
	return fs.ReadDir(s.fs, name)
}

func TestResolver_ListWithErrors_ReturnsNonNotExistErrors(t *testing.T) {
	boom := errors.New("io boom")
	r := &Resolver{FS: readDirErrFS{
		fs:      fsWith(map[string]string{"good/story.md": "# ?"}),
		errRoot: "bad",
		err:     boom,
	}, Roots: []string{"bad", "good"}}
	entries, errs := r.ListWithErrors()
	require.Len(t, entries, 1)
	assert.Equal(t, "story", entries[0].Name)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "reading archetype root")
	assert.True(t, errors.Is(errs[0], boom))
}

func TestResolver_ListWithErrors_SilentOnNotExist(t *testing.T) {
	// Missing root is fine — the resolver tolerates empty directories.
	r := &Resolver{Roots: []string{"missing"}, FS: fsWith(nil)}
	entries, errs := r.ListWithErrors()
	assert.Empty(t, entries)
	assert.Empty(t, errs)
}

func TestResolver_LookupSkipsDirectoryMatch(t *testing.T) {
	// A directory named "story.md" under the root must not be treated
	// as the archetype file.
	m := fstest.MapFS{
		"archetypes/story.md/placeholder": &fstest.MapFile{Data: []byte("x")},
	}
	r := &Resolver{FS: m}
	_, err := r.Lookup("story")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "unknown archetype")
}
