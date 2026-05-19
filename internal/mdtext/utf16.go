package mdtext

import (
	"unicode/utf16"
	"unicode/utf8"
)

// NonNegativeUTF16RuneLen wraps utf16.RuneLen so its negative
// "invalid code point" return cannot decrement a caller's
// running UTF-16 unit total. utf8.DecodeRune already maps
// invalid bytes to RuneError (U+FFFD, width 1), so in practice
// utf16.RuneLen never returns a negative for runes decoded from
// real input; the guard is defensive against a future Go change
// that weakens that invariant. A negative width means the rune
// is outside [0, MaxRune] or is a surrogate, both of which take
// one UTF-16 unit when serialized as RuneError.
func NonNegativeUTF16RuneLen(r rune) int {
	if w := utf16.RuneLen(r); w >= 0 {
		return w
	}
	return 1
}

// UTF16FromByteOffset returns the UTF-16 code-unit offset that
// corresponds to UTF-8 byte offset byteOff within line. The
// result is clamped to [0, total UTF-16 length of line] so
// callers cannot receive a negative or past-end position even
// when given a malformed byte column.
func UTF16FromByteOffset(line []byte, byteOff int) int {
	if byteOff <= 0 {
		return 0
	}
	if byteOff > len(line) {
		byteOff = len(line)
	}
	units := 0
	for i := 0; i < byteOff; {
		r, size := utf8.DecodeRune(line[i:])
		units += NonNegativeUTF16RuneLen(r)
		i += size
	}
	return units
}

// UTF16ToByteOffset returns the byte offset in line at the given
// UTF-16 code-unit count. Offsets past the line's end clamp to
// len(line) so a defensive guard upstream still sees an in-range
// value. A target that lands inside a surrogate pair rounds up
// to the next codepoint boundary.
func UTF16ToByteOffset(line []byte, target int) int {
	if target <= 0 {
		return 0
	}
	units := 0
	for i := 0; i < len(line); {
		if units >= target {
			return i
		}
		r, size := utf8.DecodeRune(line[i:])
		units += NonNegativeUTF16RuneLen(r)
		i += size
	}
	return len(line)
}
