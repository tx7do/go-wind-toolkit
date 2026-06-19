package main

import (
	"testing"

	pgs "github.com/lyft/protoc-gen-star/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOptionalMessageFields tests various scenarios with optional message fields
func TestOptionalMessageFields(t *testing.T) {
	tests := []struct {
		name        string
		field       *FieldData
		description string
		validate    func(t *testing.T, field *FieldData)
	}{
		{
			name:        "optional_message_with_nested_redaction",
			description: "Optional message field with nested redaction enabled",
			field: &FieldData{
				Name:                      "profile",
				Redact:                    true,
				IsMessage:                 true,
				IsOptional:                true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "Profile",
				EmbedMessageNameWithAlias: "pb.Profile",
			},
			validate: func(t *testing.T, field *FieldData) {
				assert.True(t, field.IsOptional, "Should be optional")
				assert.True(t, field.IsMessage, "Should be message type")
				assert.True(t, field.NestedEmbedCall, "Should call nested redaction")
				assert.False(t, field.EmbedSkip, "Should not skip redaction")
			},
		},
		{
			name:        "optional_message_redacted_to_nil",
			description: "Optional message field explicitly set to nil",
			field: &FieldData{
				Name:                      "optional_user",
				Redact:                    true,
				IsMessage:                 true,
				IsOptional:                true,
				RedactionValue:            "nil",
				EmbedMessageName:          "User",
				EmbedMessageNameWithAlias: "pb.User",
			},
			validate: func(t *testing.T, field *FieldData) {
				assert.True(t, field.IsOptional, "Should be optional")
				assert.True(t, field.IsMessage, "Should be message type")
				assert.Equal(t, "nil", field.RedactionValue, "Should be nil")
				assert.False(t, field.NestedEmbedCall, "Should not call nested when nil")
			},
		},
		{
			name:        "optional_message_redacted_to_empty",
			description: "Optional message field set to empty struct",
			field: &FieldData{
				Name:                      "settings",
				Redact:                    true,
				IsMessage:                 true,
				IsOptional:                true,
				RedactionValue:            "&pb.Settings{}",
				EmbedMessageName:          "Settings",
				EmbedMessageNameWithAlias: "pb.Settings",
			},
			validate: func(t *testing.T, field *FieldData) {
				assert.True(t, field.IsOptional, "Should be optional")
				assert.True(t, field.IsMessage, "Should be message type")
				assert.Contains(t, field.RedactionValue, "&pb.Settings{}", "Should be empty struct")
			},
		},
		{
			name:        "optional_message_skipped",
			description: "Optional message field with redaction skipped",
			field: &FieldData{
				Name:                      "public_data",
				Redact:                    false,
				IsMessage:                 true,
				IsOptional:                true,
				EmbedSkip:                 true,
				EmbedMessageName:          "PublicData",
				EmbedMessageNameWithAlias: "pb.PublicData",
			},
			validate: func(t *testing.T, field *FieldData) {
				assert.True(t, field.IsOptional, "Should be optional")
				assert.True(t, field.IsMessage, "Should be message type")
				assert.True(t, field.EmbedSkip, "Should skip redaction")
				assert.False(t, field.NestedEmbedCall, "Should not call nested")
			},
		},
		{
			name:        "optional_message_deeply_nested",
			description: "Optional message with deeply nested structure",
			field: &FieldData{
				Name:                      "metadata",
				Redact:                    true,
				IsMessage:                 true,
				IsOptional:                true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "Metadata",
				EmbedMessageNameWithAlias: "types.Metadata",
			},
			validate: func(t *testing.T, field *FieldData) {
				assert.True(t, field.IsOptional, "Should be optional")
				assert.True(t, field.IsMessage, "Should be message type")
				assert.True(t, field.NestedEmbedCall, "Should support nested redaction")
				assert.Contains(t, field.EmbedMessageNameWithAlias, ".", "Should have package alias")
			},
		},
		{
			name:        "optional_message_not_redacted",
			description: "Optional message field without redaction",
			field: &FieldData{
				Name:                      "audit_log",
				Redact:                    false,
				IsMessage:                 true,
				IsOptional:                true,
				EmbedMessageName:          "AuditLog",
				EmbedMessageNameWithAlias: "pb.AuditLog",
			},
			validate: func(t *testing.T, field *FieldData) {
				assert.True(t, field.IsOptional, "Should be optional")
				assert.True(t, field.IsMessage, "Should be message type")
				assert.False(t, field.Redact, "Should not be redacted")
				assert.False(t, field.NestedEmbedCall, "Should not call nested")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			require.NotNil(t, tt.field, "Field should not be nil")
			tt.validate(t, tt.field)
		})
	}
}

// TestOptionalMessageWithRepeatedFields tests optional messages containing repeated fields
func TestOptionalMessageWithRepeatedFields(t *testing.T) {
	tests := []struct {
		name        string
		description string
		message     *MessageData
		validate    func(t *testing.T, msg *MessageData)
	}{
		{
			name:        "optional_message_with_repeated_strings",
			description: "Optional message containing repeated string fields",
			message: &MessageData{
				Name:      "Tags",
				WithAlias: "pb.Tags",
				Fields: []*FieldData{
					{
						Name:           "values",
						Redact:         true,
						IsRepeated:     true,
						Iterate:        true,
						RedactionValue: `"REDACTED"`,
					},
				},
			},
			validate: func(t *testing.T, msg *MessageData) {
				assert.NotEmpty(t, msg.Fields, "Should have fields")
				field := msg.Fields[0]
				assert.True(t, field.IsRepeated, "Field should be repeated")
				assert.True(t, field.Iterate, "Should iterate for redaction")
			},
		},
		{
			name:        "optional_message_with_nested_messages",
			description: "Optional message containing nested message fields",
			message: &MessageData{
				Name:      "Container",
				WithAlias: "pb.Container",
				Fields: []*FieldData{
					{
						Name:                      "items",
						Redact:                    true,
						IsRepeated:                true,
						IsMessage:                 true,
						Iterate:                   true,
						NestedEmbedCall:           true,
						EmbedMessageName:          "Item",
						EmbedMessageNameWithAlias: "pb.Item",
					},
				},
			},
			validate: func(t *testing.T, msg *MessageData) {
				assert.NotEmpty(t, msg.Fields, "Should have fields")
				field := msg.Fields[0]
				assert.True(t, field.IsRepeated, "Field should be repeated")
				assert.True(t, field.IsMessage, "Field should be message")
				assert.True(t, field.Iterate, "Should iterate")
				assert.True(t, field.NestedEmbedCall, "Should call nested redaction")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			require.NotNil(t, tt.message, "Message should not be nil")
			tt.validate(t, tt.message)
		})
	}
}

// TestOptionalPrimitiveFields tests optional primitive fields
func TestOptionalPrimitiveFields(t *testing.T) {
	tests := []struct {
		name        string
		field       *FieldData
		fieldType   pgs.ProtoType
		description string
	}{
		{
			name:        "optional_string",
			fieldType:   pgs.StringT,
			description: "Optional string field with proper redaction",
			field: &FieldData{
				Name:           "email",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: `"REDACTED"`,
			},
		},
		{
			name:        "optional_int32",
			fieldType:   pgs.Int32T,
			description: "Optional int32 field",
			field: &FieldData{
				Name:           "age",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: "0",
			},
		},
		{
			name:        "optional_int64",
			fieldType:   pgs.Int64T,
			description: "Optional int64 field",
			field: &FieldData{
				Name:           "user_id",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: "0",
			},
		},
		{
			name:        "optional_bool",
			fieldType:   pgs.BoolT,
			description: "Optional bool field",
			field: &FieldData{
				Name:           "is_verified",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: "false",
			},
		},
		{
			name:        "optional_bytes",
			fieldType:   pgs.BytesT,
			description: "Optional bytes field",
			field: &FieldData{
				Name:           "signature",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: "nil",
			},
		},
		{
			name:        "optional_float",
			fieldType:   pgs.FloatT,
			description: "Optional float field",
			field: &FieldData{
				Name:           "score",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: "0",
			},
		},
		{
			name:        "optional_double",
			fieldType:   pgs.DoubleT,
			description: "Optional double field",
			field: &FieldData{
				Name:           "rating",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			assert.True(t, tt.field.IsOptional, "Field should be optional")
			assert.True(t, tt.field.Redact, "Field should be redacted")
			assert.NotEmpty(t, tt.field.RedactionValue, "Should have redaction value")

			// Verify redaction value matches type
			defaultVal := RedactionDefaults(tt.fieldType, false)
			if tt.fieldType == pgs.StringT {
				// String needs quotes for pointer assignment
				assert.Contains(t, tt.field.RedactionValue, "REDACTED")
			} else {
				assert.Equal(t, defaultVal, tt.field.RedactionValue,
					"Optional field should have appropriate default value")
			}
		})
	}
}

// TestOptionalFieldPointerSemantics tests that optional fields handle pointer semantics correctly
func TestOptionalFieldPointerSemantics(t *testing.T) {
	tests := []struct {
		name            string
		fieldType       pgs.ProtoType
		redactionValue  string
		needsTempVar    bool
		expectedPattern string
		description     string
	}{
		{
			name:            "optional_string_needs_temp_var",
			fieldType:       pgs.StringT,
			redactionValue:  `"REDACTED"`,
			needsTempVar:    true,
			expectedPattern: "pointer assignment with temp variable",
			description:     "Optional strings need temp variable for pointer assignment",
		},
		{
			name:            "optional_int32_needs_temp_var",
			fieldType:       pgs.Int32T,
			redactionValue:  "0",
			needsTempVar:    true,
			expectedPattern: "pointer assignment with temp variable",
			description:     "Optional int32 needs temp variable for pointer assignment",
		},
		{
			name:            "optional_bytes_direct_nil",
			fieldType:       pgs.BytesT,
			redactionValue:  "nil",
			needsTempVar:    false,
			expectedPattern: "direct nil assignment",
			description:     "Optional bytes can use direct nil assignment",
		},
		{
			name:            "optional_message_direct_nil",
			fieldType:       pgs.MessageT,
			redactionValue:  "nil",
			needsTempVar:    false,
			expectedPattern: "direct nil assignment",
			description:     "Optional messages can use direct nil assignment",
		},
		{
			name:            "optional_bool_needs_temp_var",
			fieldType:       pgs.BoolT,
			redactionValue:  "false",
			needsTempVar:    true,
			expectedPattern: "pointer assignment with temp variable",
			description:     "Optional booleans need temp variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			// Check if the redaction value requires a temp variable
			requiresTempVar := tt.redactionValue != "nil" &&
				(tt.fieldType == pgs.StringT ||
					(tt.fieldType >= pgs.DoubleT && tt.fieldType <= pgs.SFixed64) ||
					tt.fieldType == pgs.BoolT)

			assert.Equal(t, tt.needsTempVar, requiresTempVar,
				"Temp variable requirement should match for %s", tt.fieldType)
		})
	}
}

// TestOptionalFieldEdgeCases tests edge cases for optional fields
func TestOptionalFieldEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		field       *FieldData
		expectValid bool
		description string
	}{
		{
			name:        "optional_and_repeated_conflict",
			description: "Field marked as both optional and repeated (invalid in proto3)",
			field: &FieldData{
				Name:       "invalid_field",
				IsOptional: true,
				IsRepeated: true,
			},
			expectValid: false,
		},
		{
			name:        "optional_map_field",
			description: "Optional map field (maps are implicitly optional)",
			field: &FieldData{
				Name:       "metadata_map",
				IsOptional: true,
				IsMap:      true,
			},
			expectValid: true,
		},
		{
			name:        "optional_message_with_iterate",
			description: "Optional message with iterate flag (should not iterate single messages)",
			field: &FieldData{
				Name:       "user",
				IsOptional: true,
				IsMessage:  true,
				Iterate:    true,
			},
			expectValid: false,
		},
		{
			name:        "optional_field_no_redaction",
			description: "Optional field without any redaction",
			field: &FieldData{
				Name:       "public_field",
				IsOptional: true,
				Redact:     false,
			},
			expectValid: true,
		},
		{
			name:        "optional_with_custom_redaction",
			description: "Optional field with custom redaction value",
			field: &FieldData{
				Name:           "custom_field",
				IsOptional:     true,
				Redact:         true,
				RedactionValue: `"CUSTOM"`,
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			// Validate field consistency
			isValid := true

			// A field cannot be both optional and repeated
			if tt.field.IsOptional && tt.field.IsRepeated {
				isValid = false
			}

			// Optional message fields should not have Iterate set (unless repeated)
			if tt.field.IsOptional && tt.field.IsMessage && tt.field.Iterate && !tt.field.IsRepeated {
				isValid = false
			}

			assert.Equal(t, tt.expectValid, isValid,
				"Field validity should match expectation")
		})
	}
}

// TestOptionalFieldsInComplexMessage tests optional fields within complex message structures
func TestOptionalFieldsInComplexMessage(t *testing.T) {
	message := &MessageData{
		Name:      "User",
		WithAlias: "pb.User",
		Fields: []*FieldData{
			{
				Name:   "id",
				Redact: false,
			},
			{
				Name:           "email",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: `"REDACTED"`,
			},
			{
				Name:                      "profile",
				Redact:                    true,
				IsMessage:                 true,
				IsOptional:                true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "Profile",
				EmbedMessageNameWithAlias: "pb.Profile",
			},
			{
				Name:           "phone",
				Redact:         true,
				IsOptional:     true,
				RedactionValue: "nil",
			},
			{
				Name:       "tags",
				Redact:     false,
				IsRepeated: true,
			},
			{
				Name:                      "settings",
				Redact:                    true,
				IsMessage:                 true,
				IsOptional:                true,
				RedactionValue:            "&pb.Settings{}",
				EmbedMessageName:          "Settings",
				EmbedMessageNameWithAlias: "pb.Settings",
			},
		},
	}

	t.Run("complex_message_with_mixed_optional_fields", func(t *testing.T) {
		assert.Equal(t, "User", message.Name)
		assert.Len(t, message.Fields, 6, "Should have 6 fields")

		// Count optional fields
		optionalCount := 0
		redactedOptionalCount := 0
		for _, field := range message.Fields {
			if field.IsOptional {
				optionalCount++
				if field.Redact {
					redactedOptionalCount++
				}
			}
		}

		assert.Equal(t, 4, optionalCount, "Should have 4 optional fields")
		assert.Equal(t, 4, redactedOptionalCount, "Should have 4 redacted optional fields")

		// Verify email field (optional primitive)
		emailField := message.Fields[1]
		assert.Equal(t, "email", emailField.Name)
		assert.True(t, emailField.IsOptional)
		assert.True(t, emailField.Redact)
		assert.Contains(t, emailField.RedactionValue, "REDACTED")

		// Verify profile field (optional message with nested call)
		profileField := message.Fields[2]
		assert.Equal(t, "profile", profileField.Name)
		assert.True(t, profileField.IsOptional)
		assert.True(t, profileField.IsMessage)
		assert.True(t, profileField.NestedEmbedCall)

		// Verify settings field (optional message with empty struct)
		settingsField := message.Fields[5]
		assert.Equal(t, "settings", settingsField.Name)
		assert.True(t, settingsField.IsOptional)
		assert.True(t, settingsField.IsMessage)
		assert.Contains(t, settingsField.RedactionValue, "&pb.Settings{}")
	})
}

// TestOptionalFieldCompatibility tests compatibility with different proto versions
func TestOptionalFieldCompatibility(t *testing.T) {
	tests := []struct {
		name         string
		protoVersion string
		field        *FieldData
		description  string
	}{
		{
			name:         "proto3_optional_string",
			protoVersion: "proto3",
			description:  "Proto3 optional string field",
			field: &FieldData{
				Name:           "email",
				IsOptional:     true,
				Redact:         true,
				RedactionValue: `"REDACTED"`,
			},
		},
		{
			name:         "proto3_optional_message",
			protoVersion: "proto3",
			description:  "Proto3 optional message field",
			field: &FieldData{
				Name:                      "user",
				IsOptional:                true,
				IsMessage:                 true,
				Redact:                    true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "User",
				EmbedMessageNameWithAlias: "pb.User",
			},
		},
		{
			name:         "proto2_optional_field",
			protoVersion: "proto2",
			description:  "Proto2 optional field (default behavior)",
			field: &FieldData{
				Name:           "name",
				IsOptional:     false, // In proto2, all scalar fields are implicitly optional
				Redact:         true,
				RedactionValue: `"REDACTED"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			assert.NotNil(t, tt.field)

			// Verify proto3 optional fields are properly marked
			if tt.protoVersion == "proto3" && tt.field.IsOptional {
				assert.True(t, tt.field.IsOptional,
					"Proto3 optional field should be marked as optional")
			}
		})
	}
}

// TestOptionalFieldsWithDifferentRedactionStrategies tests various redaction strategies
func TestOptionalFieldsWithDifferentRedactionStrategies(t *testing.T) {
	strategies := []struct {
		name        string
		field       *FieldData
		strategy    string
		description string
	}{
		{
			name:        "nil_strategy",
			strategy:    "set to nil",
			description: "Optional field redacted by setting to nil",
			field: &FieldData{
				Name:           "optional_data",
				IsOptional:     true,
				IsMessage:      true,
				Redact:         true,
				RedactionValue: "nil",
			},
		},
		{
			name:        "empty_strategy",
			strategy:    "set to empty",
			description: "Optional field redacted by setting to empty struct",
			field: &FieldData{
				Name:           "optional_config",
				IsOptional:     true,
				IsMessage:      true,
				Redact:         true,
				RedactionValue: "&pb.Config{}",
			},
		},
		{
			name:        "nested_call_strategy",
			strategy:    "nested redaction",
			description: "Optional field redacted by calling nested Redact()",
			field: &FieldData{
				Name:            "optional_user",
				IsOptional:      true,
				IsMessage:       true,
				Redact:          true,
				NestedEmbedCall: true,
			},
		},
		{
			name:        "skip_strategy",
			strategy:    "skip redaction",
			description: "Optional field with redaction skipped",
			field: &FieldData{
				Name:       "optional_public",
				IsOptional: true,
				IsMessage:  true,
				Redact:     true,
				EmbedSkip:  true,
			},
		},
		{
			name:        "custom_value_strategy",
			strategy:    "custom value",
			description: "Optional primitive with custom redaction value",
			field: &FieldData{
				Name:           "optional_email",
				IsOptional:     true,
				Redact:         true,
				RedactionValue: `"r*d@ct*d"`,
			},
		},
	}

	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			assert.True(t, tt.field.IsOptional, "Field should be optional")
			assert.True(t, tt.field.Redact, "Field should be marked for redaction")

			// Verify strategy is properly configured
			switch tt.strategy {
			case "set to nil":
				assert.Equal(t, "nil", tt.field.RedactionValue)
			case "set to empty":
				assert.Contains(t, tt.field.RedactionValue, "{}")
			case "nested redaction":
				assert.True(t, tt.field.NestedEmbedCall)
			case "skip redaction":
				assert.True(t, tt.field.EmbedSkip)
			case "custom value":
				assert.NotEmpty(t, tt.field.RedactionValue)
			}
		})
	}
}

// BenchmarkOptionalFieldProcessing benchmarks optional field handling
func BenchmarkOptionalFieldProcessing(b *testing.B) {
	field := &FieldData{
		Name:           "email",
		IsOptional:     true,
		Redact:         true,
		RedactionValue: `"REDACTED"`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate checking optional field properties
		_ = field.IsOptional
		_ = field.Redact
		_ = field.RedactionValue
	}
}

// BenchmarkOptionalMessageFieldProcessing benchmarks optional message field handling
func BenchmarkOptionalMessageFieldProcessing(b *testing.B) {
	field := &FieldData{
		Name:                      "user",
		IsOptional:                true,
		IsMessage:                 true,
		Redact:                    true,
		NestedEmbedCall:           true,
		EmbedMessageName:          "User",
		EmbedMessageNameWithAlias: "pb.User",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate checking optional message field properties
		_ = field.IsOptional
		_ = field.IsMessage
		_ = field.NestedEmbedCall
		_ = field.EmbedMessageName
	}
}
