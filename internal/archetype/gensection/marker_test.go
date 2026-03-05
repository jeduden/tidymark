package gensection

import "testing"

func TestIsRawStartMarker_Exact(t *testing.T) {
	if !IsRawStartMarker([]byte("<?catalog?>"), "catalog") {
		t.Error("expected match for <?catalog?>")
	}
}

func TestIsRawStartMarker_WithBody(t *testing.T) {
	if !IsRawStartMarker([]byte("<?catalog glob: plan/*.md"), "catalog") {
		t.Error("expected match for <?catalog with body")
	}
}

func TestIsRawStartMarker_WhitespacePrefix(t *testing.T) {
	if !IsRawStartMarker([]byte("  <?catalog?>"), "catalog") {
		t.Error("expected match with leading whitespace")
	}
}

func TestIsRawStartMarker_NameBoundary(t *testing.T) {
	if IsRawStartMarker([]byte("<?catalogue?>"), "catalog") {
		t.Error("should not match <?catalogue?> for name 'catalog'")
	}
}

func TestIsRawStartMarker_NoMatch(t *testing.T) {
	if IsRawStartMarker([]byte("some text"), "catalog") {
		t.Error("should not match plain text")
	}
}

func TestIsRawStartMarker_NameOnly(t *testing.T) {
	if !IsRawStartMarker([]byte("<?catalog"), "catalog") {
		t.Error("expected match for bare <?catalog")
	}
}

func TestIsRawStartMarker_TabAfterName(t *testing.T) {
	if !IsRawStartMarker([]byte("<?catalog\tglob: x"), "catalog") {
		t.Error("expected match with tab after name")
	}
}

func TestIsRawEndMarker_Exact(t *testing.T) {
	if !IsRawEndMarker([]byte("<?/catalog?>"), "catalog") {
		t.Error("expected match for <?/catalog?>")
	}
}

func TestIsRawEndMarker_WhitespacePrefix(t *testing.T) {
	if !IsRawEndMarker([]byte("  <?/catalog?>"), "catalog") {
		t.Error("expected match with leading whitespace")
	}
}

func TestIsRawEndMarker_TrailingContent(t *testing.T) {
	if IsRawEndMarker([]byte("<?/catalog?> extra"), "catalog") {
		t.Error("should not match end marker with trailing content")
	}
}

func TestIsRawEndMarker_NoMatch(t *testing.T) {
	if IsRawEndMarker([]byte("<?/include?>"), "catalog") {
		t.Error("should not match wrong name")
	}
}
