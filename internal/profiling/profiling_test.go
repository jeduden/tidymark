package profiling

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
)

func TestReport(t *testing.T) {
	var b bytes.Buffer
	report(&b, "cpu", errors.New("boom"))
	if got := b.String(); got != "profiling: cpu profile: boom\n" {
		t.Fatalf("unexpected report output: %q", got)
	}
}

func TestBeginCPU(t *testing.T) {
	t.Run("success returns an open file", func(t *testing.T) {
		var diag bytes.Buffer
		f := beginCPU(filepath.Join(t.TempDir(), "c.out"), &diag)
		if f == nil {
			t.Fatal("expected a file on success")
		}
		pprof.StopCPUProfile() // caller's responsibility per the contract
		_ = f.Close()
		if diag.Len() != 0 {
			t.Fatalf("unexpected diagnostic: %q", diag.String())
		}
	})

	t.Run("create error returns nil and reports", func(t *testing.T) {
		var diag bytes.Buffer
		if f := beginCPU(filepath.Join(t.TempDir(), "no", "c.out"), &diag); f != nil {
			t.Fatal("expected nil on create error")
		}
		if !strings.Contains(diag.String(), "cpu profile") {
			t.Fatalf("missing diagnostic: %q", diag.String())
		}
	})

	t.Run("already running returns nil and reports", func(t *testing.T) {
		live := beginCPU(filepath.Join(t.TempDir(), "live.out"), &bytes.Buffer{})
		if live == nil {
			t.Fatal("setup: first profile must start")
		}
		defer func() { pprof.StopCPUProfile(); _ = live.Close() }()
		var diag bytes.Buffer
		if f := beginCPU(filepath.Join(t.TempDir(), "second.out"), &diag); f != nil {
			t.Fatal("expected nil while a profile is already active")
		}
		if !strings.Contains(diag.String(), "cpu profile") {
			t.Fatalf("missing diagnostic: %q", diag.String())
		}
	})
}

func TestWriteHeap(t *testing.T) {
	t.Run("success writes a non-empty profile", func(t *testing.T) {
		var diag bytes.Buffer
		p := filepath.Join(t.TempDir(), "m.out")
		writeHeap(p, &diag)
		fi, err := os.Stat(p)
		if err != nil || fi.Size() == 0 {
			t.Fatalf("heap profile missing or empty: err=%v", err)
		}
		if diag.Len() != 0 {
			t.Fatalf("unexpected diagnostic: %q", diag.String())
		}
	})

	t.Run("create error reports", func(t *testing.T) {
		var diag bytes.Buffer
		writeHeap(filepath.Join(t.TempDir(), "no", "m.out"), &diag)
		if !strings.Contains(diag.String(), "mem profile") {
			t.Fatalf("missing diagnostic: %q", diag.String())
		}
	})
}

func sizeOf(t *testing.T, path string) int64 {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return fi.Size()
}

func TestStart_NoEnv_IsNoOp(t *testing.T) {
	var diag bytes.Buffer
	stop := start("", "", &diag)
	if stop == nil {
		t.Fatal("stop must never be nil")
	}
	stop() // must not panic and must write nothing
	if diag.Len() != 0 {
		t.Fatalf("expected no diagnostics, got %q", diag.String())
	}
}

func TestStart_MemProfile_WritesNonEmpty(t *testing.T) {
	mem := filepath.Join(t.TempDir(), "mem.out")
	var diag bytes.Buffer
	stop := start("", mem, &diag)
	sink := make([]byte, 1<<20) // force a live allocation
	_ = sink
	stop()
	if got := sizeOf(t, mem); got == 0 {
		t.Fatalf("heap profile is empty")
	}
	if diag.Len() != 0 {
		t.Fatalf("unexpected diagnostics: %q", diag.String())
	}
}

func TestStart_CPUProfile_WritesNonEmpty(t *testing.T) {
	cpu := filepath.Join(t.TempDir(), "cpu.out")
	var diag bytes.Buffer
	stop := start(cpu, "", &diag)
	// Burn a little CPU so the profile has at least a header.
	x := 0
	for i := 0; i < 5_000_000; i++ {
		x += i % 7
	}
	_ = x
	stop()
	if got := sizeOf(t, cpu); got == 0 {
		t.Fatalf("cpu profile is empty")
	}
	if diag.Len() != 0 {
		t.Fatalf("unexpected diagnostics: %q", diag.String())
	}
}

func TestStart_CPUAlreadyRunning_ReportsAndContinues(t *testing.T) {
	// pprof allows only one active CPU profile per process, so a
	// second Start while the first is running drives the
	// StartCPUProfile error branch deterministically.
	first := start(filepath.Join(t.TempDir(), "first.out"), "", &bytes.Buffer{})
	defer first()

	cpu2 := filepath.Join(t.TempDir(), "second.out")
	var diag bytes.Buffer
	stop := start(cpu2, "", &diag)
	stop() // no second profile was started; safe no-op
	if !strings.Contains(diag.String(), "cpu profile") {
		t.Fatalf("expected cpu-profile diagnostic, got %q", diag.String())
	}
}

func TestStart_CPUCreateError_ReportsAndContinues(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "nope", "cpu.out") // missing dir
	var diag bytes.Buffer
	stop := start(bad, "", &diag)
	stop() // no cpu profiling was started; must be a safe no-op
	if !strings.Contains(diag.String(), "cpu profile") {
		t.Fatalf("expected cpu-profile diagnostic, got %q", diag.String())
	}
}

func TestStart_MemCreateError_ReportsOnStop(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "nope", "mem.out")
	var diag bytes.Buffer
	stop := start("", bad, &diag)
	stop()
	if !strings.Contains(diag.String(), "mem profile") {
		t.Fatalf("expected mem-profile diagnostic, got %q", diag.String())
	}
}

func TestStart_EnvWrapper_NonNil(t *testing.T) {
	t.Setenv("MDSMITH_CPUPROFILE", "")
	t.Setenv("MDSMITH_MEMPROFILE", "")
	stop := Start()
	if stop == nil {
		t.Fatal("Start must return a non-nil stop")
	}
	stop()
}
