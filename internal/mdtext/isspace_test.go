package mdtext_test

import (
	"testing"
	"unicode"

	"github.com/jeduden/mdsmith/internal/mdtext"
)

// IsSpace must be a behaviour-exact, faster substitute for
// unicode.IsSpace: CountWords / CountSentences and the MDS024
// cheapBounds guard all swap one for the other on the check hot path
// (plan 175), so any divergence would silently change word and
// sentence counts. The sweep covers the whole rune domain — plus two
// negative sentinels — so the ASCII fast path is proven equivalent,
// not spot-checked.
func TestIsSpaceMatchesUnicodeExhaustive(t *testing.T) {
	for r := rune(-2); r <= unicode.MaxRune; r++ {
		if got, want := mdtext.IsSpace(r), unicode.IsSpace(r); got != want {
			t.Fatalf("IsSpace(%#U)=%v, unicode.IsSpace=%v", r, got, want)
		}
	}
}

func TestIsSpaceSpotChecks(t *testing.T) {
	for _, c := range []struct {
		r    rune
		want bool
	}{
		{' ', true}, {'\t', true}, {'\n', true}, {'\v', true},
		{'\f', true}, {'\r', true},
		{'a', false}, {'0', false}, {'-', false}, {0x00, false},
		{0x1F, false}, {0x21, false},
		{0x85, true},    // NEL — Latin-1, non-ASCII
		{0xA0, true},    // NBSP
		{0x2000, true},  // EN QUAD
		{0x3000, true},  // IDEOGRAPHIC SPACE
		{0x200B, false}, // ZERO WIDTH SPACE is not whitespace
		{'界', false},
	} {
		if got := mdtext.IsSpace(c.r); got != c.want {
			t.Errorf("IsSpace(%#U)=%v, want %v", c.r, got, c.want)
		}
	}
}
