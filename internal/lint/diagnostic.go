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
	File     string
	Line     int
	Column   int
	RuleID   string
	RuleName string
	Severity Severity
	Message  string
}
