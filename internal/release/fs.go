package release

import (
	"io/fs"
	"os"
)

// FS is the small filesystem surface the release toolkit uses.
// Production paths use osFS (delegating to the real syscalls);
// tests inject a fault-injecting fake to exercise IO error
// branches that real filesystems don't reliably trigger
// (mid-pipeline mkdir failure, rename target on a non-directory,
// disk-full WriteFile, etc.).
//
// We intentionally keep this independent of stdlib's read-only
// `io/fs.FS`. The release toolkit needs Mkdir/Write/Rename
// alongside Read/Stat, and stdlib's interface only covers reads.
type FS interface {
	// Stat returns the FileInfo for the named file, mirroring
	// os.Stat.
	Stat(name string) (os.FileInfo, error)
	// ReadFile reads the named file, mirroring os.ReadFile.
	ReadFile(name string) ([]byte, error)
	// WriteFile writes data to the named file, mirroring
	// os.WriteFile.
	WriteFile(name string, data []byte, perm fs.FileMode) error
	// ReadDir reads the named directory, mirroring os.ReadDir.
	ReadDir(name string) ([]os.DirEntry, error)
	// MkdirAll creates name and any parents, mirroring
	// os.MkdirAll.
	MkdirAll(path string, perm fs.FileMode) error
	// MkdirTemp creates a new temporary directory, mirroring
	// os.MkdirTemp.
	MkdirTemp(dir, pattern string) (string, error)
	// Rename renames (moves) oldpath to newpath, mirroring
	// os.Rename.
	Rename(oldpath, newpath string) error
	// RemoveAll removes path and any children, mirroring
	// os.RemoveAll.
	RemoveAll(path string) error
}

// osFS is the production filesystem implementation. Each method
// delegates straight to the corresponding os.* call.
type osFS struct{}

func (osFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (osFS) ReadFile(name string) ([]byte, error)  { return os.ReadFile(name) }
func (osFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (osFS) ReadDir(name string) ([]os.DirEntry, error) { return os.ReadDir(name) }
func (osFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}
func (osFS) MkdirTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}
func (osFS) Rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }
func (osFS) RemoveAll(path string) error          { return os.RemoveAll(path) }

// Toolkit owns a configured FS and exposes the release helpers
// (Stamp, Check, BuildNpmPlatforms, BuildWheels) as methods.
// `New()` returns a Toolkit backed by the OS; tests can use
// `NewWithFS(fakeFS)` to drive error paths that the OS does not
// expose.
type Toolkit struct {
	fs FS
}

// New returns a Toolkit backed by the real OS filesystem.
func New() *Toolkit { return &Toolkit{fs: osFS{}} }

// NewWithFS returns a Toolkit backed by the supplied FS. Used
// by tests that need to inject IO failures.
func NewWithFS(fsys FS) *Toolkit { return &Toolkit{fs: fsys} }
