package release

import (
	"archive/zip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func haveCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func haveModule(t *testing.T, mod string) bool {
	t.Helper()
	if !haveCmd("python") && !haveCmd("python3") {
		return false
	}
	py := "python"
	if !haveCmd(py) {
		py = "python3"
	}
	cmd := exec.Command(py, "-c", "import "+mod)
	return cmd.Run() == nil
}

func stageBuildWheels(t *testing.T, version string) (string, string, string) {
	t.Helper()
	repo := projectRoot(t)
	root := t.TempDir()
	fixtureManifests(t, root)

	// Replace the minimal fixture pyproject.toml with the real one,
	// otherwise hatchling won't pick up the build configuration.
	realPy, err := os.ReadFile(filepath.Join(repo, "python", "pyproject.toml"))
	if err != nil {
		t.Fatalf("read pyproject.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "python", "pyproject.toml"), realPy, 0o644); err != nil {
		t.Fatalf("write pyproject.toml: %v", err)
	}
	// Hatchling needs the package source tree and a README.
	for _, p := range []string{"python/mdsmith/__init__.py", "python/mdsmith/__main__.py", "python/README.md"} {
		body, err := os.ReadFile(filepath.Join(repo, p))
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		dst := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
		}
		if err := os.WriteFile(dst, body, 0o644); err != nil {
			t.Fatalf("write %s: %v", dst, err)
		}
	}

	if _, stderr, err := runScript(t,
		filepath.Join(repo, "scripts", "set-version.sh"),
		version, "--root", root,
	); err != nil {
		t.Fatalf("set-version.sh: %v\nstderr: %s", err, stderr)
	}

	artifacts := filepath.Join(root, "artifacts")
	if err := os.MkdirAll(artifacts, 0o755); err != nil {
		t.Fatalf("mkdir artifacts: %v", err)
	}
	fakeArtifacts(t, artifacts)

	scriptsDir := filepath.Join(root, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	scriptSrc := filepath.Join(repo, "scripts", "build-wheels.sh")
	scriptDst := filepath.Join(scriptsDir, "build-wheels.sh")
	body, err := os.ReadFile(scriptSrc)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if err := os.WriteFile(scriptDst, body, 0o755); err != nil {
		t.Fatalf("copy script: %v", err)
	}

	return artifacts, scriptDst, filepath.Join(root, "wheels")
}

func readZipMember(t *testing.T, whlPath, member string) string {
	t.Helper()
	r, err := zip.OpenReader(whlPath)
	if err != nil {
		t.Fatalf("open %s: %v", whlPath, err)
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, member) {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open zip member %s: %v", f.Name, err)
			}
			body, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				t.Fatalf("read zip member %s: %v", f.Name, err)
			}
			return string(body)
		}
	}
	return ""
}

// zipHasFile reports whether the wheel contains the named entry.
func zipHasFile(t *testing.T, whlPath, name string) bool {
	t.Helper()
	r, err := zip.OpenReader(whlPath)
	if err != nil {
		t.Fatalf("open %s: %v", whlPath, err)
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if f.Name == name {
			return true
		}
	}
	return false
}

// wheelCase pins a single platform-tagged wheel: the substring that
// uniquely identifies its filename and the platform tag substring its
// dist-info/WHEEL metadata must carry after retagging.
type wheelCase struct {
	uniqueFilenameSubstr string
	tagInWheelMetadata   string
	binName              string
}

func wheelCases() []wheelCase {
	return []wheelCase{
		{"x86_64.manylinux", "manylinux_2_17_x86_64", "mdsmith"},
		{"aarch64.manylinux", "manylinux_2_17_aarch64", "mdsmith"},
		{"macosx_11_0_x86_64", "macosx_11_0_x86_64", "mdsmith"},
		{"macosx_11_0_arm64", "macosx_11_0_arm64", "mdsmith"},
		{"win_amd64", "win_amd64", "mdsmith.exe"},
	}
}

func assertWheel(t *testing.T, out string, entries []os.DirEntry, c wheelCase) {
	t.Helper()
	var match string
	for _, e := range entries {
		if strings.Contains(e.Name(), c.uniqueFilenameSubstr) {
			match = e.Name()
			break
		}
	}
	if match == "" {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("no wheel matched filename containing %q in %v", c.uniqueFilenameSubstr, names)
		return
	}
	whl := filepath.Join(out, match)
	meta := readZipMember(t, whl, "/WHEEL")
	if !strings.Contains(meta, c.tagInWheelMetadata) {
		t.Errorf("%s: WHEEL metadata missing platform tag %q\n%s", whl, c.tagInWheelMetadata, meta)
	}
	if strings.Contains(meta, "py3-none-any") {
		t.Errorf("%s: WHEEL metadata still claims py3-none-any\n%s", whl, meta)
	}
	if !zipHasFile(t, whl, "mdsmith/_bin/"+c.binName) {
		t.Errorf("%s: bundled binary mdsmith/_bin/%s missing", whl, c.binName)
	}
}

// TestBuildWheelsLayout shells out to scripts/build-wheels.sh and
// asserts (a) one wheel per platform tag, (b) the WHEEL metadata
// inside each wheel claims the same platform tag the filename does,
// and (c) the bundled binary is present under mdsmith/_bin/.
func TestBuildWheelsLayout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("build-wheels.sh requires a POSIX shell")
	}
	if !haveCmd("python") && !haveCmd("python3") {
		t.Skip("python is required to exercise build-wheels.sh")
	}
	if !haveModule(t, "build") || !haveModule(t, "wheel") || !haveModule(t, "hatchling") {
		t.Skip("python -m build, python -m wheel, and hatchling are required")
	}

	const ver = "7.8.9"
	artifacts, script, out := stageBuildWheels(t, ver)

	if _, stderr, err := runScript(t, script, artifacts, out); err != nil {
		t.Fatalf("build-wheels.sh failed: %v\nstderr: %s", err, stderr)
	}

	cases := wheelCases()
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatalf("readdir %s: %v", out, err)
	}
	if len(entries) != len(cases) {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected %d wheels, got %d: %v", len(cases), len(entries), names)
	}
	for _, c := range cases {
		assertWheel(t, out, entries, c)
	}
}
