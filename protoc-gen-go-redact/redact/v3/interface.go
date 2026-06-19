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
