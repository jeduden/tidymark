package rule

import "reflect"

// CloneRule creates a deep copy of a rule. If the rule implements
// Configurable, the clone is produced by creating a new zero-value
// instance and applying the original's DefaultSettings. Otherwise
// it falls back to a reflect-based shallow copy of the struct.
func CloneRule(r Rule) Rule {
	if c, ok := r.(Configurable); ok {
		// Create a new zero-value instance of the same concrete type.
		rv := reflect.ValueOf(r)
		if rv.Kind() == reflect.Ptr {
			newPtr := reflect.New(rv.Elem().Type())
			clone := newPtr.Interface().(Rule)
			if cc, ok := clone.(Configurable); ok {
				_ = cc.ApplySettings(c.DefaultSettings())
			}
			return clone
		}
	}

	// Fallback: reflect-based copy for non-Configurable rules.
	rv := reflect.ValueOf(r)
	if rv.Kind() == reflect.Ptr {
		newPtr := reflect.New(rv.Elem().Type())
		newPtr.Elem().Set(rv.Elem())
		return newPtr.Interface().(Rule)
	}

	// Value type — already a copy.
	return r
}
