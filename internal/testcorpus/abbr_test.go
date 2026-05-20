package testcorpus

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAbbrHeavy_NonEmpty pins the corpus invariant the rest of the
// suite assumes — at least one paragraph and every entry non-empty.
// A truncated or empty corpus would silently weaken benchmarks that
// run "for each paragraph" because the loop body would never fire.
func TestAbbrHeavy_NonEmpty(t *testing.T) {
	require.NotEmpty(t, AbbrHeavy)
	for i, s := range AbbrHeavy {
		assert.NotEmptyf(t, strings.TrimSpace(s),
			"AbbrHeavy[%d] must not be empty", i)
	}
}

// TestAbbrHeavyParagraph_JoinsCorpus pins the join contract: the
// returned paragraph contains every entry of AbbrHeavy in order, with
// a single space between entries. A regression here would change what
// MDS024's BenchmarkRule_MDS024 actually measures, so this anchors
// the fixture even though the function body is trivial.
func TestAbbrHeavyParagraph_JoinsCorpus(t *testing.T) {
	got := AbbrHeavyParagraph()
	for _, s := range AbbrHeavy {
		assert.Containsf(t, got, s,
			"AbbrHeavyParagraph must include corpus entry %q", s)
	}
	// Joined with a single space — not a newline (paragraph stays
	// a single paragraph) and not multiple spaces.
	want := strings.Join(AbbrHeavy, " ")
	assert.Equal(t, want, got,
		"AbbrHeavyParagraph must join entries with a single space")
}

// TestJoinWithSpace_EmptyCorpus is the explicit zero case: an empty
// slice must produce an empty paragraph. Tests joinWithSpace
// directly so the exported AbbrHeavy variable is not mutated —
// `go test ./...` runs packages in parallel, and other consumers of
// AbbrHeavy (paragraph-structure's alloc-budget gate, mdtext's
// abbr-heavy benchmark) read it concurrently.
func TestJoinWithSpace_EmptyCorpus(t *testing.T) {
	assert.Equal(t, "", joinWithSpace(nil),
		"nil input must produce an empty paragraph")
	assert.Equal(t, "", joinWithSpace([]string{}),
		"empty slice must produce an empty paragraph")
}

// TestJoinWithSpace_SingleEntry pins the single-entry branch: no
// leading or trailing space.
func TestJoinWithSpace_SingleEntry(t *testing.T) {
	assert.Equal(t, "lonely",
		joinWithSpace([]string{"lonely"}),
		"single-entry slice must not add a separator")
}
