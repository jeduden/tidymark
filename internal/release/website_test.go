package release

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWebsite_RunsFixThenSync(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")
	writeFile(t, filepath.Join(src, "top.md"), "top body\n")
	rec := &recordingRunner{}

	require.NoError(t, NewWithDeps(osFS{}, rec).BuildWebsite(src, dst, true))

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "go", rec.calls[0].name)
	assert.Equal(t, []string{"run", "./cmd/mdsmith", "fix", src}, rec.calls[0].args)
	assertFile(t, filepath.Join(dst, "top.md"), "top body\n")
}

func TestBuildWebsite_NoFixSkipsRunner(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")
	writeFile(t, filepath.Join(src, "top.md"), "top body\n")
	rec := &recordingRunner{}

	require.NoError(t, NewWithDeps(osFS{}, rec).BuildWebsite(src, dst, false))

	assert.Empty(t, rec.calls, "no-fix must not invoke the runner")
	assertFile(t, filepath.Join(dst, "top.md"), "top body\n")
}

func TestBuildWebsite_FixFailureWraps(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "top.md"), "top body\n")

	err := NewWithDeps(osFS{}, &fakeRunner{failOnCall: 1}).
		BuildWebsite(src, filepath.Join(t.TempDir(), "out"), true)

	require.Error(t, err)
	assert.ErrorIs(t, err, errInjected)
	assert.Contains(t, err.Error(), "mdsmith fix")
}

func TestBuildWebsite_SyncErrorSurfacedNotDoubleWrapped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "x.md"), "x\n")

	// recordingRunner succeeds on fix; src==dst trips the
	// SyncDocs overlap guard. BuildWebsite must surface that
	// error unwrapped — SyncDocs already contextualizes it,
	// so there must be no duplicated prefix.
	err := NewWithDeps(osFS{}, &recordingRunner{}).BuildWebsite(dir, dir, true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "same path")
	assert.NotContains(t, err.Error(), "sync ", "no redundant build-website wrap")
}

// TestBuildWebsite_SyncErrorNotDoubleWrapped is the regression
// for the duplicated `sync a -> b: sync a -> b:` prefix:
// SyncDocs already wraps the syncDocsDir failure with the
// `sync <src> -> <dst>:` prefix, so BuildWebsite must not add
// its own — the prefix must appear exactly once.
func TestBuildWebsite_SyncErrorNotDoubleWrapped(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "x.md"), "x\n")
	ff := newFakeFS()
	ff.failOnReadDirCall = 1

	err := NewWithDeps(ff, &recordingRunner{}).
		BuildWebsite(src, filepath.Join(t.TempDir(), "out"), false)

	require.Error(t, err)
	assert.ErrorIs(t, err, errInjected)
	assert.Contains(t, err.Error(), "read dir")
	assert.Equal(t, 1, strings.Count(err.Error(), "sync "),
		"the sync prefix must appear exactly once")
}
