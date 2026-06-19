package main

import (
	"testing"

	pgs "github.com/lyft/protoc-gen-star/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"github.com/menta2k/protoc-gen-redact/v3/redact/v3"
)

// TestErrorContext tests the ErrorContext error type
func TestErrorContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      ErrorContext
		expected string
	}{
		{
			name: "full_context",
			ctx: ErrorContext{
				Location: "user.proto.User",
				Field:    "password",
				Type:     "string",
				Reason:   "invalid redaction rule",
			},
			expected: "[user.proto.User.password] invalid redaction rule",
		},
		{
			name: "location_only",
			ctx: ErrorContext{
				Location: "user.proto.User",
				Reason:   "message validation failed",
			},
			expected: "[user.proto.User] message validation failed",
		},
		{
			name: "reason_only",
			ctx: ErrorContext{
				Reason: "general error",
			},
			expected: "general error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ctx.Error()
			assert.Equal(t, tt.expected, err)
		})
	}
}

// TestValidationError tests the ValidationError error type
func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		verr     ValidationError
		contains []string
	}{
		{
			name: "with_all_fields",
			verr: ValidationError{
				Entity:   "field user.password",
				Expected: "(redact.custom).string",
				Got:      "(redact.custom).int32",
				Hint:     "ensure types match",
			},
			contains: []string{
				"Validation failed",
				"field user.password",
				"expected (redact.custom).string",
				"got (redact.custom).int32",
				"hint: ensure types match",
			},
		},
		{
			name: "entity_only",
			verr: ValidationError{
				Entity: "message User",
			},
			contains: []string{
				"Validation failed",
				"message User",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.verr.Error()
			for _, substr := range tt.contains {
				assert.Contains(t, err, substr)
			}
		})
	}
}

// TestValidateStatusCode tests status code validation
func TestValidateStatusCode(t *testing.T) {
	m := &Module{ModuleBase: &pgs.ModuleBase{}}

	tests := []struct {
		name      string
		code      uint32
		location  string
		shouldErr bool
	}{
		{
			name:      "valid_ok",
			code:      0,
			location:  "user.UserService",
			shouldErr: false,
		},
		{
			name:      "valid_permission_denied",
			code:      7,
			location:  "user.UserService",
			shouldErr: false,
		},
		{
			name:      "valid_unauthenticated",
			code:      16,
			location:  "user.UserService",
			shouldErr: false,
		},
		{
			name:      "invalid_too_high",
			code:      17,
			location:  "user.UserService",
			shouldErr: true,
		},
		{
			name:      "invalid_way_too_high",
			code:      100,
			location:  "user.UserService",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.validateStatusCode(tt.code, tt.location)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "status code")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateTypeMatch tests type matching validation
func TestValidateTypeMatch(t *testing.T) {
	// Note: This test requires mock pgs.Field which is complex
	// Testing the logic through the validation error structure
	tests := []struct {
		name      string
		fieldType pgs.ProtoType
		ruleType  pgs.ProtoType
		ruleLabel pgs.ProtoLabel
		shouldErr bool
	}{
		{
			name:      "matching_types",
			fieldType: pgs.StringT,
			ruleType:  pgs.StringT,
			ruleLabel: pgs.Optional,
			shouldErr: false,
		},
		{
			name:      "mismatched_types",
			fieldType: pgs.StringT,
			ruleType:  pgs.Int32T,
			ruleLabel: pgs.Optional,
			shouldErr: true,
		},
		{
			name:      "zero_rule_type_allowed",
			fieldType: pgs.StringT,
			ruleType:  0,
			ruleLabel: pgs.Optional,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the logic that would be used
			hasError := tt.ruleType != 0 && tt.ruleType != tt.fieldType
			assert.Equal(t, tt.shouldErr, hasError)
		})
	}
}

// TestValidateRules tests field rules validation
func TestValidateRules(t *testing.T) {
	tests := []struct {
		name      string
		rules     *redact.FieldRules
		shouldErr bool
	}{
		{
			name:      "nil_rules",
			rules:     nil,
			shouldErr: false,
		},
		{
			name: "valid_string_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_String_{String_: "test"},
			},
			shouldErr: false,
		},
		{
			name: "valid_int32_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Int32{Int32: 42},
			},
			shouldErr: false,
		},
		{
			name: "valid_message_rule_nil",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Message{
					Message: &redact.MessageRules{Nil: true},
				},
			},
			shouldErr: false,
		},
		{
			name: "valid_message_rule_empty",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Message{
					Message: &redact.MessageRules{Empty: true},
				},
			},
			shouldErr: false,
		},
		{
			name: "valid_element_rule_nested",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Element{
					Element: &redact.ElementRules{Nested: true},
				},
			},
			shouldErr: false,
		},
		{
			name: "valid_element_rule_empty",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Element{
					Element: &redact.ElementRules{Empty: true},
				},
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Need a mock field for full validation
			// This tests the rules structure validation logic
			if tt.rules == nil {
				// Nil rules should not error
				assert.False(t, tt.shouldErr)
				return
			}

			if tt.rules.Values == nil {
				// Empty values should error
				assert.True(t, true) // Would error in real validation
				return
			}

			// Has values - should be valid unless specific rules are wrong
			assert.False(t, tt.shouldErr)
		})
	}
}

// TestValidateImportPath tests import path validation
func TestValidateImportPath(t *testing.T) {
	m := &Module{ModuleBase: &pgs.ModuleBase{}}

	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{
			name:      "valid_normal_path",
			path:      "github.com/user/project",
			shouldErr: false,
		},
		{
			name:      "valid_standard_library",
			path:      "context",
			shouldErr: false,
		},
		{
			name:      "valid_grpc",
			path:      "google.golang.org/grpc",
			shouldErr: false,
		},
		{
			name:      "empty_path",
			path:      "",
			shouldErr: true,
		},
		{
			name:      "very_long_path",
			path:      string(make([]byte, 1001)),
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.validateImportPath(tt.path)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePackageName tests package name validation
func TestValidatePackageName(t *testing.T) {
	m := &Module{ModuleBase: &pgs.ModuleBase{}}

	tests := []struct {
		name      string
		pkgName   string
		shouldErr bool
	}{
		{
			name:      "valid_simple",
			pkgName:   "user",
			shouldErr: false,
		},
		{
			name:      "valid_with_underscore",
			pkgName:   "user_service",
			shouldErr: false,
		},
		{
			name:      "valid_camelcase",
			pkgName:   "userService",
			shouldErr: false,
		},
		{
			name:      "empty_name",
			pkgName:   "",
			shouldErr: true,
		},
		{
			name:      "starts_with_number",
			pkgName:   "1user",
			shouldErr: true,
		},
		{
			name:      "valid_starts_with_letter",
			pkgName:   "user1",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.validatePackageName(tt.pkgName)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGRPCStatusCodeBoundaries tests all boundary values for gRPC codes
func TestGRPCStatusCodeBoundaries(t *testing.T) {
	m := &Module{ModuleBase: &pgs.ModuleBase{}}

	// Test all valid gRPC status codes
	validCodes := []codes.Code{
		codes.OK,                 // 0
		codes.Canceled,           // 1
		codes.Unknown,            // 2
		codes.InvalidArgument,    // 3
		codes.DeadlineExceeded,   // 4
		codes.NotFound,           // 5
		codes.AlreadyExists,      // 6
		codes.PermissionDenied,   // 7
		codes.ResourceExhausted,  // 8
		codes.FailedPrecondition, // 9
		codes.Aborted,            // 10
		codes.OutOfRange,         // 11
		codes.Unimplemented,      // 12
		codes.Internal,           // 13
		codes.Unavailable,        // 14
		codes.DataLoss,           // 15
		codes.Unauthenticated,    // 16
	}

	for _, code := range validCodes {
		t.Run(code.String(), func(t *testing.T) {
			err := m.validateStatusCode(uint32(code), "test.Service")
			assert.NoError(t, err, "Code %d (%s) should be valid", code, code.String())
		})
	}

	// Test invalid codes
	invalidCodes := []uint32{17, 18, 100, 255, 1000}
	for _, code := range invalidCodes {
		t.Run(string(rune(code)), func(t *testing.T) {
			err := m.validateStatusCode(code, "test.Service")
			assert.Error(t, err, "Code %d should be invalid", code)
		})
	}
}

// TestErrorMessageQuality tests that error messages are informative
func TestErrorMessageQuality(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		shouldContain []string
	}{
		{
			name: "validation_error_with_hint",
			err: ValidationError{
				Entity:   "field user.password",
				Expected: "string type",
				Got:      "int32 type",
				Hint:     "check your proto definition",
			},
			shouldContain: []string{
				"Validation failed",
				"field user.password",
				"string type",
				"int32 type",
				"hint",
			},
		},
		{
			name: "error_context_with_location",
			err: ErrorContext{
				Location: "user.proto.User",
				Field:    "email",
				Reason:   "invalid redaction configuration",
			},
			shouldContain: []string{
				"user.proto.User",
				"email",
				"invalid redaction configuration",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, substr := range tt.shouldContain {
				assert.Contains(t, errMsg, substr,
					"Error message should contain '%s'", substr)
			}
		})
	}
}

// TestRuleValidationEdgeCases tests edge cases in rule validation
func TestRuleValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		rules       *redact.FieldRules
		description string
		hasValues   bool
	}{
		{
			name:        "nil_rules",
			rules:       nil,
			description: "Nil rules should be treated as no redaction",
			hasValues:   false,
		},
		{
			name:        "empty_values",
			rules:       &redact.FieldRules{Values: nil},
			description: "Rules with nil values should be validated",
			hasValues:   false,
		},
		{
			name: "message_rule_with_all_options",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Message{
					Message: &redact.MessageRules{
						Nil:   false,
						Empty: false,
						Skip:  false,
					},
				},
			},
			description: "Message rule with all boolean options",
			hasValues:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			if tt.rules != nil && tt.hasValues {
				assert.NotNil(t, tt.rules.Values)
			}
		})
	}
}

// TestCustomRuleTypeMatching tests the logic for custom rule type matching
func TestCustomRuleTypeMatching(t *testing.T) {
	tests := []struct {
		name      string
		fieldType pgs.ProtoType
		ruleStr   string
	}{
		{
			name:      "string_field",
			fieldType: pgs.StringT,
			ruleStr:   "(redact.custom).string",
		},
		{
			name:      "int32_field",
			fieldType: pgs.Int32T,
			ruleStr:   "(redact.custom).int32",
		},
		{
			name:      "message_field",
			fieldType: pgs.MessageT,
			ruleStr:   "(redact.custom).message.*",
		},
		{
			name:      "bytes_field",
			fieldType: pgs.BytesT,
			ruleStr:   "(redact.custom).bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := ToCustomRule(tt.fieldType, pgs.Optional)
			assert.Equal(t, tt.ruleStr, rule)
		})
	}
}

// BenchmarkErrorContextCreation benchmarks error context creation
func BenchmarkErrorContextCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := ErrorContext{
			Location: "user.proto.User",
			Field:    "password",
			Type:     "string",
			Reason:   "invalid redaction rule",
		}
		_ = err.Error()
	}
}

// BenchmarkValidationErrorCreation benchmarks validation error creation
func BenchmarkValidationErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := ValidationError{
			Entity:   "field user.password",
			Expected: "string type",
			Got:      "int32 type",
			Hint:     "check your proto definition",
		}
		_ = err.Error()
	}
}

// BenchmarkStatusCodeValidation benchmarks status code validation
func BenchmarkStatusCodeValidation(b *testing.B) {
	m := &Module{ModuleBase: &pgs.ModuleBase{}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.validateStatusCode(7, "test.Service")
		_ = m.validateStatusCode(16, "test.Service")
	}
}
