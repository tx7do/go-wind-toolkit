// Package redact provides interfaces and methods to help implement redaction.
package redact

import "context"

// Redactor provides the method to be used to Redact
type Redactor interface {
	Redact()
}

// Apply will apply redaction on the input, if it implements Redactor.
// It will do nothing if the object does not implement the interface.
func Apply(in interface{}) {
	if red, ok := in.(Redactor); ok {
		red.Redact()
	}
}

// Bypass provides a way to bypass the internal marked methods to be ran
// through clients
type Bypass interface {
	CheckInternal(ctx context.Context) bool
}

// Wrapper helps to implement Bypass
type Wrapper func(ctx context.Context) bool

// CheckInternal for Wrapper
func (w Wrapper) CheckInternal(ctx context.Context) bool { return w(ctx) }

// Falsy is the nil implementation for Bypass
var Falsy = Wrapper(func(_ context.Context) bool {
	return false
})

// --- Custom Redactor Registry ---

// CustomRedactor is a user-supplied function that transforms a string value.
type CustomRedactor func(string) string

var customRedactors = map[string]CustomRedactor{}

// RegisterCustomRedactor registers a named redactor function.
// Call this at init time (or early in program startup) so that generated
// code can reference it via the `custom` field rule.
func RegisterCustomRedactor(name string, fn CustomRedactor) {
	customRedactors[name] = fn
}

// ApplyCustomRedactor looks up and invokes a registered redactor by name.
// If no redactor is registered with the given name, the value is returned
// unchanged.
func ApplyCustomRedactor(name string, value string) string {
	if fn, ok := customRedactors[name]; ok {
		return fn(value)
	}
	return value
}
