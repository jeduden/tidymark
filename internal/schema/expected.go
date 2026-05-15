package schema

import (
	"math"
	"strconv"
	"strings"
)

// RenderExpected converts a raw CUE constraint expression into a
// user-facing "expected" string. It recognises the common shapes
// listed in plan 147 — string disjunctions, regex matchers,
// numeric ranges, non-empty strings, and bool — and falls back to
// the verbatim expression when nothing matches. The fallback is
// deliberately the raw CUE so the user can still copy/paste into
// the schema; an over-eager translation that drops type
// information would be worse than the literal expression.
func RenderExpected(expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return expr
	}
	if rendered, ok := renderStringDisjunction(expr); ok {
		return rendered
	}
	if rendered, ok := renderRegex(expr); ok {
		return rendered
	}
	if rendered, ok := renderIntRange(expr); ok {
		return rendered
	}
	if rendered, ok := renderNonEmptyString(expr); ok {
		return rendered
	}
	if expr == "bool" {
		return "true or false"
	}
	return expr
}

// renderStringDisjunction matches `"a" | "b" | "c"` and renders
// it as `one of: "a", "b", "c"`. Disjunctions mixing strings with
// other types fall through to the raw expression so the user is
// not misled into thinking only the string branches are allowed.
func renderStringDisjunction(expr string) (string, bool) {
	parts := splitTopLevel(expr, '|')
	if len(parts) < 2 {
		return "", false
	}
	literals := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		// Strip a leading `*` (CUE default marker) before checking
		// for a string literal; the default is a UX hint, not a
		// new shape.
		s = strings.TrimPrefix(s, "*")
		if !isQuotedString(s) {
			return "", false
		}
		literals = append(literals, s)
	}
	return "one of: " + strings.Join(literals, ", "), true
}

// renderRegex matches `=~"<pattern>"` (with optional surrounding
// `string &`) and renders it as `string matching <pattern>`.
// Bare regex matchers without a `string &` cover the common
// proto.md shape; constraints that further restrict the type
// fall through to the raw expression.
func renderRegex(expr string) (string, bool) {
	body := expr
	body = strings.TrimSpace(body)
	body = strings.TrimPrefix(body, "string &")
	body = strings.TrimSpace(body)
	if !strings.HasPrefix(body, "=~") {
		return "", false
	}
	pattern := strings.TrimSpace(strings.TrimPrefix(body, "=~"))
	if !isQuotedString(pattern) {
		return "", false
	}
	unquoted, err := strconv.Unquote(pattern)
	if err != nil {
		return "", false
	}
	return "string matching " + unquoted, true
}

// intRangeBounds captures the parsed lower/upper bounds of an
// `int & ...` constraint, along with whether each remains
// exclusive after the overflow-safe `+1`/`-1` inclusive shift.
type intRangeBounds struct {
	lo, hi                   string
	loExclusive, hiExclusive bool
}

// renderIntRange matches `int & >=N & <=M`, `int & >N & <M`, or
// any combination of the two operators in either order and
// renders it as `int between N and M`. Half-open ranges
// (`int & >=N`) render as `int >= N` so the reader sees the
// bound; fully unbounded ints fall through to the raw expression.
func renderIntRange(expr string) (string, bool) {
	parts := splitTopLevel(expr, '&')
	if len(parts) < 2 {
		return "", false
	}
	bounds, hasInt, ok := parseIntRangeBounds(parts)
	if !ok || !hasInt {
		return "", false
	}
	bounds.tryConvertExclusive()
	return bounds.render()
}

// parseIntRangeBounds walks the &-separated parts and pulls out
// the `int` marker plus any `>`, `>=`, `<`, `<=` bounds. A part
// that doesn't fit the small grammar aborts the rendering by
// returning ok=false so the caller falls back to the raw CUE
// expression rather than rendering a partial constraint.
func parseIntRangeBounds(parts []string) (intRangeBounds, bool, bool) {
	var b intRangeBounds
	hasInt := false
	loInclusive, hiInclusive := false, false
	for _, p := range parts {
		s := strings.TrimSpace(p)
		switch {
		case s == "int":
			hasInt = true
		case strings.HasPrefix(s, ">="):
			b.lo = strings.TrimSpace(strings.TrimPrefix(s, ">="))
			loInclusive = true
		case strings.HasPrefix(s, ">"):
			b.lo = strings.TrimSpace(strings.TrimPrefix(s, ">"))
		case strings.HasPrefix(s, "<="):
			b.hi = strings.TrimSpace(strings.TrimPrefix(s, "<="))
			hiInclusive = true
		case strings.HasPrefix(s, "<"):
			b.hi = strings.TrimSpace(strings.TrimPrefix(s, "<"))
		default:
			return intRangeBounds{}, false, false
		}
	}
	b.loExclusive = !loInclusive && b.lo != ""
	b.hiExclusive = !hiInclusive && b.hi != ""
	return b, hasInt, true
}

// tryConvertExclusive turns `> N` into `>= N+1` (and `< N` into
// `<= N-1`) when N is an integer literal and the increment stays
// inside int's range. Bounds that would overflow keep their
// exclusive form so the rendered message preserves the original
// semantics — `int & >MaxInt` is unsatisfiable, but rendering it
// as `int >= MaxInt` would suggest MaxInt itself satisfied the
// constraint. CUE permits non-integer comparison targets, but
// our schemas almost always use integer literals.
func (b *intRangeBounds) tryConvertExclusive() {
	if b.loExclusive {
		if n, err := strconv.Atoi(b.lo); err == nil && n < math.MaxInt {
			b.lo = strconv.Itoa(n + 1)
			b.loExclusive = false
		}
	}
	if b.hiExclusive {
		if n, err := strconv.Atoi(b.hi); err == nil && n > math.MinInt {
			b.hi = strconv.Itoa(n - 1)
			b.hiExclusive = false
		}
	}
}

// render produces the final user-facing string from the parsed
// bounds: `int between N and M` for closed two-sided ranges,
// `int > N and <= M` (or similar) when one side stays exclusive,
// and `int >= N` / `int <= N` for half-open ranges.
func (b *intRangeBounds) render() (string, bool) {
	switch {
	case b.lo != "" && b.hi != "" && !b.loExclusive && !b.hiExclusive:
		return "int between " + b.lo + " and " + b.hi, true
	case b.lo != "" && b.hi != "":
		return "int " + lowerOp(b.loExclusive) + " " + b.lo +
			" and " + upperOp(b.hiExclusive) + " " + b.hi, true
	case b.lo != "":
		return "int " + lowerOp(b.loExclusive) + " " + b.lo, true
	case b.hi != "":
		return "int " + upperOp(b.hiExclusive) + " " + b.hi, true
	}
	return "", false
}

func lowerOp(exclusive bool) string {
	if exclusive {
		return ">"
	}
	return ">="
}

func upperOp(exclusive bool) string {
	if exclusive {
		return "<"
	}
	return "<="
}

// renderNonEmptyString matches `string & != ""` (in either
// operand order) and renders it as `non-empty string`.
func renderNonEmptyString(expr string) (string, bool) {
	parts := splitTopLevel(expr, '&')
	if len(parts) != 2 {
		return "", false
	}
	hasString, hasNonEmpty := false, false
	for _, p := range parts {
		s := strings.TrimSpace(p)
		switch s {
		case "string":
			hasString = true
		case `!= ""`, `!=""`:
			hasNonEmpty = true
		}
	}
	if hasString && hasNonEmpty {
		return "non-empty string", true
	}
	return "", false
}

// splitTopLevel splits expr on every occurrence of sep that is
// not inside double quotes or parentheses. CUE expressions
// occasionally embed `|` inside a regex string or a parenthesised
// subexpression; a naive strings.Split would mangle those.
func splitTopLevel(expr string, sep byte) []string {
	var out []string
	depth := 0
	inString := false
	start := 0
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		switch {
		case c == '\\' && i+1 < len(expr):
			i++ // skip escaped char
		case c == '"':
			inString = !inString
		case !inString && (c == '(' || c == '['):
			depth++
		case !inString && (c == ')' || c == ']'):
			depth--
		case !inString && depth == 0 && c == sep:
			out = append(out, expr[start:i])
			start = i + 1
		}
	}
	out = append(out, expr[start:])
	return out
}

// isQuotedString reports whether s begins and ends with `"`.
// CUE single-quote string literals are rare in our schemas; we
// accept only the canonical double-quoted form so the extractor
// is unambiguous.
func isQuotedString(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}
