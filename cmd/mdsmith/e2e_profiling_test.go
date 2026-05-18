package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The env-gated profiler is a process-boundary feature: the built
// binary must honor MDSMITH_CPUPROFILE / MDSMITH_MEMPROFILE with no
// CLI flag. Unit tests in internal/profiling cover the logic; this
// e2e only verifies main wires it in (the `defer profiling.Start()()`
// in run()), which no lower layer can prove.
func TestE2E_ProfilingEnvHook(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(md, []byte("# Title\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cpu := filepath.Join(dir, "cpu.out")
	mem := filepath.Join(dir, "mem.out")

	cmd := exec.Command(binaryPath, "check", md)
	cmd.Dir = dir
	cmd.Env = append(envWithCoverDir(coverDir),
		"MDSMITH_CPUPROFILE="+cpu,
		"MDSMITH_MEMPROFILE="+mem,
	)
	// `check` exits non-zero on lint findings; the profile must be
	// written regardless, so the exit code is intentionally ignored.
	_ = cmd.Run()

	for _, p := range []string{cpu, mem} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatalf("profile %s not written: %v", filepath.Base(p), err)
		}
		if fi.Size() == 0 {
			t.Fatalf("profile %s is empty", filepath.Base(p))
		}
	}
}
