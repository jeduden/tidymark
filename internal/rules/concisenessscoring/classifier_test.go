package concisenessscoring

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeduden/mdsmith/internal/lint"
)

type stubClassifier struct {
	risk  float64
	delay time.Duration
}

func (s *stubClassifier) ModelID() string {
	return "stub"
}

func (s *stubClassifier) Version() string {
	return "test"
}

func (s *stubClassifier) DefaultThreshold() float64 {
	return 0.60
}

func (s *stubClassifier) Predict(_ paragraphSignals) (float64, error) {
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	return s.risk, nil
}

func TestLoadEmbeddedClassifier(t *testing.T) {
	clf, err := loadEmbeddedClassifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clf.ModelID() != "cue-linear-v1" {
		t.Fatalf("expected cue-linear-v1, got %q", clf.ModelID())
	}
	if clf.Version() == "" {
		t.Fatal("expected non-empty model version")
	}
	if clf.DefaultThreshold() <= 0 || clf.DefaultThreshold() > 1 {
		t.Fatalf(
			"expected threshold in (0,1], got %.2f",
			clf.DefaultThreshold(),
		)
	}
}

func TestCheck_ClassifierModeUsesExternalModel(t *testing.T) {
	model := []byte(`{
  "model_id": "test-linear-v1",
  "version": "2026-02-16",
  "threshold": 0.60,
  "features": ["bias"],
  "weights": {"bias": -9.0}
}`)
	path, checksum := writeModelFile(t, model)

	src := []byte(verboseParagraph() + "\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{
		MinScore:            0.95,
		MinWords:            20,
		Mode:                "classifier",
		ClassifierModelPath: path,
		ClassifierChecksum:  checksum,
	}

	diags := r.Check(f)
	if len(diags) != 0 {
		t.Fatalf("expected classifier path to suppress diagnostic, got %d", len(diags))
	}
}

func TestCheck_ClassifierChecksumMismatchFallsBack(t *testing.T) {
	model := []byte(`{
  "model_id": "test-linear-v1",
  "version": "2026-02-16",
  "threshold": 0.60,
  "features": ["bias"],
  "weights": {"bias": -9.0}
}`)
	path, _ := writeModelFile(t, model)

	src := []byte(verboseParagraph() + "\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{
		MinScore:            0.95,
		MinWords:            20,
		Mode:                "classifier",
		ClassifierModelPath: path,
		ClassifierChecksum:  stringsOf("0", 64),
	}

	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected fallback heuristic diagnostic, got %d", len(diags))
	}
}

func TestCheck_ClassifierTimeoutFallsBack(t *testing.T) {
	src := []byte(verboseParagraph() + "\n")
	f, err := lint.NewFile("test.md", src)
	if err != nil {
		t.Fatal(err)
	}

	r := &Rule{
		MinScore:            0.95,
		MinWords:            20,
		Mode:                "classifier",
		Threshold:           0.60,
		ClassifierTimeoutMS: 1,
		classifier: &stubClassifier{
			risk:  0.05,
			delay: 10 * time.Millisecond,
		},
	}

	diags := r.Check(f)
	if len(diags) != 1 {
		t.Fatalf("expected timeout fallback diagnostic, got %d", len(diags))
	}
}

func TestApplySettings_InvalidMode(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{"mode": "gpu"})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestApplySettings_ClassifierPathRequiresChecksum(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"classifier-model-path": "/tmp/model.json",
	})
	if err == nil {
		t.Fatal("expected error when path is set without checksum")
	}
}

func TestApplySettings_ClassifierChecksumRequiresPath(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"classifier-checksum": stringsOf("a", 64),
	})
	if err == nil {
		t.Fatal("expected error when checksum is set without path")
	}
}

func TestApplySettings_InvalidClassifierChecksum(t *testing.T) {
	r := &Rule{}
	err := r.ApplySettings(map[string]any{
		"classifier-checksum": "xyz",
	})
	if err == nil {
		t.Fatal("expected error for invalid checksum")
	}
}

func writeModelFile(t *testing.T, data []byte) (string, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "model.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing model: %v", err)
	}

	sum := sha256.Sum256(data)
	return path, hex.EncodeToString(sum[:])
}

func stringsOf(ch string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += ch
	}
	return out
}
