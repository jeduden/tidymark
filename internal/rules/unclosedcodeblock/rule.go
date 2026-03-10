package unclosedcodeblock

import (
	"fmt"
	"strings"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
)

func init() {
	rule.Register(&Rule{})
}

// Rule reports fenced code blocks that are opened but never closed.
type Rule struct{}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS031" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "unclosed-code-block" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "code" }

// Check implements rule.Rule.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	var diags []lint.Diagnostic
	lines := strings.Split(string(f.Source), "\n")

	var inFence bool
	var fenceChar byte
	var fenceLen int
	var fenceLine int

	for i, line := range lines {
		if inFence {
			if isClosingFence(line, fenceChar, fenceLen) {
				inFence = false
			}
			continue
		}
		char, length, ok := openingFence(line)
		if ok {
			inFence = true
			fenceChar = char
			fenceLen = length
			fenceLine = i + 1 // 1-based
		}
	}

	if inFence {
		diags = append(diags, lint.Diagnostic{
			File:     f.Path,
			Line:     fenceLine,
			Column:   1,
			RuleID:   r.ID(),
			RuleName: r.Name(),
			Severity: lint.Error,
			Message:  fmt.Sprintf("fenced code block opened at line %d is never closed", fenceLine),
		})
	}

	return diags
}

// openingFence checks if a line is a fenced code block opening.
// Returns the fence character, fence length, and whether it matched.
// Per CommonMark: 0-3 spaces of indentation, then 3+ backticks or tildes.
// Backtick fences must not contain backticks in the info string.
func openingFence(line string) (byte, int, bool) {
	// Strip up to 3 leading spaces.
	trimmed := line
	spaces := 0
	for spaces < 3 && spaces < len(trimmed) && trimmed[spaces] == ' ' {
		spaces++
	}
	trimmed = trimmed[spaces:]

	if len(trimmed) < 3 {
		return 0, 0, false
	}

	ch := trimmed[0]
	if ch != '`' && ch != '~' {
		return 0, 0, false
	}

	count := 0
	for count < len(trimmed) && trimmed[count] == ch {
		count++
	}
	if count < 3 {
		return 0, 0, false
	}

	// Backtick fences: info string must not contain backticks.
	if ch == '`' && strings.ContainsRune(trimmed[count:], '`') {
		return 0, 0, false
	}

	return ch, count, true
}

// isClosingFence checks if a line closes a fenced code block.
// Per CommonMark: 0-3 spaces, then at least fenceLen of the same character,
// followed only by optional spaces.
func isClosingFence(line string, fenceChar byte, fenceLen int) bool {
	trimmed := line
	spaces := 0
	for spaces < 3 && spaces < len(trimmed) && trimmed[spaces] == ' ' {
		spaces++
	}
	trimmed = trimmed[spaces:]

	if len(trimmed) < fenceLen {
		return false
	}

	count := 0
	for count < len(trimmed) && trimmed[count] == fenceChar {
		count++
	}
	if count < fenceLen {
		return false
	}

	// After the fence characters, only spaces are allowed.
	return strings.TrimSpace(trimmed[count:]) == ""
}
