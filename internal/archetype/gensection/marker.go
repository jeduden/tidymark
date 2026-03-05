package gensection

import "bytes"

// IsRawStartMarker reports whether line begins a processing-instruction
// start marker for the given directive name (e.g. "<?catalog ...").
// It operates on raw bytes for use before AST parsing (e.g. merge driver).
func IsRawStartMarker(line []byte, name string) bool {
	trimmed := bytes.TrimSpace(line)
	prefix := []byte("<?" + name)

	if !bytes.HasPrefix(trimmed, prefix) {
		return false
	}

	rest := trimmed[len(prefix):]
	if len(rest) == 0 {
		return true
	}

	// Require a name boundary: whitespace or "?>".
	if len(rest) >= 2 && rest[0] == '?' && rest[1] == '>' {
		return true
	}

	return rest[0] == ' ' || rest[0] == '\t'
}

// IsRawEndMarker reports whether line is a processing-instruction
// end marker for the given directive name (e.g. "<?/catalog?>").
// It operates on raw bytes for use before AST parsing (e.g. merge driver).
func IsRawEndMarker(line []byte, name string) bool {
	return bytes.Equal(bytes.TrimSpace(line), []byte("<?/"+name+"?>"))
}
