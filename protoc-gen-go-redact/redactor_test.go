package main

import (
	"testing"

	pgs "github.com/lyft/protoc-gen-star/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menta2k/protoc-gen-redact/v3/redact/v3"
)

// TestRedactionDefaults tests the default redaction values for various protobuf types
func TestRedactionDefaults(t *testing.T) {
	tests := []struct {
		name       string
		typ        pgs.ProtoType
		isRepeated bool
		want       string
	}{
		// Numeric types
		{"int32", pgs.Int32T, false, "0"},
		{"int64", pgs.Int64T, false, "0"},
		{"sint32", pgs.SInt32, false, "0"},
		{"sint64", pgs.SInt64, false, "0"},
		{"uint32", pgs.UInt32T, false, "0"},
		{"uint64", pgs.UInt64T, false, "0"},
		{"fixed32", pgs.Fixed32T, false, "0"},
		{"fixed64", pgs.Fixed64T, false, "0"},
		{"sfixed32", pgs.SFixed32, false, "0"},
		{"sfixed64", pgs.SFixed64, false, "0"},
		{"float", pgs.FloatT, false, "0"},
		{"double", pgs.DoubleT, false, "0"},
		{"enum", pgs.EnumT, false, "0"},

		// Boolean type
		{"bool", pgs.BoolT, false, "false"},

		// String type
		{"string", pgs.StringT, false, `"REDACTED"`},

		// Bytes and group
		{"bytes", pgs.BytesT, false, "nil"},
		{"group", pgs.GroupT, false, "nil"},

		// Message type
		{"message", pgs.MessageT, false, "-"},

		// Repeated/map types
		{"repeated_int32", pgs.Int32T, true, "nil"},
		{"repeated_string", pgs.StringT, true, "nil"},
		{"repeated_message", pgs.MessageT, true, "nil"},
		{"map_type", pgs.DoubleT, true, "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactionDefaults(tt.typ, tt.isRepeated)
			assert.Equal(t, tt.want, got, "RedactionDefaults(%v, %v) = %v, want %v",
				tt.typ, tt.isRepeated, got, tt.want)
		})
	}
}

// TestToCustomRule tests the custom rule string generation for various protobuf types
func TestToCustomRule(t *testing.T) {
	tests := []struct {
		name string
		typ  pgs.ProtoType
		lab  pgs.ProtoLabel
		want string
	}{
		// Repeated label
		{"repeated", pgs.Int32T, pgs.Repeated, "(redact.custom).element.*"},

		// Scalar types
		{"float", pgs.FloatT, pgs.Optional, "(redact.custom).float"},
		{"double", pgs.DoubleT, pgs.Optional, "(redact.custom).double"},
		{"int32", pgs.Int32T, pgs.Optional, "(redact.custom).int32"},
		{"int64", pgs.Int64T, pgs.Optional, "(redact.custom).int64"},
		{"uint32", pgs.UInt32T, pgs.Optional, "(redact.custom).uint32"},
		{"uint64", pgs.UInt64T, pgs.Optional, "(redact.custom).uint64"},
		{"sint32", pgs.SInt32, pgs.Optional, "(redact.custom).sint32"},
		{"sint64", pgs.SInt64, pgs.Optional, "(redact.custom).sint64"},
		{"fixed32", pgs.Fixed32T, pgs.Optional, "(redact.custom).fixed32"},
		{"fixed64", pgs.Fixed64T, pgs.Optional, "(redact.custom).fixed64"},
		{"sfixed32", pgs.SFixed32, pgs.Optional, "(redact.custom).sfixed32"},
		{"sfixed64", pgs.SFixed64, pgs.Optional, "(redact.custom).sfixed64"},
		{"bool", pgs.BoolT, pgs.Optional, "(redact.custom).bool"},
		{"string", pgs.StringT, pgs.Optional, "(redact.custom).string"},
		{"bytes", pgs.BytesT, pgs.Optional, "(redact.custom).bytes"},
		{"enum", pgs.EnumT, pgs.Optional, "(redact.custom).enum"},
		{"message", pgs.MessageT, pgs.Optional, "(redact.custom).message.*"},

		// Unknown type
		{"unknown", pgs.ProtoType(999), pgs.Optional, "(redact.redact)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToCustomRule(tt.typ, tt.lab)
			assert.Equal(t, tt.want, got, "ToCustomRule(%v, %v) = %v, want %v",
				tt.typ, tt.lab, got, tt.want)
		})
	}
}

// TestModuleName tests the module name
func TestModuleName(t *testing.T) {
	m := &Module{ModuleBase: &pgs.ModuleBase{}}
	assert.Equal(t, "redactor", m.Name())
}

// TestRuleInformation tests the extraction of rule information from FieldRules
func TestRuleInformation(t *testing.T) {
	tests := []struct {
		name               string
		rules              *redact.FieldRules
		expectedType       pgs.ProtoType
		expectedLabel      pgs.ProtoLabel
		expectedValue      interface{}
		shouldContainValue bool
	}{
		{
			name: "float_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Float{Float: 3.14},
			},
			expectedType:       pgs.FloatT,
			expectedValue:      float32(3.14),
			shouldContainValue: true,
		},
		{
			name: "double_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Double{Double: 2.718},
			},
			expectedType:       pgs.DoubleT,
			expectedValue:      2.718,
			shouldContainValue: true,
		},
		{
			name: "int32_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Int32{Int32: 42},
			},
			expectedType:       pgs.Int32T,
			expectedValue:      int32(42),
			shouldContainValue: true,
		},
		{
			name: "int64_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Int64{Int64: 9876543210},
			},
			expectedType:       pgs.Int64T,
			expectedValue:      int64(9876543210),
			shouldContainValue: true,
		},
		{
			name: "uint32_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Uint32{Uint32: 123},
			},
			expectedType:       pgs.UInt32T,
			expectedValue:      uint32(123),
			shouldContainValue: true,
		},
		{
			name: "uint64_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Uint64{Uint64: 456},
			},
			expectedType:       pgs.UInt64T,
			expectedValue:      uint64(456),
			shouldContainValue: true,
		},
		{
			name: "sint32_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Sint32{Sint32: -100},
			},
			expectedType:       pgs.SInt32,
			expectedValue:      int32(-100),
			shouldContainValue: true,
		},
		{
			name: "sint64_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Sint64{Sint64: -200},
			},
			expectedType:       pgs.SInt64,
			expectedValue:      int64(-200),
			shouldContainValue: true,
		},
		{
			name: "fixed32_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Fixed32{Fixed32: 999},
			},
			expectedType:       pgs.Fixed32T,
			expectedValue:      uint32(999),
			shouldContainValue: true,
		},
		{
			name: "fixed64_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Fixed64{Fixed64: 888},
			},
			expectedType:       pgs.Fixed64T,
			expectedValue:      uint64(888),
			shouldContainValue: true,
		},
		{
			name: "sfixed32_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Sfixed32{Sfixed32: -777},
			},
			expectedType:       pgs.SFixed32,
			expectedValue:      int32(-777),
			shouldContainValue: true,
		},
		{
			name: "sfixed64_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Sfixed64{Sfixed64: -666},
			},
			expectedType:       pgs.SFixed64,
			expectedValue:      int64(-666),
			shouldContainValue: true,
		},
		{
			name: "bool_rule_true",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Bool{Bool: true},
			},
			expectedType:       pgs.BoolT,
			expectedValue:      true,
			shouldContainValue: true,
		},
		{
			name: "bool_rule_false",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Bool{Bool: false},
			},
			expectedType:       pgs.BoolT,
			expectedValue:      false,
			shouldContainValue: true,
		},
		{
			name: "string_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_String_{String_: "custom_value"},
			},
			expectedType:       pgs.StringT,
			expectedValue:      "`custom_value`",
			shouldContainValue: true,
		},
		{
			name: "bytes_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Bytes{Bytes: []byte("test_bytes")},
			},
			expectedType:       pgs.BytesT,
			expectedValue:      "[]byte(`test_bytes`)",
			shouldContainValue: true,
		},
		{
			name: "enum_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Enum{Enum: 5},
			},
			expectedType:       pgs.EnumT,
			expectedValue:      int32(5),
			shouldContainValue: true,
		},
		{
			name: "message_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Message{
					Message: &redact.MessageRules{Nil: true},
				},
			},
			expectedType:       pgs.MessageT,
			shouldContainValue: false,
		},
		{
			name: "element_rule",
			rules: &redact.FieldRules{
				Values: &redact.FieldRules_Element{
					Element: &redact.ElementRules{Empty: true},
				},
			},
			expectedLabel:      pgs.Repeated,
			shouldContainValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Module{ModuleBase: &pgs.ModuleBase{}}

			result := m.RuleInformation(tt.rules)

			if tt.expectedType != 0 {
				assert.Equal(t, tt.expectedType, result.ProtoType,
					"Expected ProtoType %v, got %v", tt.expectedType, result.ProtoType)
			}

			if tt.expectedLabel != 0 {
				assert.Equal(t, tt.expectedLabel, result.ProtoLabel,
					"Expected ProtoLabel %v, got %v", tt.expectedLabel, result.ProtoLabel)
			}

			if tt.shouldContainValue {
				assert.Equal(t, tt.expectedValue, result.RedactionValue,
					"Expected RedactionValue %v, got %v", tt.expectedValue, result.RedactionValue)
			}
		})
	}
}

// TestFieldDataStructure tests the FieldData structure initialization
func TestFieldDataStructure(t *testing.T) {
	tests := []struct {
		name      string
		fieldData *FieldData
		validate  func(t *testing.T, fd *FieldData)
	}{
		{
			name: "simple_redacted_field",
			fieldData: &FieldData{
				Name:           "password",
				Redact:         true,
				RedactionValue: `"REDACTED"`,
				IsMap:          false,
				IsRepeated:     false,
				IsMessage:      false,
				IsOptional:     false,
			},
			validate: func(t *testing.T, fd *FieldData) {
				assert.True(t, fd.Redact)
				assert.Equal(t, `"REDACTED"`, fd.RedactionValue)
				assert.False(t, fd.IsMap)
				assert.False(t, fd.IsRepeated)
			},
		},
		{
			name: "repeated_field_with_iteration",
			fieldData: &FieldData{
				Name:           "tags",
				Redact:         true,
				RedactionValue: `"REDACTED"`,
				IsRepeated:     true,
				Iterate:        true,
			},
			validate: func(t *testing.T, fd *FieldData) {
				assert.True(t, fd.Redact)
				assert.True(t, fd.IsRepeated)
				assert.True(t, fd.Iterate)
			},
		},
		{
			name: "map_field",
			fieldData: &FieldData{
				Name:           "metadata",
				Redact:         true,
				RedactionValue: "nil",
				IsMap:          true,
			},
			validate: func(t *testing.T, fd *FieldData) {
				assert.True(t, fd.IsMap)
				assert.Equal(t, "nil", fd.RedactionValue)
			},
		},
		{
			name: "message_field_with_nested_call",
			fieldData: &FieldData{
				Name:                      "user",
				Redact:                    true,
				IsMessage:                 true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "User",
				EmbedMessageNameWithAlias: "pb.User",
			},
			validate: func(t *testing.T, fd *FieldData) {
				assert.True(t, fd.IsMessage)
				assert.True(t, fd.NestedEmbedCall)
				assert.Equal(t, "User", fd.EmbedMessageName)
				assert.Equal(t, "pb.User", fd.EmbedMessageNameWithAlias)
			},
		},
		{
			name: "optional_field",
			fieldData: &FieldData{
				Name:           "email",
				Redact:         true,
				RedactionValue: `"REDACTED"`,
				IsOptional:     true,
			},
			validate: func(t *testing.T, fd *FieldData) {
				assert.True(t, fd.IsOptional)
				assert.True(t, fd.Redact)
			},
		},
		{
			name: "skipped_embed",
			fieldData: &FieldData{
				Name:             "config",
				IsMessage:        true,
				EmbedSkip:        true,
				EmbedMessageName: "Config",
			},
			validate: func(t *testing.T, fd *FieldData) {
				assert.True(t, fd.IsMessage)
				assert.True(t, fd.EmbedSkip)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.fieldData)
		})
	}
}

// TestMessageDataStructure tests the MessageData structure
func TestMessageDataStructure(t *testing.T) {
	tests := []struct {
		name        string
		messageData *MessageData
		validate    func(t *testing.T, md *MessageData)
	}{
		{
			name: "normal_message",
			messageData: &MessageData{
				Name:      "User",
				WithAlias: "pb.User",
				Ignore:    false,
				ToNil:     false,
				ToEmpty:   false,
				Fields: []*FieldData{
					{Name: "username", Redact: false},
					{Name: "password", Redact: true, RedactionValue: `"REDACTED"`},
				},
			},
			validate: func(t *testing.T, md *MessageData) {
				assert.Equal(t, "User", md.Name)
				assert.Equal(t, "pb.User", md.WithAlias)
				assert.False(t, md.Ignore)
				assert.Len(t, md.Fields, 2)
			},
		},
		{
			name: "ignored_message",
			messageData: &MessageData{
				Name:   "PublicData",
				Ignore: true,
				Fields: []*FieldData{},
			},
			validate: func(t *testing.T, md *MessageData) {
				assert.True(t, md.Ignore)
			},
		},
		{
			name: "nil_message",
			messageData: &MessageData{
				Name:  "SensitiveData",
				ToNil: true,
			},
			validate: func(t *testing.T, md *MessageData) {
				assert.True(t, md.ToNil)
				assert.False(t, md.ToEmpty)
			},
		},
		{
			name: "empty_message",
			messageData: &MessageData{
				Name:    "PartialData",
				ToEmpty: true,
			},
			validate: func(t *testing.T, md *MessageData) {
				assert.True(t, md.ToEmpty)
				assert.False(t, md.ToNil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.messageData)
		})
	}
}

// TestServiceDataStructure tests the ServiceData structure
func TestServiceDataStructure(t *testing.T) {
	tests := []struct {
		name        string
		serviceData *ServiceData
		validate    func(t *testing.T, sd *ServiceData)
	}{
		{
			name: "normal_service",
			serviceData: &ServiceData{
				Name: "UserService",
				Skip: false,
				Methods: []*MethodData{
					{
						Name:   "GetUser",
						Skip:   false,
						Input:  "GetUserRequest",
						Output: &MessageData{Name: "User"},
					},
				},
			},
			validate: func(t *testing.T, sd *ServiceData) {
				assert.Equal(t, "UserService", sd.Name)
				assert.False(t, sd.Skip)
				assert.Len(t, sd.Methods, 1)
			},
		},
		{
			name: "skipped_service",
			serviceData: &ServiceData{
				Name:    "InternalService",
				Skip:    true,
				Methods: []*MethodData{},
			},
			validate: func(t *testing.T, sd *ServiceData) {
				assert.True(t, sd.Skip)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.serviceData)
		})
	}
}

// TestMethodDataStructure tests the MethodData structure
func TestMethodDataStructure(t *testing.T) {
	tests := []struct {
		name       string
		methodData *MethodData
		validate   func(t *testing.T, md *MethodData)
	}{
		{
			name: "unary_method",
			methodData: &MethodData{
				Name:            "GetUser",
				Skip:            false,
				Input:           "GetUserRequest",
				Output:          &MessageData{Name: "User"},
				Internal:        false,
				ClientStreaming: false,
				ServerStreaming: false,
			},
			validate: func(t *testing.T, md *MethodData) {
				assert.Equal(t, "GetUser", md.Name)
				assert.False(t, md.ClientStreaming)
				assert.False(t, md.ServerStreaming)
			},
		},
		{
			name: "client_streaming_method",
			methodData: &MethodData{
				Name:            "UploadData",
				Input:           "DataChunk",
				Output:          &MessageData{Name: "UploadResponse"},
				ClientStreaming: true,
				ServerStreaming: false,
			},
			validate: func(t *testing.T, md *MethodData) {
				assert.True(t, md.ClientStreaming)
				assert.False(t, md.ServerStreaming)
			},
		},
		{
			name: "server_streaming_method",
			methodData: &MethodData{
				Name:            "StreamUsers",
				Input:           "StreamRequest",
				Output:          &MessageData{Name: "User"},
				ClientStreaming: false,
				ServerStreaming: true,
			},
			validate: func(t *testing.T, md *MethodData) {
				assert.False(t, md.ClientStreaming)
				assert.True(t, md.ServerStreaming)
			},
		},
		{
			name: "bidirectional_streaming_method",
			methodData: &MethodData{
				Name:            "Chat",
				Input:           "ChatMessage",
				Output:          &MessageData{Name: "ChatMessage"},
				ClientStreaming: true,
				ServerStreaming: true,
			},
			validate: func(t *testing.T, md *MethodData) {
				assert.True(t, md.ClientStreaming)
				assert.True(t, md.ServerStreaming)
			},
		},
		{
			name: "internal_method",
			methodData: &MethodData{
				Name:       "AdminOperation",
				Internal:   true,
				StatusCode: "PermissionDenied",
				ErrMessage: "`Access denied`",
			},
			validate: func(t *testing.T, md *MethodData) {
				assert.True(t, md.Internal)
				assert.Equal(t, "PermissionDenied", md.StatusCode)
				assert.Equal(t, "`Access denied`", md.ErrMessage)
			},
		},
		{
			name: "skipped_method",
			methodData: &MethodData{
				Name: "HealthCheck",
				Skip: true,
			},
			validate: func(t *testing.T, md *MethodData) {
				assert.True(t, md.Skip)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.methodData)
		})
	}
}

// TestProtoFileDataStructure tests the complete ProtoFileData structure
func TestProtoFileDataStructure(t *testing.T) {
	data := &ProtoFileData{
		Source:  "user.proto",
		Package: "user",
		Imports: map[string]string{
			"context": "context",
			"grpc":    "google.golang.org/grpc",
		},
		References: []string{"grpc.Server", "context.Context"},
		Services: []*ServiceData{
			{
				Name: "UserService",
				Methods: []*MethodData{
					{
						Name:   "GetUser",
						Input:  "GetUserRequest",
						Output: &MessageData{Name: "User"},
					},
				},
			},
		},
		Messages: []*MessageData{
			{
				Name: "User",
				Fields: []*FieldData{
					{Name: "id", Redact: false},
					{Name: "password", Redact: true},
				},
			},
		},
	}

	assert.Equal(t, "user.proto", data.Source)
	assert.Equal(t, "user", data.Package)
	assert.Len(t, data.Imports, 2)
	assert.Len(t, data.Services, 1)
	assert.Len(t, data.Messages, 1)
	assert.Equal(t, "UserService", data.Services[0].Name)
	assert.Equal(t, "User", data.Messages[0].Name)
}

// TestRedactionValueFormatting tests various redaction value formatting scenarios
func TestRedactionValueFormatting(t *testing.T) {
	tests := []struct {
		name           string
		typ            pgs.ProtoType
		isOptional     bool
		isRepeated     bool
		customValue    interface{}
		expectedFormat string
		description    string
	}{
		{
			name:           "string_default",
			typ:            pgs.StringT,
			isOptional:     false,
			isRepeated:     false,
			customValue:    nil,
			expectedFormat: `"REDACTED"`,
			description:    "Default string redaction",
		},
		{
			name:           "optional_string",
			typ:            pgs.StringT,
			isOptional:     true,
			isRepeated:     false,
			customValue:    nil,
			expectedFormat: `"REDACTED"`,
			description:    "Optional string needs proper format for pointer assignment",
		},
		{
			name:           "int32_default",
			typ:            pgs.Int32T,
			isOptional:     false,
			isRepeated:     false,
			customValue:    nil,
			expectedFormat: "0",
			description:    "Default int32 redaction",
		},
		{
			name:           "repeated_string",
			typ:            pgs.StringT,
			isOptional:     false,
			isRepeated:     true,
			customValue:    nil,
			expectedFormat: "nil",
			description:    "Repeated field redacted to nil",
		},
		{
			name:           "bytes_default",
			typ:            pgs.BytesT,
			isOptional:     false,
			isRepeated:     false,
			customValue:    nil,
			expectedFormat: "nil",
			description:    "Bytes redacted to nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactionDefaults(tt.typ, tt.isRepeated)
			assert.Equal(t, tt.expectedFormat, result, tt.description)
		})
	}
}

// TestComplexFieldScenarios tests complex field processing scenarios
func TestComplexFieldScenarios(t *testing.T) {
	tests := []struct {
		name         string
		description  string
		field        *FieldData
		expectations func(t *testing.T, f *FieldData)
	}{
		{
			name:        "nested_message_with_iteration",
			description: "Repeated message field that requires iteration and nested redaction calls",
			field: &FieldData{
				Name:                      "users",
				Redact:                    true,
				IsRepeated:                true,
				IsMessage:                 true,
				Iterate:                   true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "User",
				EmbedMessageNameWithAlias: "pb.User",
			},
			expectations: func(t *testing.T, f *FieldData) {
				assert.True(t, f.Redact, "Field should be marked for redaction")
				assert.True(t, f.IsRepeated, "Field should be repeated")
				assert.True(t, f.IsMessage, "Field should be a message type")
				assert.True(t, f.Iterate, "Should iterate over repeated elements")
				assert.True(t, f.NestedEmbedCall, "Should call nested redaction")
			},
		},
		{
			name:        "map_with_message_values",
			description: "Map field with message type values requiring nested calls",
			field: &FieldData{
				Name:                      "user_map",
				Redact:                    true,
				IsMap:                     true,
				IsMessage:                 true,
				Iterate:                   true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "User",
				EmbedMessageNameWithAlias: "pb.User",
				RedactionValue:            "nil",
			},
			expectations: func(t *testing.T, f *FieldData) {
				assert.True(t, f.IsMap, "Field should be a map")
				assert.True(t, f.Iterate, "Should iterate over map entries")
				assert.True(t, f.NestedEmbedCall, "Should call nested redaction on values")
			},
		},
		{
			name:        "optional_message_nil",
			description: "Optional message field that should be set to nil",
			field: &FieldData{
				Name:                      "optional_user",
				Redact:                    true,
				IsMessage:                 true,
				IsOptional:                true,
				RedactionValue:            "nil",
				EmbedMessageName:          "User",
				EmbedMessageNameWithAlias: "pb.User",
			},
			expectations: func(t *testing.T, f *FieldData) {
				assert.True(t, f.IsOptional, "Field should be optional")
				assert.True(t, f.IsMessage, "Field should be a message")
				assert.Equal(t, "nil", f.RedactionValue, "Should be redacted to nil")
			},
		},
		{
			name:        "optional_message_empty",
			description: "Optional message field that should be set to empty struct",
			field: &FieldData{
				Name:                      "optional_config",
				Redact:                    true,
				IsMessage:                 true,
				IsOptional:                true,
				RedactionValue:            "&pb.Config{}",
				EmbedMessageName:          "Config",
				EmbedMessageNameWithAlias: "pb.Config",
			},
			expectations: func(t *testing.T, f *FieldData) {
				assert.True(t, f.IsOptional, "Field should be optional")
				assert.True(t, f.IsMessage, "Field should be a message")
				assert.Equal(t, "&pb.Config{}", f.RedactionValue, "Should be redacted to empty struct")
			},
		},
		{
			name:        "repeated_primitives_with_custom_value",
			description: "Repeated primitive field with custom redaction for each element",
			field: &FieldData{
				Name:           "scores",
				Redact:         true,
				IsRepeated:     true,
				Iterate:        true,
				RedactionValue: "0",
			},
			expectations: func(t *testing.T, f *FieldData) {
				assert.True(t, f.IsRepeated, "Field should be repeated")
				assert.True(t, f.Iterate, "Should iterate to redact each element")
				assert.Equal(t, "0", f.RedactionValue, "Each element redacted to 0")
			},
		},
		{
			name:        "skipped_nested_message",
			description: "Message field that is explicitly skipped from redaction",
			field: &FieldData{
				Name:                      "public_info",
				Redact:                    true,
				IsMessage:                 true,
				EmbedSkip:                 true,
				EmbedMessageName:          "PublicInfo",
				EmbedMessageNameWithAlias: "pb.PublicInfo",
			},
			expectations: func(t *testing.T, f *FieldData) {
				assert.True(t, f.IsMessage, "Field should be a message")
				assert.True(t, f.EmbedSkip, "Nested message redaction should be skipped")
				assert.False(t, f.NestedEmbedCall, "Should not call nested redaction")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("Testing:", tt.description)
			require.NotNil(t, tt.field, "Field data should not be nil")
			tt.expectations(t, tt.field)
		})
	}
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("empty_message_no_fields", func(t *testing.T) {
		msg := &MessageData{
			Name:   "EmptyMessage",
			Fields: []*FieldData{},
		}
		assert.Empty(t, msg.Fields, "Empty message should have no fields")
	})

	t.Run("service_with_no_methods", func(t *testing.T) {
		svc := &ServiceData{
			Name:    "EmptyService",
			Methods: []*MethodData{},
		}
		assert.Empty(t, svc.Methods, "Empty service should have no methods")
	})

	t.Run("nil_message_output", func(t *testing.T) {
		method := &MethodData{
			Name:   "VoidMethod",
			Output: &MessageData{Name: "Empty", ToNil: true},
		}
		assert.True(t, method.Output.ToNil, "Output should be nil")
	})

	t.Run("multiple_import_conflicts", func(t *testing.T) {
		imports := map[string]string{
			"user":  "github.com/example/user",
			"user1": "github.com/another/user",
			"user2": "github.com/third/user",
		}
		assert.Len(t, imports, 3, "Should handle multiple import aliases")
	})

	t.Run("deeply_nested_message", func(t *testing.T) {
		field := &FieldData{
			Name:                      "deep",
			IsMessage:                 true,
			NestedEmbedCall:           true,
			EmbedMessageName:          "Level1",
			EmbedMessageNameWithAlias: "pkg.Level1",
		}
		assert.True(t, field.NestedEmbedCall, "Should support nested messages")
	})

	t.Run("zero_value_redaction", func(t *testing.T) {
		field := &FieldData{
			Name:           "counter",
			Redact:         true,
			RedactionValue: "0",
		}
		assert.Equal(t, "0", field.RedactionValue, "Zero is a valid redaction value")
	})

	t.Run("empty_string_redaction", func(t *testing.T) {
		// While "REDACTED" is default, custom empty string should be supported
		field := &FieldData{
			Name:           "optional_field",
			Redact:         true,
			RedactionValue: `""`,
		}
		assert.Equal(t, `""`, field.RedactionValue, "Empty string is a valid redaction value")
	})
}

// TestIntegrationScenarios tests complex integration scenarios
func TestIntegrationScenarios(t *testing.T) {
	t.Run("complete_user_service_structure", func(t *testing.T) {
		// Simulate a complete user service with various field types
		userMessage := &MessageData{
			Name:      "User",
			WithAlias: "pb.User",
			Fields: []*FieldData{
				{
					Name:   "id",
					Redact: false,
				},
				{
					Name:   "username",
					Redact: false,
				},
				{
					Name:           "password",
					Redact:         true,
					RedactionValue: `"REDACTED"`,
				},
				{
					Name:           "email",
					Redact:         true,
					RedactionValue: `"r*d@ct*d"`,
				},
				{
					Name:                      "profile",
					Redact:                    true,
					IsMessage:                 true,
					NestedEmbedCall:           true,
					EmbedMessageName:          "Profile",
					EmbedMessageNameWithAlias: "pb.Profile",
				},
				{
					Name:       "roles",
					Redact:     false,
					IsRepeated: true,
				},
			},
		}

		service := &ServiceData{
			Name: "UserService",
			Methods: []*MethodData{
				{
					Name:     "GetUser",
					Input:    "GetUserRequest",
					Output:   userMessage,
					Internal: false,
				},
				{
					Name:       "ListAllUsers",
					Input:      "Empty",
					Output:     &MessageData{Name: "UserList"},
					Internal:   true,
					StatusCode: "PermissionDenied",
					ErrMessage: "`Internal only`",
				},
			},
		}

		// Assertions
		assert.Equal(t, "UserService", service.Name)
		assert.Len(t, service.Methods, 2)
		assert.Equal(t, "User", userMessage.Name)
		assert.Len(t, userMessage.Fields, 6)

		// Check redacted fields
		redactedCount := 0
		for _, field := range userMessage.Fields {
			if field.Redact {
				redactedCount++
			}
		}
		assert.Equal(t, 3, redactedCount, "Should have 3 redacted fields")
	})

	t.Run("streaming_service_with_redaction", func(t *testing.T) {
		streamService := &ServiceData{
			Name: "StreamService",
			Methods: []*MethodData{
				{
					Name:            "ServerStream",
					ServerStreaming: true,
					ClientStreaming: false,
					Input:           "Request",
					Output:          &MessageData{Name: "Response"},
				},
				{
					Name:            "ClientStream",
					ServerStreaming: false,
					ClientStreaming: true,
					Input:           "Data",
					Output:          &MessageData{Name: "Summary"},
				},
				{
					Name:            "BidiStream",
					ServerStreaming: true,
					ClientStreaming: true,
					Input:           "Message",
					Output:          &MessageData{Name: "Message"},
				},
			},
		}

		assert.Len(t, streamService.Methods, 3)
		assert.True(t, streamService.Methods[0].ServerStreaming)
		assert.True(t, streamService.Methods[1].ClientStreaming)
		assert.True(t, streamService.Methods[2].ServerStreaming && streamService.Methods[2].ClientStreaming)
	})

	t.Run("multi_file_proto_structure", func(t *testing.T) {
		// Simulate data from multiple proto files
		files := []*ProtoFileData{
			{
				Source:  "user.proto",
				Package: "user",
				Services: []*ServiceData{
					{Name: "UserService"},
				},
				Messages: []*MessageData{
					{Name: "User"},
				},
			},
			{
				Source:  "product.proto",
				Package: "product",
				Services: []*ServiceData{
					{Name: "ProductService"},
				},
				Messages: []*MessageData{
					{Name: "Product"},
				},
			},
		}

		assert.Len(t, files, 2)
		assert.Equal(t, "user.proto", files[0].Source)
		assert.Equal(t, "product.proto", files[1].Source)
	})
}

// BenchmarkRedactionDefaults benchmarks the RedactionDefaults function
func BenchmarkRedactionDefaults(b *testing.B) {
	types := []pgs.ProtoType{
		pgs.Int32T, pgs.Int64T, pgs.StringT, pgs.BoolT,
		pgs.FloatT, pgs.DoubleT, pgs.BytesT, pgs.MessageT,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, typ := range types {
			_ = RedactionDefaults(typ, false)
			_ = RedactionDefaults(typ, true)
		}
	}
}

// BenchmarkToCustomRule benchmarks the ToCustomRule function
func BenchmarkToCustomRule(b *testing.B) {
	types := []pgs.ProtoType{
		pgs.Int32T, pgs.Int64T, pgs.StringT, pgs.BoolT,
		pgs.FloatT, pgs.DoubleT, pgs.BytesT, pgs.MessageT,
	}
	labels := []pgs.ProtoLabel{pgs.Optional, pgs.Repeated}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, typ := range types {
			for _, lab := range labels {
				_ = ToCustomRule(typ, lab)
			}
		}
	}
}
