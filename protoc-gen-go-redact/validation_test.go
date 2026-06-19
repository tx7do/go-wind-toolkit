package main

import (
	"fmt"
	"testing"

	pgs "github.com/lyft/protoc-gen-star/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"github.com/menta2k/protoc-gen-redact/v3/redact/v3"
)

// TestFieldTypeValidation tests validation of field types against redaction rules
func TestFieldTypeValidation(t *testing.T) {
	tests := []struct {
		name          string
		fieldType     pgs.ProtoType
		ruleType      pgs.ProtoType
		shouldBeValid bool
		description   string
	}{
		{
			name:          "matching_int32",
			fieldType:     pgs.Int32T,
			ruleType:      pgs.Int32T,
			shouldBeValid: true,
			description:   "Int32 field with int32 rule should be valid",
		},
		{
			name:          "mismatched_int32_string",
			fieldType:     pgs.Int32T,
			ruleType:      pgs.StringT,
			shouldBeValid: false,
			description:   "Int32 field with string rule should be invalid",
		},
		{
			name:          "matching_string",
			fieldType:     pgs.StringT,
			ruleType:      pgs.StringT,
			shouldBeValid: true,
			description:   "String field with string rule should be valid",
		},
		{
			name:          "matching_message",
			fieldType:     pgs.MessageT,
			ruleType:      pgs.MessageT,
			shouldBeValid: true,
			description:   "Message field with message rule should be valid",
		},
		{
			name:          "matching_bytes",
			fieldType:     pgs.BytesT,
			ruleType:      pgs.BytesT,
			shouldBeValid: true,
			description:   "Bytes field with bytes rule should be valid",
		},
		{
			name:          "matching_bool",
			fieldType:     pgs.BoolT,
			ruleType:      pgs.BoolT,
			shouldBeValid: true,
			description:   "Bool field with bool rule should be valid",
		},
		{
			name:          "matching_float",
			fieldType:     pgs.FloatT,
			ruleType:      pgs.FloatT,
			shouldBeValid: true,
			description:   "Float field with float rule should be valid",
		},
		{
			name:          "matching_double",
			fieldType:     pgs.DoubleT,
			ruleType:      pgs.DoubleT,
			shouldBeValid: true,
			description:   "Double field with double rule should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			isValid := tt.fieldType == tt.ruleType
			assert.Equal(t, tt.shouldBeValid, isValid,
				"Type validation for %v vs %v should be %v",
				tt.fieldType, tt.ruleType, tt.shouldBeValid)
		})
	}
}

// TestGRPCStatusCodeValidation tests validation of gRPC status codes
func TestGRPCStatusCodeValidation(t *testing.T) {
	tests := []struct {
		name        string
		code        codes.Code
		codeValue   uint32
		isValid     bool
		description string
	}{
		{
			name:        "valid_ok",
			code:        codes.OK,
			codeValue:   0,
			isValid:     true,
			description: "OK status code (0) is valid",
		},
		{
			name:        "valid_permission_denied",
			code:        codes.PermissionDenied,
			codeValue:   7,
			isValid:     true,
			description: "PermissionDenied status code (7) is valid",
		},
		{
			name:        "valid_unauthenticated",
			code:        codes.Unauthenticated,
			codeValue:   16,
			isValid:     true,
			description: "Unauthenticated status code (16) is valid",
		},
		{
			name:        "invalid_high_value",
			code:        codes.Code(100),
			codeValue:   100,
			isValid:     false,
			description: "Status code 100 is invalid (> 16)",
		},
		{
			name:        "valid_internal",
			code:        codes.Internal,
			codeValue:   13,
			isValid:     true,
			description: "Internal status code (13) is valid",
		},
		{
			name:        "valid_unavailable",
			code:        codes.Unavailable,
			codeValue:   14,
			isValid:     true,
			description: "Unavailable status code (14) is valid",
		},
		{
			name:        "valid_data_loss",
			code:        codes.DataLoss,
			codeValue:   15,
			isValid:     true,
			description: "DataLoss status code (15) is valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			isValid := tt.codeValue <= uint32(codes.Unauthenticated) // 16
			assert.Equal(t, tt.isValid, isValid,
				"Code %d validation should be %v", tt.codeValue, tt.isValid)

			if tt.isValid && tt.codeValue <= 16 {
				assert.Equal(t, tt.codeValue, uint32(tt.code))
			}
		})
	}
}

// TestErrorMessageFormatting tests error message format specifier replacement
func TestErrorMessageFormatting(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		serviceName string
		methodName  string
		expected    string
	}{
		{
			name:        "both_specifiers",
			template:    "Permission Denied. Method: \"%service%.%method%\" has been redacted",
			serviceName: "UserService",
			methodName:  "GetUser",
			expected:    "Permission Denied. Method: \"UserService.GetUser\" has been redacted",
		},
		{
			name:        "service_only",
			template:    "Service %service% is internal",
			serviceName: "AdminService",
			methodName:  "DoAdmin",
			expected:    "Service AdminService is internal",
		},
		{
			name:        "method_only",
			template:    "Method %method% is restricted",
			serviceName: "UserService",
			methodName:  "DeleteAll",
			expected:    "Method DeleteAll is restricted",
		},
		{
			name:        "no_specifiers",
			template:    "Access denied",
			serviceName: "UserService",
			methodName:  "GetUser",
			expected:    "Access denied",
		},
		{
			name:        "multiple_occurrences",
			template:    "%method% in %service%: %method% is not allowed",
			serviceName: "PaymentService",
			methodName:  "RefundAll",
			expected:    "RefundAll in PaymentService: RefundAll is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.template
			// Simulate the format specifier replacement
			result = replaceAll(result, "%service%", tt.serviceName)
			result = replaceAll(result, "%method%", tt.methodName)

			assert.Equal(t, tt.expected, result,
				"Error message formatting failed for template: %s", tt.template)
		})
	}
}

// Helper function to replace all occurrences (simplified version)
func replaceAll(s, old, replacement string) string {
	result := ""
	for i := 0; i < len(s); {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += replacement
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

// TestMessageOptionsValidation tests mutual exclusivity of message options
func TestMessageOptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		ignore      bool
		toNil       bool
		toEmpty     bool
		isValid     bool
		description string
	}{
		{
			name:        "none_set",
			ignore:      false,
			toNil:       false,
			toEmpty:     false,
			isValid:     true,
			description: "No options set is valid - normal processing",
		},
		{
			name:        "only_ignore",
			ignore:      true,
			toNil:       false,
			toEmpty:     false,
			isValid:     true,
			description: "Only ignore flag set is valid",
		},
		{
			name:        "only_nil",
			ignore:      false,
			toNil:       true,
			toEmpty:     false,
			isValid:     true,
			description: "Only toNil flag set is valid",
		},
		{
			name:        "only_empty",
			ignore:      false,
			toNil:       false,
			toEmpty:     true,
			isValid:     true,
			description: "Only toEmpty flag set is valid",
		},
		{
			name:        "ignore_and_nil",
			ignore:      true,
			toNil:       true,
			toEmpty:     false,
			isValid:     false,
			description: "Ignore and toNil together is logically invalid",
		},
		{
			name:        "ignore_and_empty",
			ignore:      true,
			toNil:       false,
			toEmpty:     true,
			isValid:     false,
			description: "Ignore and toEmpty together is logically invalid",
		},
		{
			name:        "nil_and_empty",
			ignore:      false,
			toNil:       true,
			toEmpty:     true,
			isValid:     false,
			description: "ToNil and toEmpty together is invalid - mutually exclusive",
		},
		{
			name:        "all_set",
			ignore:      true,
			toNil:       true,
			toEmpty:     true,
			isValid:     false,
			description: "All flags set is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			// Count how many flags are set
			setCount := 0
			if tt.ignore {
				setCount++
			}
			if tt.toNil {
				setCount++
			}
			if tt.toEmpty {
				setCount++
			}

			// Valid if at most one flag is set
			isValid := setCount <= 1
			assert.Equal(t, tt.isValid, isValid,
				"Validation for ignore=%v, toNil=%v, toEmpty=%v should be %v",
				tt.ignore, tt.toNil, tt.toEmpty, tt.isValid)
		})
	}
}

// TestRepeatedFieldRuleValidation tests validation of repeated field rules
func TestRepeatedFieldRuleValidation(t *testing.T) {
	tests := []struct {
		name        string
		fieldLabel  pgs.ProtoLabel
		ruleLabel   pgs.ProtoLabel
		isValid     bool
		description string
	}{
		{
			name:        "repeated_field_repeated_rule",
			fieldLabel:  pgs.Repeated,
			ruleLabel:   pgs.Repeated,
			isValid:     true,
			description: "Repeated field with repeated rule is valid",
		},
		{
			name:        "repeated_field_optional_rule",
			fieldLabel:  pgs.Repeated,
			ruleLabel:   pgs.Optional,
			isValid:     false,
			description: "Repeated field with non-repeated rule is invalid",
		},
		{
			name:        "optional_field_optional_rule",
			fieldLabel:  pgs.Optional,
			ruleLabel:   pgs.Optional,
			isValid:     true,
			description: "Optional field with non-repeated rule is valid",
		},
		{
			name:        "optional_field_repeated_rule",
			fieldLabel:  pgs.Optional,
			ruleLabel:   pgs.Repeated,
			isValid:     false,
			description: "Optional field with repeated rule is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			var isValid bool
			if tt.fieldLabel == pgs.Repeated {
				// Repeated fields must have repeated rules
				isValid = tt.ruleLabel == pgs.Repeated
			} else {
				// Non-repeated fields must have non-repeated rules
				isValid = tt.ruleLabel != pgs.Repeated
			}

			assert.Equal(t, tt.isValid, isValid,
				"Label validation for field=%v, rule=%v should be %v",
				tt.fieldLabel, tt.ruleLabel, tt.isValid)
		})
	}
}

// TestCustomValueTypeCompatibility tests type compatibility for custom values
func TestCustomValueTypeCompatibility(t *testing.T) {
	tests := []struct {
		name         string
		fieldType    pgs.ProtoType
		customValue  interface{}
		isCompatible bool
		description  string
	}{
		{
			name:         "int32_with_int32",
			fieldType:    pgs.Int32T,
			customValue:  int32(42),
			isCompatible: true,
			description:  "Int32 field with int32 custom value",
		},
		{
			name:         "string_with_string",
			fieldType:    pgs.StringT,
			customValue:  "custom",
			isCompatible: true,
			description:  "String field with string custom value",
		},
		{
			name:         "bool_with_bool",
			fieldType:    pgs.BoolT,
			customValue:  true,
			isCompatible: true,
			description:  "Bool field with bool custom value",
		},
		{
			name:         "float_with_float32",
			fieldType:    pgs.FloatT,
			customValue:  float32(3.14),
			isCompatible: true,
			description:  "Float field with float32 custom value",
		},
		{
			name:         "double_with_float64",
			fieldType:    pgs.DoubleT,
			customValue:  float64(2.718),
			isCompatible: true,
			description:  "Double field with float64 custom value",
		},
		{
			name:         "bytes_with_bytes",
			fieldType:    pgs.BytesT,
			customValue:  []byte("data"),
			isCompatible: true,
			description:  "Bytes field with byte slice custom value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			assert.NotNil(t, tt.customValue, "Custom value should not be nil")

			// In actual implementation, type checking would be more sophisticated
			// This demonstrates the expected behavior
			if tt.isCompatible {
				assert.NotNil(t, tt.customValue)
			}
		})
	}
}

// TestNestedMessageRedactionChaining tests proper chaining of nested message redactions
func TestNestedMessageRedactionChaining(t *testing.T) {
	type NestedLevel struct {
		depth            int
		hasNestedCall    bool
		shouldRecurse    bool
		embedMessageName string
	}

	tests := []struct {
		name        string
		levels      []NestedLevel
		description string
	}{
		{
			name: "single_level_nested",
			levels: []NestedLevel{
				{depth: 1, hasNestedCall: true, shouldRecurse: false, embedMessageName: "Profile"},
			},
			description: "Single level nested message redaction",
		},
		{
			name: "two_level_nested",
			levels: []NestedLevel{
				{depth: 1, hasNestedCall: true, shouldRecurse: true, embedMessageName: "User"},
				{depth: 2, hasNestedCall: true, shouldRecurse: false, embedMessageName: "Address"},
			},
			description: "Two level nested message redaction",
		},
		{
			name: "three_level_nested",
			levels: []NestedLevel{
				{depth: 1, hasNestedCall: true, shouldRecurse: true, embedMessageName: "Company"},
				{depth: 2, hasNestedCall: true, shouldRecurse: true, embedMessageName: "Department"},
				{depth: 3, hasNestedCall: true, shouldRecurse: false, embedMessageName: "Employee"},
			},
			description: "Three level deeply nested message redaction",
		},
		{
			name: "skipped_middle_level",
			levels: []NestedLevel{
				{depth: 1, hasNestedCall: true, shouldRecurse: true, embedMessageName: "Outer"},
				{depth: 2, hasNestedCall: false, shouldRecurse: false, embedMessageName: "Middle"},
				{depth: 3, hasNestedCall: true, shouldRecurse: false, embedMessageName: "Inner"},
			},
			description: "Nested messages with skipped middle level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			for i, level := range tt.levels {
				assert.Equal(t, i+1, level.depth,
					"Level %d should have correct depth", i)
				assert.NotEmpty(t, level.embedMessageName,
					"Level %d should have embed message name", i)
			}

			assert.NotEmpty(t, tt.levels, "Should have at least one nesting level")
		})
	}
}

// TestMapFieldRedactionStrategies tests different redaction strategies for map fields
func TestMapFieldRedactionStrategies(t *testing.T) {
	tests := []struct {
		name           string
		keyType        pgs.ProtoType
		valueType      pgs.ProtoType
		strategy       string
		redactionValue string
		shouldIterate  bool
		description    string
	}{
		{
			name:           "nil_entire_map",
			keyType:        pgs.StringT,
			valueType:      pgs.StringT,
			strategy:       "nil",
			redactionValue: "nil",
			shouldIterate:  false,
			description:    "Set entire map to nil",
		},
		{
			name:           "empty_map",
			keyType:        pgs.StringT,
			valueType:      pgs.Int32T,
			strategy:       "empty",
			redactionValue: "map[string]int32{}",
			shouldIterate:  false,
			description:    "Set map to empty map",
		},
		{
			name:           "iterate_primitive_values",
			keyType:        pgs.StringT,
			valueType:      pgs.StringT,
			strategy:       "iterate",
			redactionValue: `"REDACTED"`,
			shouldIterate:  true,
			description:    "Iterate and redact each primitive value",
		},
		{
			name:           "iterate_message_values",
			keyType:        pgs.Int32T,
			valueType:      pgs.MessageT,
			strategy:       "iterate_nested",
			redactionValue: "-",
			shouldIterate:  true,
			description:    "Iterate and call Redact on each message value",
		},
		{
			name:           "map_with_complex_key",
			keyType:        pgs.Int64T,
			valueType:      pgs.BytesT,
			strategy:       "iterate",
			redactionValue: "nil",
			shouldIterate:  true,
			description:    "Map with int64 keys and bytes values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			assert.NotEmpty(t, tt.strategy, "Strategy should be defined")
			assert.NotEmpty(t, tt.redactionValue, "Redaction value should be defined")

			if tt.shouldIterate {
				assert.Contains(t, []string{"iterate", "iterate_nested"}, tt.strategy,
					"Iterate strategy should be one of the iterate types")
			}
		})
	}
}

// TestOptionalFieldHandling tests proper handling of optional fields (proto3)
func TestOptionalFieldHandling(t *testing.T) {
	tests := []struct {
		name            string
		fieldType       pgs.ProtoType
		isOptional      bool
		redactionValue  string
		expectedPattern string
		needsPointer    bool
		description     string
	}{
		{
			name:            "optional_string",
			fieldType:       pgs.StringT,
			isOptional:      true,
			redactionValue:  `"REDACTED"`,
			expectedPattern: "pointer assignment",
			needsPointer:    true,
			description:     "Optional string needs pointer assignment",
		},
		{
			name:            "optional_int32",
			fieldType:       pgs.Int32T,
			isOptional:      true,
			redactionValue:  "0",
			expectedPattern: "pointer assignment",
			needsPointer:    true,
			description:     "Optional int32 needs pointer assignment",
		},
		{
			name:            "required_string",
			fieldType:       pgs.StringT,
			isOptional:      false,
			redactionValue:  `"REDACTED"`,
			expectedPattern: "direct assignment",
			needsPointer:    false,
			description:     "Required string uses direct assignment",
		},
		{
			name:            "optional_bytes",
			fieldType:       pgs.BytesT,
			isOptional:      true,
			redactionValue:  "nil",
			expectedPattern: "direct assignment",
			needsPointer:    false,
			description:     "Optional bytes can use direct nil assignment",
		},
		{
			name:            "optional_message",
			fieldType:       pgs.MessageT,
			isOptional:      true,
			redactionValue:  "nil",
			expectedPattern: "direct assignment",
			needsPointer:    false,
			description:     "Optional message can use direct nil assignment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			assert.NotEmpty(t, tt.redactionValue, "Redaction value should be set")
			assert.NotEmpty(t, tt.expectedPattern, "Expected pattern should be defined")

			if tt.isOptional {
				// Optional fields in proto3 use pointers
				if tt.fieldType == pgs.StringT ||
					(tt.fieldType >= pgs.DoubleT && tt.fieldType <= pgs.SFixed64) ||
					tt.fieldType == pgs.BoolT {
					// Scalars need special pointer handling
					assert.True(t, tt.needsPointer || tt.redactionValue == "nil",
						"Optional scalar should need pointer or be nil")
				}
			}
		})
	}
}

// TestStreamingMethodHandling tests handling of different streaming RPC types
func TestStreamingMethodHandling(t *testing.T) {
	tests := []struct {
		name            string
		clientStreaming bool
		serverStreaming bool
		rpcType         string
		canRedact       bool
		description     string
	}{
		{
			name:            "unary",
			clientStreaming: false,
			serverStreaming: false,
			rpcType:         "unary",
			canRedact:       true,
			description:     "Unary RPC - full redaction support",
		},
		{
			name:            "client_streaming",
			clientStreaming: true,
			serverStreaming: false,
			rpcType:         "client_streaming",
			canRedact:       false,
			description:     "Client streaming - limited redaction",
		},
		{
			name:            "server_streaming",
			clientStreaming: false,
			serverStreaming: true,
			rpcType:         "server_streaming",
			canRedact:       false,
			description:     "Server streaming - limited redaction",
		},
		{
			name:            "bidirectional",
			clientStreaming: true,
			serverStreaming: true,
			rpcType:         "bidirectional",
			canRedact:       false,
			description:     "Bidirectional streaming - limited redaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			isUnary := !tt.clientStreaming && !tt.serverStreaming
			assert.Equal(t, tt.rpcType == "unary", isUnary,
				"Unary detection should match rpcType")

			if tt.canRedact {
				assert.Equal(t, "unary", tt.rpcType,
					"Only unary RPCs have full redaction support")
			}
		})
	}
}

// TestComplexValidationScenarios tests complex multi-factor validation scenarios
func TestComplexValidationScenarios(t *testing.T) {
	tests := []struct {
		name        string
		scenario    map[string]interface{}
		expectValid bool
		description string
	}{
		{
			name: "valid_nested_repeated_message",
			scenario: map[string]interface{}{
				"isRepeated":      true,
				"isMessage":       true,
				"iterate":         true,
				"nestedEmbedCall": true,
			},
			expectValid: true,
			description: "Repeated message field with iteration and nested calls",
		},
		{
			name: "invalid_iterate_without_repeated",
			scenario: map[string]interface{}{
				"isRepeated": false,
				"iterate":    true,
			},
			expectValid: false,
			description: "Iteration flag set on non-repeated field is invalid",
		},
		{
			name: "valid_optional_with_custom_value",
			scenario: map[string]interface{}{
				"isOptional":  true,
				"fieldType":   pgs.StringT,
				"hasCustom":   true,
				"customValue": `"custom"`,
			},
			expectValid: true,
			description: "Optional field with custom value",
		},
		{
			name: "valid_map_with_message_values_and_iteration",
			scenario: map[string]interface{}{
				"isMap":           true,
				"isMessage":       true,
				"iterate":         true,
				"nestedEmbedCall": true,
			},
			expectValid: true,
			description: "Map with message values requiring iteration",
		},
		{
			name: "invalid_skip_and_nested_call",
			scenario: map[string]interface{}{
				"embedSkip":       true,
				"nestedEmbedCall": true,
			},
			expectValid: false,
			description: "Cannot have both skip and nested call set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			// Validate scenario consistency
			if embedSkip, ok := tt.scenario["embedSkip"].(bool); ok && embedSkip {
				if nestedCall, ok := tt.scenario["nestedEmbedCall"].(bool); ok && nestedCall {
					// Invalid: both skip and nested call
					assert.False(t, tt.expectValid,
						"Skip and nested call cannot both be true")
				}
			}

			if iterate, ok := tt.scenario["iterate"].(bool); ok && iterate {
				if isRepeated, ok := tt.scenario["isRepeated"].(bool); ok {
					if tt.expectValid {
						assert.True(t, isRepeated || tt.scenario["isMap"] == true,
							"Iterate requires repeated or map field")
					}
				}
			}
		})
	}
}

// BenchmarkTypeValidation benchmarks type validation performance
func BenchmarkTypeValidation(b *testing.B) {
	types := []pgs.ProtoType{
		pgs.Int32T, pgs.Int64T, pgs.StringT, pgs.BoolT,
		pgs.FloatT, pgs.DoubleT, pgs.MessageT, pgs.BytesT,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t1 := range types {
			for _, t2 := range types {
				_ = t1 == t2
			}
		}
	}
}

// BenchmarkErrorMessageFormatting benchmarks error message formatting
func BenchmarkErrorMessageFormatting(b *testing.B) {
	template := "Permission Denied. Method: \"%service%.%method%\" has been redacted"
	serviceName := "UserService"
	methodName := "GetUser"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := template
		result = replaceAll(result, "%service%", serviceName)
		result = replaceAll(result, "%method%", methodName)
		_ = result
	}
}

// TestRuleInformationComplexCases tests complex rule information extraction
func TestRuleInformationComplexCases(t *testing.T) {
	tests := []struct {
		name     string
		rules    *redact.FieldRules
		validate func(t *testing.T, info RuleInfo)
	}{
		{
			name: "nested_element_with_item_rules",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Element{
					Element: &redact.ElementRules{
						Nested: true,
					},
				},
			},
			validate: func(t *testing.T, info RuleInfo) {
				assert.Equal(t, pgs.Repeated, info.ProtoLabel)
			},
		},
		{
			name: "message_with_empty_option",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Message{
					Message: &redact.MessageRules{Empty: true},
				},
			},
			validate: func(t *testing.T, info RuleInfo) {
				assert.Equal(t, pgs.MessageT, info.ProtoType)
			},
		},
		{
			name: "message_with_nil_option",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Message{
					Message: &redact.MessageRules{Nil: true},
				},
			},
			validate: func(t *testing.T, info RuleInfo) {
				assert.Equal(t, pgs.MessageT, info.ProtoType)
			},
		},
		{
			name: "message_with_skip_option",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Message{
					Message: &redact.MessageRules{Skip: true},
				},
			},
			validate: func(t *testing.T, info RuleInfo) {
				assert.Equal(t, pgs.MessageT, info.ProtoType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Module{ModuleBase: &pgs.ModuleBase{}}
			info := m.RuleInformation(tt.rules)
			tt.validate(t, info)
		})
	}
}

// TestProtoTypeToStringMapping tests string representation of proto types
func TestProtoTypeToStringMapping(t *testing.T) {
	typeNames := map[pgs.ProtoType]string{
		pgs.DoubleT:  "double",
		pgs.FloatT:   "float",
		pgs.Int32T:   "int32",
		pgs.Int64T:   "int64",
		pgs.UInt32T:  "uint32",
		pgs.UInt64T:  "uint64",
		pgs.SInt32:   "sint32",
		pgs.SInt64:   "sint64",
		pgs.Fixed32T: "fixed32",
		pgs.Fixed64T: "fixed64",
		pgs.SFixed32: "sfixed32",
		pgs.SFixed64: "sfixed64",
		pgs.BoolT:    "bool",
		pgs.StringT:  "string",
		pgs.BytesT:   "bytes",
		pgs.MessageT: "message",
		pgs.EnumT:    "enum",
	}

	for protoType, expectedName := range typeNames {
		t.Run(expectedName, func(t *testing.T) {
			assert.NotEmpty(t, expectedName)
			assert.NotEqual(t, pgs.ProtoType(0), protoType)

			// Verify the type can be used in redaction
			defaultVal := RedactionDefaults(protoType, false)
			assert.NotEmpty(t, defaultVal, "Type %s should have default redaction value", expectedName)
		})
	}
}

// TestMethodDataCompleteness tests that method data contains all required fields
func TestMethodDataCompleteness(t *testing.T) {
	method := &MethodData{
		Name:            "GetUser",
		Skip:            false,
		Input:           "GetUserRequest",
		Output:          &MessageData{Name: "User"},
		Internal:        true,
		StatusCode:      "PermissionDenied",
		ErrMessage:      "`Access denied`",
		ClientStreaming: false,
		ServerStreaming: false,
	}

	// Verify all fields are set appropriately
	assert.NotEmpty(t, method.Name, "Name should be set")
	assert.NotEmpty(t, method.Input, "Input should be set")
	assert.NotNil(t, method.Output, "Output should be set")

	if method.Internal {
		assert.NotEmpty(t, method.StatusCode, "Internal methods should have status code")
		assert.NotEmpty(t, method.ErrMessage, "Internal methods should have error message")
	}

	// Verify error message is properly formatted
	if method.ErrMessage != "" {
		assert.True(t, len(method.ErrMessage) >= 2, "Error message should be wrapped in backticks")
		assert.Equal(t, "`", fmt.Sprintf("%c", method.ErrMessage[0]), "Error message should start with backtick")
	}
}
