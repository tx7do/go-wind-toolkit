package main

import (
	"fmt"

	pgs "github.com/lyft/protoc-gen-star/v2"
	"google.golang.org/grpc/codes"

	"github.com/menta2k/protoc-gen-redact/v3/redact/v3"
)

// ErrorContext provides additional context for errors
type ErrorContext struct {
	Location string
	Field    string
	Type     string
	Rule     string
	Reason   string
}

// Error returns a formatted error message with context
func (e ErrorContext) Error() string {
	if e.Location != "" && e.Field != "" {
		return fmt.Sprintf("[%s.%s] %s", e.Location, e.Field, e.Reason)
	}
	if e.Location != "" {
		return fmt.Sprintf("[%s] %s", e.Location, e.Reason)
	}
	return e.Reason
}

// ValidationError represents a validation error with detailed context
type ValidationError struct {
	Entity   string
	Expected string
	Got      string
	Hint     string
}

// Error returns a formatted validation error message
func (v ValidationError) Error() string {
	msg := fmt.Sprintf("Validation failed for %s", v.Entity)
	if v.Expected != "" {
		msg += fmt.Sprintf(": expected %s", v.Expected)
	}
	if v.Got != "" {
		msg += fmt.Sprintf(", got %s", v.Got)
	}
	if v.Hint != "" {
		msg += fmt.Sprintf(" (hint: %s)", v.Hint)
	}
	return msg
}

// must wraps error checking with improved error messages
func (m *Module) must(ok bool, err error) bool {
	if err != nil {
		m.Fail(err)
	}
	return ok
}

// validateField performs comprehensive field validation
func (m *Module) validateField(field pgs.Field) error {
	if field == nil {
		return fmt.Errorf("field is nil")
	}

	typ := field.Type()
	if typ == nil {
		return fmt.Errorf("field %s has nil type", field.Name())
	}

	return nil
}

// validateMessage performs comprehensive message validation
func (m *Module) validateMessage(msg pgs.Message) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	// Check for conflicting options
	ignore := false
	toNil := false
	toEmpty := false

	m.must(msg.Extension(redact.E_Ignored, &ignore))
	m.must(msg.Extension(redact.E_Nil, &toNil))
	m.must(msg.Extension(redact.E_Empty, &toEmpty))

	conflictCount := 0
	if ignore {
		conflictCount++
	}
	if toNil {
		conflictCount++
	}
	if toEmpty {
		conflictCount++
	}

	if conflictCount > 1 {
		return ValidationError{
			Entity:   fmt.Sprintf("message %s", msg.FullyQualifiedName()),
			Expected: "at most one of (redact.ignored), (redact.nil), or (redact.empty)",
			Got:      fmt.Sprintf("multiple options set (ignored=%v, nil=%v, empty=%v)", ignore, toNil, toEmpty),
			Hint:     "these options are mutually exclusive",
		}
	}

	return nil
}

// validateService performs comprehensive service validation
func (m *Module) validateService(srv pgs.Service) error {
	if srv == nil {
		return fmt.Errorf("service is nil")
	}

	// Validate internal service status code
	srvCode := uint32(codes.PermissionDenied)
	if m.must(srv.Extension(redact.E_InternalServiceCode, &srvCode)) {
		if err := m.validateStatusCode(srvCode, srv.FullyQualifiedName()); err != nil {
			return err
		}
	}

	return nil
}

// validateMethod performs comprehensive method validation
func (m *Module) validateMethod(meth pgs.Method) error {
	if meth == nil {
		return fmt.Errorf("method is nil")
	}

	if meth.Input() == nil {
		return fmt.Errorf("method %s has nil input", meth.Name())
	}

	if meth.Output() == nil {
		return fmt.Errorf("method %s has nil output", meth.Name())
	}

	// Validate internal method status code
	methCode := uint32(codes.PermissionDenied)
	if m.must(meth.Extension(redact.E_InternalMethodCode, &methCode)) {
		if err := m.validateStatusCode(methCode, meth.FullyQualifiedName()); err != nil {
			return err
		}
	}

	return nil
}

// validateStatusCode validates a gRPC status code
func (m *Module) validateStatusCode(code uint32, location string) error {
	if code > uint32(codes.Unauthenticated) { // 16
		return ValidationError{
			Entity:   fmt.Sprintf("status code in %s", location),
			Expected: "valid gRPC status code (0-16)",
			Got:      fmt.Sprintf("%d", code),
			Hint:     "see https://grpc.io/docs/guides/status-codes/ for valid codes",
		}
	}
	return nil
}

// validateTypeMatch validates that a field type matches a rule type
func (m *Module) validateTypeMatch(
	field pgs.Field,
	ruleType pgs.ProtoType,
	ruleLabel pgs.ProtoLabel,
) error {
	fieldType := field.Type()

	// Check type match
	if ruleType != 0 && ruleType != fieldType.ProtoType() {
		return ValidationError{
			Entity:   fmt.Sprintf("field %s", field.FullyQualifiedName()),
			Expected: fmt.Sprintf("rule for type %s", fieldType.ProtoType()),
			Got:      fmt.Sprintf("rule for type %s", ruleType),
			Hint:     fmt.Sprintf("use %s instead", ToCustomRule(fieldType.ProtoType(), fieldType.ProtoLabel())),
		}
	}

	// Check label match for repeated fields
	if fieldType.ProtoLabel() == pgs.Repeated && ruleLabel != pgs.Repeated {
		return ValidationError{
			Entity:   fmt.Sprintf("repeated field %s", field.FullyQualifiedName()),
			Expected: "(redact.custom).element.*",
			Got:      "non-repeated rule",
			Hint:     "repeated fields require element rules",
		}
	}

	return nil
}

// failWithInvalidType generates an error for type mismatch with helpful suggestions
func (m *Module) failWithInvalidType(field pgs.Field) {
	typ := field.Type()
	want := ToCustomRule(typ.ProtoType(), typ.ProtoLabel())

	err := ValidationError{
		Entity:   field.FullyQualifiedName(),
		Expected: want,
		Got:      "incompatible redaction rule type",
		Hint:     fmt.Sprintf("ensure the rule type matches the field type (%s)", typ.ProtoType()),
	}

	m.Fail(err)
}

// failWithContext fails with additional contextual information
func (m *Module) failWithContext(field pgs.Field, reason string) {
	err := ErrorContext{
		Location: field.Message().FullyQualifiedName(),
		Field:    field.Name().String(),
		Type:     field.Type().ProtoType().String(),
		Reason:   reason,
	}
	m.Fail(err)
}

// failWithNestedError fails when nested element rules are incorrectly used
func (m *Module) failWithNestedError(field pgs.Field) {
	err := ErrorContext{
		Location: field.Message().FullyQualifiedName(),
		Field:    field.Name().String(),
		Type:     field.Type().ProtoType().String(),
		Reason:   "nested element.item.element... is not supported - maximum nesting depth is 1",
	}
	m.Failf("%s\n\nHint: Use either:\n  - (redact.custom).element.nested for iteration\n  - (redact.custom).element.item.* for custom item values\n  - (redact.custom).element.empty for empty list", err.Error())
}

// validateFile performs file-level validation
func (m *Module) validateFile(file pgs.File) error {
	if file == nil {
		return fmt.Errorf("file is nil")
	}

	if file.Package() == nil {
		return fmt.Errorf("file %s has no package", file.Name())
	}

	return nil
}

// validateRules validates FieldRules for correctness
func (m *Module) validateRules(rules *redact.FieldRules, field pgs.Field) error {
	if rules == nil {
		return nil // No rules is valid
	}

	if rules.Values == nil {
		return ValidationError{
			Entity:   field.FullyQualifiedName(),
			Expected: "redaction rule with values",
			Got:      "empty rule",
			Hint:     "define a value for the custom redaction rule",
		}
	}

	// Validate message rules
	if msgRule, ok := rules.Values.(*redact.FieldRules_Message); ok {
		if msgRule.Message == nil {
			return ValidationError{
				Entity:   field.FullyQualifiedName(),
				Expected: "message rule definition",
				Got:      "nil message rule",
				Hint:     "use (redact.custom).message.nil, .empty, or .skip",
			}
		}
	}

	// Validate element rules
	if elemRule, ok := rules.Values.(*redact.FieldRules_Element); ok {
		if elemRule.Element == nil {
			return ValidationError{
				Entity:   field.FullyQualifiedName(),
				Expected: "element rule definition",
				Got:      "nil element rule",
				Hint:     "use (redact.custom).element.nested, .empty, or .item.*",
			}
		}

		// Check for invalid nested element rules
		if elemRule.Element.Item != nil && elemRule.Element.Item.Values != nil {
			if _, ok := elemRule.Element.Item.Values.(*redact.FieldRules_Element); ok {
				return ValidationError{
					Entity:   field.FullyQualifiedName(),
					Expected: "single-level element nesting",
					Got:      "element.item.element",
					Hint:     "nested element rules are not supported",
				}
			}
		}
	}

	return nil
}

// recoverFromPanic recovers from panics and converts them to errors
func (m *Module) recoverFromPanic(context string) {
	if r := recover(); r != nil {
		m.Failf("Panic in %s: %v", context, r)
	}
}

// validateImportPath validates an import path
func (m *Module) validateImportPath(path string) error {
	if path == "" {
		return fmt.Errorf("import path is empty")
	}

	// Basic validation - can be extended
	if len(path) > 1000 {
		return fmt.Errorf("import path too long: %d characters", len(path))
	}

	return nil
}

// validatePackageName validates a package name
func (m *Module) validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("package name is empty")
	}

	// Check for invalid characters (basic validation)
	for i, c := range name {
		if i == 0 && c >= '0' && c <= '9' {
			return ValidationError{
				Entity:   "package name",
				Expected: "identifier starting with letter or underscore",
				Got:      fmt.Sprintf("name starting with digit: %s", name),
				Hint:     "package names cannot start with numbers",
			}
		}
	}

	return nil
}
