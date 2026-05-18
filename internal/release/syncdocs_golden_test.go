package release

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// reconcileDir holds one minimal Markdown fixture per
// reconcileDocForHugo branch. Each case is a directory with an `in`
// source and a `golden` expected output. Inputs are kept as small as
// possible — just enough bytes to drive the branch — and are checked
// in as files (not inline strings) the way rule fixtures are, so the
// strategic coverage is reviewable at a glance. The fixtures use no
// `.md` extension so `mdsmith check` does not lint deliberately
// malformed cases (the project ignore list cannot be extended
// without consent).
//
// The one-time byte-identical proof for the pkg/markdown migration
// (plan 163 AC4) was a full-docs-corpus snapshot captured from the
// pre-migration goldmark path; it passed unchanged post-migration at
// commit b150678. That snapshot was a migration artifact, not a
// permanent fixture — re-rendering all of docs/ churned ~22k lines
// of testdata on every docs edit. These minimal fixtures are the
// permanent guard.
const reconcileDir = "testdata/reconcile"

// TestReconcileDocForHugo pins reconcileDocForHugo byte-for-byte on
// each minimal branch fixture. Regenerate goldens with
// UPDATE_GOLDEN=1.
func TestReconcileDocForHugo(t *testing.T) {
	entries, err := os.ReadDir(reconcileDir)
	require.NoError(t, err)
	cases := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			in, rerr := os.ReadFile(filepath.Join(reconcileDir, name, "in"))
			require.NoError(t, rerr)
			got := reconcileDocForHugo(in)
			goldenPath := filepath.Join(reconcileDir, name, "golden")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				require.NoError(t, os.WriteFile(goldenPath, got, 0o644))
				t.Skip("golden regenerated")
			}
			want, rerr := os.ReadFile(goldenPath)
			require.NoError(t, rerr, "golden missing; run with UPDATE_GOLDEN=1")
			require.Equal(t, string(want), string(got),
				"reconcileDocForHugo output drifted for case %q", name)
		})
		cases++
	}
	require.Positive(t, cases, "expected reconcile fixture directories")
}
