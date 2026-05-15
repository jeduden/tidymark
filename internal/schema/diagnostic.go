package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SchemaDiagnostic is the structured form of an MDS020 violation.
// Every CUE-error, structure-mismatch, filename-mismatch, and
// require-directive failure flows through this type so the
// rendered message names the field, shows the value, names the
// constraint, and (when applicable) suggests a fix.
//
// The type lives in the schema package because schema.Validate is
// the producer; internal/rules/requiredstructure is the sole
// consumer and would import this from anywhere else. Keeping the
// formatter alongside the producer avoids duplicating extractor
// logic across the two packages and sidesteps the import cycle
// that would result from defining it under internal/rules/.
//
// The `Schema` prefix on the name disambiguates from lint.Diagnostic
// at import sites where both packages are visible — plan 147 names
// the type explicitly, and renaming to schema.Diagnostic would
// require touching every caller for purely cosmetic reasons.
//
//nolint:revive // intentional name; see comment above.
type SchemaDiagnostic struct {
	// Field names the offending input. For front-matter CUE errors
	// it is the top-level key (or dotted path for nested values).
	// For structure failures it is the formatted heading
	// (e.g. "## Goal"). For filename failures it is "filename".
	Field string

	// Actual is the value the user wrote, rendered for display:
	// strings appear quoted, scalars raw. Empty when the user
	// supplied nothing concrete (e.g. a missing required section).
	Actual string

	// Expected describes the constraint in user vocabulary
	// (e.g. `one of: "open", "in-progress", "done"`). Empty when
	// the expectation is implied by Field — for instance, a
	// "missing required section" diagnostic does not repeat the
	// section name as the expected value.
	Expected string

	// Hint, when non-empty, points the reader at a likely fix
	// (typically the nearest valid literal or numeric bound).
	// Hints are best-effort; a noisy hint is worse than none, so
	// the extractor only fires on a small set of shapes.
	Hint string

	// SchemaRef names the schema source and (when known) the line
	// of the constraint, so the reader can locate the rule
	// without parsing the message. Examples: "plan/proto.md:4",
	// "inline kind schema".
	SchemaRef string
}

// Format renders the diagnostic as the two-line message described
// in plan 147: the first line carries field/actual/expected, an
// optional hint follows in parentheses on its own indented line,
// and the schema reference appears on a trailing line so it stays
// greppable without parsing the message body.
func (d SchemaDiagnostic) Format() string {
	var b strings.Builder
	b.WriteString(d.Field)
	if d.Actual != "" {
		b.WriteString(": got ")
		b.WriteString(d.Actual)
	}
	if d.Expected != "" {
		if d.Actual == "" {
			b.WriteString(": expected ")
		} else {
			b.WriteString(", expected ")
		}
		b.WriteString(d.Expected)
	}
	if d.Hint != "" {
		b.WriteString("\n  (")
		b.WriteString(d.Hint)
		b.WriteString(")")
	}
	if d.SchemaRef != "" {
		b.WriteString("\nschema: ")
		b.WriteString(d.SchemaRef)
	}
	return b.String()
}

// formatActual renders a JSON-decoded front-matter value for the
// "got" segment of the diagnostic. Strings are quoted with %q so
// the rendered message is unambiguous; numbers and bools pass
// through Sprintf's default formatting. An explicit YAML/JSON
// null surfaces as the literal `null` so callers can distinguish
// "field present but null" from "field absent" — the latter sets
// Actual to "<missing>" at the call site (see lookupFM's
// hasActual contract). Complex shapes (maps, slices) round-trip
// through json.Marshal so the rendered output is deterministic
// (Go's default map formatting iterates keys in undefined order,
// which would make the diagnostic message non-stable and break
// the deduplication key the caller derives from Actual).
func formatActual(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return fmt.Sprintf("%q", x)
	case bool, int, int64, float64:
		return fmt.Sprintf("%v", x)
	}
	if b, err := json.Marshal(v); err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", v)
}
