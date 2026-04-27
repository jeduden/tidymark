package lint

// Severity indicates the severity level of a diagnostic.
type Severity string

// Severity levels.
const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

// Diagnostic represents a single lint finding.
type Diagnostic struct {
	File            string
	Line            int
	Column          int
	RuleID          string
	RuleName        string
	Severity        Severity
	Message         string
	SourceLines     []string // context lines around the diagnostic; empty if unavailable
	SourceStartLine int      // 1-based line number of first entry in SourceLines
	// Explanation, when non-nil, carries provenance information for the
	// rule that fired. It is populated only when the user passes
	// --explain on `check` or `fix`. Formatters render it as a one-line
	// trailer (text) or an `explanation` object (JSON).
	Explanation *Explanation
}

// Explanation carries the provenance attached to a diagnostic by the
// --explain flag. Source is the winning layer label
// ("default", "kinds.<name>", "overrides[i]", "front-matter override"),
// Kinds is the file's effective kind list at the time of the run, and
// LeafSources maps every leaf of the rule's final config to the layer
// that wrote it.
type Explanation struct {
	Rule        string
	Source      string
	Kinds       []string
	LeafSources map[string]string
}
