package mdtext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNonNegativeUTF16RuneLen(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 1, NonNegativeUTF16RuneLen('a'))
	assert.Equal(t, 2, NonNegativeUTF16RuneLen('😀'))
	// A lone surrogate is an invalid scalar value: utf16.RuneLen
	// returns -1, and the guard floors that at 1.
	assert.Equal(t, 1, NonNegativeUTF16RuneLen(rune(0xD800)))
}

func TestUTF16FromByteOffset(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, UTF16FromByteOffset([]byte("abc"), 0), "zero offset")
	assert.Equal(t, 0, UTF16FromByteOffset([]byte("abc"), -3), "negative clamps to 0")
	assert.Equal(t, 3, UTF16FromByteOffset([]byte("abc"), 99), "past end clamps to length")
	// A non-BMP rune (\U0001F600 — 😀) is encoded as 4 UTF-8 bytes
	// and 2 UTF-16 code units. The trailing 'x' is one of each.
	line := []byte("😀x")
	assert.Equal(t, 0, UTF16FromByteOffset(line, 0))
	assert.Equal(t, 2, UTF16FromByteOffset(line, 4)) // after the emoji
	assert.Equal(t, 3, UTF16FromByteOffset(line, 5)) // after the 'x'
}

func TestUTF16FromByteOffsetInvalidRunes(t *testing.T) {
	t.Parallel()
	// `string([]rune{0xD800, 'a'})` produces a UTF-8 sequence
	// containing utf8.RuneError; the running total must stay
	// non-negative even when utf16.RuneLen returns -1 for the
	// offending rune.
	line := []byte(string([]rune{0xD800, 'a'}))
	assert.Positive(t, UTF16FromByteOffset(line, len(line)))
}

func TestUTF16ToByteOffset(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, UTF16ToByteOffset([]byte("abc"), 0))
	assert.Equal(t, 0, UTF16ToByteOffset([]byte("abc"), -1))
	assert.Equal(t, 2, UTF16ToByteOffset([]byte("abc"), 2))
	assert.Equal(t, 3, UTF16ToByteOffset([]byte("abc"), 9)) // clamp past end
	// "é" is one UTF-16 unit but two UTF-8 bytes; "𝄞" is a surrogate
	// pair (two UTF-16 units, four bytes).
	row := []byte("é𝄞z")
	assert.Equal(t, 2, UTF16ToByteOffset(row, 1)) // past "é"
	assert.Equal(t, 6, UTF16ToByteOffset(row, 3)) // past the surrogate pair
}
