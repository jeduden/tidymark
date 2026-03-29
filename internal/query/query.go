package query

import (
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// Matcher holds a pre-compiled CUE expression for matching
// against front matter maps.
type Matcher struct {
	schema cue.Value
	fields []string // top-level field names required by the expression
}

// Compile parses a CUE struct literal body and returns a
// Matcher. Returns an error if the expression is invalid.
func Compile(expr string) (*Matcher, error) {
	ctx := cuecontext.New()
	// Wrap the expression body in braces to form a struct literal.
	val := ctx.CompileString("{" + expr + "}")
	if err := val.Err(); err != nil {
		return nil, fmt.Errorf("invalid CUE expression: %w", err)
	}
	// Extract top-level field names so Match can require them in data.
	var fields []string
	iter, _ := val.Fields()
	for iter.Next() {
		fields = append(fields, iter.Selector().String())
	}
	return &Matcher{schema: val, fields: fields}, nil
}

// Match reports whether fm satisfies the compiled CUE expression.
// A nil or empty map never matches an expression that requires fields.
func (m *Matcher) Match(fm map[string]any) bool {
	if fm == nil {
		fm = map[string]any{}
	}
	// Require that every field in the expression exists in the data.
	// CUE structs are open by default, so unification alone would
	// accept data missing a field by filling it from the schema.
	for _, f := range m.fields {
		if _, ok := fm[f]; !ok {
			return false
		}
	}
	data, err := json.Marshal(fm)
	if err != nil {
		return false
	}
	dataVal := m.schema.Context().CompileBytes(data)
	if dataVal.Err() != nil {
		return false
	}
	merged := m.schema.Unify(dataVal)
	return merged.Validate(cue.Concrete(true)) == nil
}

// Match is a convenience function that compiles expr and tests fm
// in a single call. Returns an error only if the CUE expression is
// invalid.
func Match(expr string, fm map[string]any) (bool, error) {
	m, err := Compile(expr)
	if err != nil {
		return false, err
	}
	return m.Match(fm), nil
}
