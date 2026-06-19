package main

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOneofDataStructure tests the OneofData and OneofFieldData types
func TestOneofDataStructure(t *testing.T) {
	tests := []struct {
		name     string
		oneof    *OneofData
		validate func(t *testing.T, o *OneofData)
	}{
		{
			name: "oneof_with_scalar_fields",
			oneof: &OneofData{
				Name: "Contact",
				Fields: []*OneofFieldData{
					{
						FieldData: &FieldData{
							Name:           "Email",
							Redact:         true,
							RedactionValue: "`r*d@ct*d`",
							FieldGoType:    "string",
						},
						WrapperTypeName: "OneofMessage_Email",
					},
					{
						FieldData: &FieldData{
							Name:           "Phone",
							Redact:         true,
							RedactionValue: "`XXX-XXX-XXXX`",
							FieldGoType:    "string",
						},
						WrapperTypeName: "OneofMessage_Phone",
					},
					{
						FieldData: &FieldData{
							Name:           "PhoneCode",
							Redact:         true,
							RedactionValue: "0",
							FieldGoType:    "int32",
						},
						WrapperTypeName: "OneofMessage_PhoneCode",
					},
				},
			},
			validate: func(t *testing.T, o *OneofData) {
				assert.Equal(t, "Contact", o.Name)
				assert.Len(t, o.Fields, 3)
				for _, f := range o.Fields {
					assert.True(t, f.Redact, "All fields in this oneof should be redacted")
					assert.NotEmpty(t, f.WrapperTypeName, "Should have wrapper type name")
				}
				assert.Equal(t, "OneofMessage_Email", o.Fields[0].WrapperTypeName)
				assert.Equal(t, "OneofMessage_Phone", o.Fields[1].WrapperTypeName)
				assert.Equal(t, "OneofMessage_PhoneCode", o.Fields[2].WrapperTypeName)
			},
		},
		{
			name: "oneof_with_message_fields",
			oneof: &OneofData{
				Name: "Payload",
				Fields: []*OneofFieldData{
					{
						FieldData: &FieldData{
							Name:                      "UserProfile",
							Redact:                    true,
							IsMessage:                 true,
							NestedEmbedCall:           true,
							EmbedMessageName:          "Profile",
							EmbedMessageNameWithAlias: "Profile",
						},
						WrapperTypeName: "OneofMessage_UserProfile",
					},
					{
						FieldData: &FieldData{
							Name:                      "UserSettings",
							Redact:                    true,
							IsMessage:                 true,
							RedactionValue:            "nil",
							EmbedMessageName:          "Settings",
							EmbedMessageNameWithAlias: "Settings",
						},
						WrapperTypeName: "OneofMessage_UserSettings",
					},
					{
						FieldData: &FieldData{
							Name:           "RawData",
							Redact:         true,
							RedactionValue: "`REDACTED`",
							FieldGoType:    "string",
						},
						WrapperTypeName: "OneofMessage_RawData",
					},
				},
			},
			validate: func(t *testing.T, o *OneofData) {
				assert.Equal(t, "Payload", o.Name)
				assert.Len(t, o.Fields, 3)

				// First field: message with nested redaction
				assert.True(t, o.Fields[0].IsMessage)
				assert.True(t, o.Fields[0].NestedEmbedCall)

				// Second field: message set to nil
				assert.True(t, o.Fields[1].IsMessage)
				assert.Equal(t, "nil", o.Fields[1].RedactionValue)

				// Third field: scalar
				assert.False(t, o.Fields[2].IsMessage)
				assert.Equal(t, "`REDACTED`", o.Fields[2].RedactionValue)
			},
		},
		{
			name: "oneof_with_mixed_redacted_and_safe_fields",
			oneof: &OneofData{
				Name: "Identifier",
				Fields: []*OneofFieldData{
					{
						FieldData: &FieldData{
							Name:           "Username",
							Redact:         true,
							RedactionValue: "`REDACTED`",
							FieldGoType:    "string",
						},
						WrapperTypeName: "OneofMessage_Username",
					},
					{
						FieldData: &FieldData{
							Name:        "PublicId",
							Redact:      false,
							FieldGoType: "string",
						},
						WrapperTypeName: "OneofMessage_PublicId",
					},
					{
						FieldData: &FieldData{
							Name:           "InternalId",
							Redact:         true,
							RedactionValue: "0",
							FieldGoType:    "int64",
						},
						WrapperTypeName: "OneofMessage_InternalId",
					},
				},
			},
			validate: func(t *testing.T, o *OneofData) {
				assert.Equal(t, "Identifier", o.Name)
				assert.Len(t, o.Fields, 3)

				// Count redacted fields
				redactedCount := 0
				for _, f := range o.Fields {
					if f.Redact {
						redactedCount++
					}
				}
				assert.Equal(t, 2, redactedCount, "Should have 2 redacted fields out of 3")

				// PublicId should not be redacted
				assert.False(t, o.Fields[1].Redact, "PublicId should not be redacted")
			},
		},
		{
			name: "oneof_with_message_skip",
			oneof: &OneofData{
				Name: "Data",
				Fields: []*OneofFieldData{
					{
						FieldData: &FieldData{
							Name:                      "SkippedMsg",
							Redact:                    true,
							IsMessage:                 true,
							EmbedSkip:                 true,
							EmbedMessageName:          "PublicData",
							EmbedMessageNameWithAlias: "PublicData",
						},
						WrapperTypeName: "OneofMessage_SkippedMsg",
					},
				},
			},
			validate: func(t *testing.T, o *OneofData) {
				assert.True(t, o.Fields[0].EmbedSkip, "Message should be marked as skip")
				assert.False(t, o.Fields[0].NestedEmbedCall, "Should not call nested redaction")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.oneof)
			tt.validate(t, tt.oneof)
		})
	}
}

// TestMessageDataWithOneofs tests MessageData containing oneof fields
func TestMessageDataWithOneofs(t *testing.T) {
	t.Run("message_with_oneofs_and_regular_fields", func(t *testing.T) {
		msg := &MessageData{
			Name:      "OneofMessage",
			WithAlias: "OneofMessage",
			Fields: []*FieldData{
				{Name: "Id", Redact: false, FieldGoType: "string"},
			},
			Oneofs: []*OneofData{
				{
					Name: "Contact",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Email",
								Redact:         true,
								RedactionValue: "`r*d@ct*d`",
								FieldGoType:    "string",
							},
							WrapperTypeName: "OneofMessage_Email",
						},
						{
							FieldData: &FieldData{
								Name:           "Phone",
								Redact:         true,
								RedactionValue: "`XXX-XXX-XXXX`",
								FieldGoType:    "string",
							},
							WrapperTypeName: "OneofMessage_Phone",
						},
					},
				},
			},
		}

		assert.Len(t, msg.Fields, 1, "Should have 1 regular field")
		assert.Len(t, msg.Oneofs, 1, "Should have 1 oneof group")
		assert.Len(t, msg.Oneofs[0].Fields, 2, "Oneof should have 2 fields")

		// Regular fields should NOT contain oneof fields
		for _, f := range msg.Fields {
			assert.NotEqual(t, "Email", f.Name, "Oneof fields should not be in regular Fields")
			assert.NotEqual(t, "Phone", f.Name, "Oneof fields should not be in regular Fields")
		}
	})

	t.Run("message_with_multiple_oneofs", func(t *testing.T) {
		msg := &MessageData{
			Name: "MultiOneofMessage",
			Oneofs: []*OneofData{
				{
					Name: "Contact",
					Fields: []*OneofFieldData{
						{
							FieldData:       &FieldData{Name: "Email", Redact: true},
							WrapperTypeName: "MultiOneofMessage_Email",
						},
					},
				},
				{
					Name: "Payload",
					Fields: []*OneofFieldData{
						{
							FieldData:       &FieldData{Name: "RawData", Redact: true},
							WrapperTypeName: "MultiOneofMessage_RawData",
						},
					},
				},
				{
					Name: "Identifier",
					Fields: []*OneofFieldData{
						{
							FieldData:       &FieldData{Name: "Username", Redact: true},
							WrapperTypeName: "MultiOneofMessage_Username",
						},
					},
				},
			},
		}

		assert.Len(t, msg.Oneofs, 3, "Should have 3 oneof groups")
		assert.Equal(t, "Contact", msg.Oneofs[0].Name)
		assert.Equal(t, "Payload", msg.Oneofs[1].Name)
		assert.Equal(t, "Identifier", msg.Oneofs[2].Name)
	})

	t.Run("message_with_no_oneofs", func(t *testing.T) {
		msg := &MessageData{
			Name: "RegularMessage",
			Fields: []*FieldData{
				{Name: "Id"},
				{Name: "Name"},
			},
		}

		assert.Nil(t, msg.Oneofs, "Message without oneofs should have nil Oneofs")
		assert.Len(t, msg.Fields, 2)
	})

	t.Run("ignored_message_with_oneofs", func(t *testing.T) {
		msg := &MessageData{
			Name:   "IgnoredMessage",
			Ignore: true,
			Oneofs: []*OneofData{
				{
					Name: "Choice",
					Fields: []*OneofFieldData{
						{
							FieldData:       &FieldData{Name: "OptionA", Redact: true},
							WrapperTypeName: "IgnoredMessage_OptionA",
						},
					},
				},
			},
		}

		assert.True(t, msg.Ignore, "Message should be ignored")
		// Even with oneofs present, ignore flag takes precedence
		assert.Len(t, msg.Oneofs, 1, "Oneofs are still populated but Ignore flag takes precedence in template")
	})
}

// TestOneofWrapperTypeNaming tests that wrapper type names follow Go protobuf conventions
func TestOneofWrapperTypeNaming(t *testing.T) {
	tests := []struct {
		name            string
		messageName     string
		fieldName       string
		expectedWrapper string
	}{
		{
			name:            "simple_string_field",
			messageName:     "MyMessage",
			fieldName:       "Email",
			expectedWrapper: "MyMessage_Email",
		},
		{
			name:            "snake_case_field",
			messageName:     "TestMessage",
			fieldName:       "PhoneCode",
			expectedWrapper: "TestMessage_PhoneCode",
		},
		{
			name:            "message_field",
			messageName:     "Container",
			fieldName:       "UserProfile",
			expectedWrapper: "Container_UserProfile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := tt.messageName + "_" + tt.fieldName
			assert.Equal(t, tt.expectedWrapper, wrapper,
				"Wrapper type should follow MessageName_FieldName convention")
		})
	}
}

// TestOneofFieldDataEmbedding tests that OneofFieldData properly embeds FieldData
func TestOneofFieldDataEmbedding(t *testing.T) {
	t.Run("access_embedded_fields", func(t *testing.T) {
		ofd := &OneofFieldData{
			FieldData: &FieldData{
				Name:           "Email",
				Redact:         true,
				RedactionValue: "`r*d@ct*d`",
				FieldGoType:    "string",
				IsMessage:      false,
			},
			WrapperTypeName: "MyMessage_Email",
		}

		// Should be able to access FieldData fields directly through embedding
		assert.Equal(t, "Email", ofd.Name)
		assert.True(t, ofd.Redact)
		assert.Equal(t, "`r*d@ct*d`", ofd.RedactionValue)
		assert.Equal(t, "string", ofd.FieldGoType)
		assert.False(t, ofd.IsMessage)
		assert.Equal(t, "MyMessage_Email", ofd.WrapperTypeName)
	})

	t.Run("message_type_in_oneof", func(t *testing.T) {
		ofd := &OneofFieldData{
			FieldData: &FieldData{
				Name:                      "Profile",
				Redact:                    true,
				IsMessage:                 true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "Profile",
				EmbedMessageNameWithAlias: "pb.Profile",
			},
			WrapperTypeName: "MyMessage_Profile",
		}

		assert.True(t, ofd.IsMessage)
		assert.True(t, ofd.NestedEmbedCall)
		assert.Equal(t, "Profile", ofd.EmbedMessageName)
	})
}

// TestOneofCompleteScenario tests a complete scenario with oneof in a full ProtoFileData
func TestOneofCompleteScenario(t *testing.T) {
	data := &ProtoFileData{
		Source:  "oneof_test.proto",
		Package: "testpkg",
		Imports: map[string]string{
			"context": "context",
			"grpc":    "google.golang.org/grpc",
			"redact":  "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
		},
		Messages: []*MessageData{
			{
				Name:      "OneofMessage",
				WithAlias: "OneofMessage",
				Fields: []*FieldData{
					{Name: "Id", Redact: false, FieldGoType: "string"},
				},
				Oneofs: []*OneofData{
					{
						Name: "Contact",
						Fields: []*OneofFieldData{
							{
								FieldData: &FieldData{
									Name:           "Email",
									Redact:         true,
									RedactionValue: "`r*d@ct*d`",
									FieldGoType:    "string",
								},
								WrapperTypeName: "OneofMessage_Email",
							},
							{
								FieldData: &FieldData{
									Name:           "Phone",
									Redact:         true,
									RedactionValue: "`XXX-XXX-XXXX`",
									FieldGoType:    "string",
								},
								WrapperTypeName: "OneofMessage_Phone",
							},
						},
					},
					{
						Name: "Payload",
						Fields: []*OneofFieldData{
							{
								FieldData: &FieldData{
									Name:                      "UserProfile",
									Redact:                    true,
									IsMessage:                 true,
									NestedEmbedCall:           true,
									EmbedMessageName:          "Profile",
									EmbedMessageNameWithAlias: "Profile",
								},
								WrapperTypeName: "OneofMessage_UserProfile",
							},
							{
								FieldData: &FieldData{
									Name:           "RawData",
									Redact:         true,
									RedactionValue: "`REDACTED`",
									FieldGoType:    "string",
								},
								WrapperTypeName: "OneofMessage_RawData",
							},
						},
					},
				},
			},
		},
	}

	require.Len(t, data.Messages, 1)
	msg := data.Messages[0]

	assert.Equal(t, "OneofMessage", msg.Name)
	assert.Len(t, msg.Fields, 1, "Regular fields")
	assert.Len(t, msg.Oneofs, 2, "Oneof groups")

	// Verify all oneof fields have redaction info
	totalOneofFields := 0
	redactedOneofFields := 0
	for _, oneof := range msg.Oneofs {
		for _, f := range oneof.Fields {
			totalOneofFields++
			if f.Redact {
				redactedOneofFields++
			}
		}
	}
	assert.Equal(t, 4, totalOneofFields, "Total oneof fields across all groups")
	assert.Equal(t, 4, redactedOneofFields, "All oneof fields should be redacted in this scenario")
}

// TestOneofEdgeCases tests edge cases for oneof handling
func TestOneofEdgeCases(t *testing.T) {
	t.Run("oneof_with_single_field", func(t *testing.T) {
		oneof := &OneofData{
			Name: "SingleChoice",
			Fields: []*OneofFieldData{
				{
					FieldData: &FieldData{
						Name:           "OnlyOption",
						Redact:         true,
						RedactionValue: "`REDACTED`",
						FieldGoType:    "string",
					},
					WrapperTypeName: "Msg_OnlyOption",
				},
			},
		}
		assert.Len(t, oneof.Fields, 1, "Oneof can have a single field")
	})

	t.Run("oneof_with_no_redacted_fields", func(t *testing.T) {
		oneof := &OneofData{
			Name: "PublicChoice",
			Fields: []*OneofFieldData{
				{
					FieldData:       &FieldData{Name: "OptionA", Redact: false},
					WrapperTypeName: "Msg_OptionA",
				},
				{
					FieldData:       &FieldData{Name: "OptionB", Redact: false},
					WrapperTypeName: "Msg_OptionB",
				},
			},
		}

		redactedCount := 0
		for _, f := range oneof.Fields {
			if f.Redact {
				redactedCount++
			}
		}
		assert.Equal(t, 0, redactedCount, "No fields should be redacted")
		assert.False(t, oneof.HasRedactableFields(), "Oneof with no redacted fields should return false")
	})

	t.Run("oneof_has_redactable_fields_helper", func(t *testing.T) {
		oneofWithRedact := &OneofData{
			Name: "Credential",
			Fields: []*OneofFieldData{
				{
					FieldData:       &FieldData{Name: "Password", Redact: true, RedactionValue: "``"},
					WrapperTypeName: "Msg_Password",
				},
				{
					FieldData:       &FieldData{Name: "Username", Redact: false},
					WrapperTypeName: "Msg_Username",
				},
			},
		}
		assert.True(t, oneofWithRedact.HasRedactableFields(), "Oneof with at least one redacted field should return true")

		emptyOneof := &OneofData{
			Name:   "Empty",
			Fields: []*OneofFieldData{},
		}
		assert.False(t, emptyOneof.HasRedactableFields(), "Empty oneof should return false")
	})

	t.Run("oneof_with_all_field_types", func(t *testing.T) {
		oneof := &OneofData{
			Name: "TypeVariety",
			Fields: []*OneofFieldData{
				{
					FieldData: &FieldData{
						Name:           "StrField",
						Redact:         true,
						RedactionValue: "`REDACTED`",
						FieldGoType:    "string",
					},
					WrapperTypeName: "Msg_StrField",
				},
				{
					FieldData: &FieldData{
						Name:           "IntField",
						Redact:         true,
						RedactionValue: "0",
						FieldGoType:    "int32",
					},
					WrapperTypeName: "Msg_IntField",
				},
				{
					FieldData: &FieldData{
						Name:           "BoolField",
						Redact:         true,
						RedactionValue: "false",
						FieldGoType:    "bool",
					},
					WrapperTypeName: "Msg_BoolField",
				},
				{
					FieldData: &FieldData{
						Name:                      "MsgField",
						Redact:                    true,
						IsMessage:                 true,
						NestedEmbedCall:           true,
						EmbedMessageName:          "SubMsg",
						EmbedMessageNameWithAlias: "SubMsg",
					},
					WrapperTypeName: "Msg_MsgField",
				},
			},
		}

		assert.Len(t, oneof.Fields, 4)
		// Check different types
		assert.Equal(t, "string", oneof.Fields[0].FieldGoType)
		assert.Equal(t, "int32", oneof.Fields[1].FieldGoType)
		assert.Equal(t, "bool", oneof.Fields[2].FieldGoType)
		assert.True(t, oneof.Fields[3].IsMessage)
	})
}

// TestOneofTemplateRendering tests that the template data is structured correctly
// for generating type switch code
func TestOneofTemplateRendering(t *testing.T) {
	t.Run("template_data_for_type_switch", func(t *testing.T) {
		// Simulate what the template would receive
		oneof := &OneofData{
			Name: "Contact",
			Fields: []*OneofFieldData{
				{
					FieldData: &FieldData{
						Name:           "Email",
						Redact:         true,
						RedactionValue: "`r*d@ct*d`",
						FieldGoType:    "string",
					},
					WrapperTypeName: "MyMessage_Email",
				},
				{
					FieldData: &FieldData{
						Name:        "PublicId",
						Redact:      false,
						FieldGoType: "string",
					},
					WrapperTypeName: "MyMessage_PublicId",
				},
			},
		}

		// The template would generate:
		// switch v := x.Contact.(type) {
		// case *MyMessage_Email:
		//     v.Email = `r*d@ct*d`
		// }
		// (no case for PublicId since Redact=false)

		// Verify the data is correct for template rendering
		assert.Equal(t, "Contact", oneof.Name, "Oneof name used in x.<Name>.(type)")
		assert.Equal(t, "MyMessage_Email", oneof.Fields[0].WrapperTypeName, "Wrapper type used in case statement")
		assert.Equal(t, "Email", oneof.Fields[0].Name, "Field name used in v.<Name>")
		assert.Equal(t, "`r*d@ct*d`", oneof.Fields[0].RedactionValue, "Redaction value assigned")
		assert.False(t, oneof.Fields[1].Redact, "Non-redacted field should not generate a case")
	})
}

// newTestTemplate parses the embedded redact template for testing.
func newTestTemplate(t *testing.T) *template.Template {
	t.Helper()
	tpl, err := template.New("redact").Parse(redactTpl)
	require.NoError(t, err, "Template should parse successfully")
	return tpl
}

// renderOneof is a helper that wraps a single message (with oneofs) into
// ProtoFileData and renders it through the template.
func renderOneof(t *testing.T, tpl *template.Template, msg *MessageData) string {
	t.Helper()
	data := &ProtoFileData{
		Source:  "test.proto",
		Package: "testpkg",
		Imports: map[string]string{
			"redact": "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
		},
		Messages: []*MessageData{msg},
	}
	var buf bytes.Buffer
	err := tpl.Execute(&buf, data)
	require.NoError(t, err, "Template execution should succeed")
	return buf.String()
}

// TestOneofTemplateExecution tests actual template rendering with oneof data
func TestOneofTemplateExecution(t *testing.T) {
	tpl, err := template.New("redact").Parse(redactTpl)
	require.NoError(t, err, "Template should parse successfully")

	t.Run("scalar_oneof_fields", func(t *testing.T) {
		data := &ProtoFileData{
			Source:  "test.proto",
			Package: "testpkg",
			Imports: map[string]string{
				"redact": "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
			},
			Messages: []*MessageData{
				{
					Name:      "MyMessage",
					WithAlias: "MyMessage",
					Fields: []*FieldData{
						{Name: "Id", Redact: false, FieldGoType: "string"},
					},
					Oneofs: []*OneofData{
						{
							Name: "Contact",
							Fields: []*OneofFieldData{
								{
									FieldData: &FieldData{
										Name:           "Email",
										Redact:         true,
										RedactionValue: "`r*d@ct*d`",
										FieldGoType:    "string",
									},
									WrapperTypeName: "MyMessage_Email",
								},
								{
									FieldData: &FieldData{
										Name:           "Phone",
										Redact:         true,
										RedactionValue: "`XXX-XXX-XXXX`",
										FieldGoType:    "string",
									},
									WrapperTypeName: "MyMessage_Phone",
								},
								{
									FieldData: &FieldData{
										Name:           "PhoneCode",
										Redact:         true,
										RedactionValue: "0",
										FieldGoType:    "int32",
									},
									WrapperTypeName: "MyMessage_PhoneCode",
								},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := tpl.Execute(&buf, data)
		require.NoError(t, err, "Template execution should succeed")

		output := buf.String()

		// Verify type switch is generated
		assert.Contains(t, output, "switch v := x.Contact.(type)", "Should have type switch on Contact")
		assert.Contains(t, output, "case *MyMessage_Email:", "Should have case for Email")
		assert.Contains(t, output, "case *MyMessage_Phone:", "Should have case for Phone")
		assert.Contains(t, output, "case *MyMessage_PhoneCode:", "Should have case for PhoneCode")

		// Verify field assignments use v. not x.
		assert.Contains(t, output, "v.Email = `r*d@ct*d`", "Should assign Email through v")
		assert.Contains(t, output, "v.Phone = `XXX-XXX-XXXX`", "Should assign Phone through v")
		assert.Contains(t, output, "v.PhoneCode = 0", "Should assign PhoneCode through v")
	})

	t.Run("message_oneof_fields", func(t *testing.T) {
		data := &ProtoFileData{
			Source:  "test.proto",
			Package: "testpkg",
			Imports: map[string]string{
				"redact": "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
			},
			Messages: []*MessageData{
				{
					Name:      "Container",
					WithAlias: "Container",
					Oneofs: []*OneofData{
						{
							Name: "Payload",
							Fields: []*OneofFieldData{
								{
									FieldData: &FieldData{
										Name:                      "Profile",
										Redact:                    true,
										IsMessage:                 true,
										NestedEmbedCall:           true,
										EmbedMessageName:          "Profile",
										EmbedMessageNameWithAlias: "Profile",
									},
									WrapperTypeName: "Container_Profile",
								},
								{
									FieldData: &FieldData{
										Name:                      "Settings",
										Redact:                    true,
										IsMessage:                 true,
										RedactionValue:            "nil",
										EmbedMessageName:          "Settings",
										EmbedMessageNameWithAlias: "Settings",
									},
									WrapperTypeName: "Container_Settings",
								},
								{
									FieldData: &FieldData{
										Name:           "RawData",
										Redact:         true,
										RedactionValue: "`REDACTED`",
										FieldGoType:    "string",
									},
									WrapperTypeName: "Container_RawData",
								},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := tpl.Execute(&buf, data)
		require.NoError(t, err, "Template execution should succeed")

		output := buf.String()

		// Verify type switch
		assert.Contains(t, output, "switch v := x.Payload.(type)", "Should have type switch on Payload")

		// Verify message with nested call
		assert.Contains(t, output, "case *Container_Profile:", "Should have case for Profile")
		assert.Contains(t, output, "redact.Apply(v.Profile)", "Should call redact.Apply through v")

		// Verify message set to nil
		assert.Contains(t, output, "case *Container_Settings:", "Should have case for Settings")
		assert.Contains(t, output, "v.Settings = nil", "Should set Settings to nil through v")

		// Verify scalar field
		assert.Contains(t, output, "case *Container_RawData:", "Should have case for RawData")
		assert.Contains(t, output, "v.RawData = `REDACTED`", "Should assign RawData through v")
	})

	t.Run("mixed_redacted_and_safe_oneof", func(t *testing.T) {
		data := &ProtoFileData{
			Source:  "test.proto",
			Package: "testpkg",
			Imports: map[string]string{
				"redact": "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
			},
			Messages: []*MessageData{
				{
					Name:      "Msg",
					WithAlias: "Msg",
					Oneofs: []*OneofData{
						{
							Name: "Identifier",
							Fields: []*OneofFieldData{
								{
									FieldData: &FieldData{
										Name:           "Username",
										Redact:         true,
										RedactionValue: "`REDACTED`",
										FieldGoType:    "string",
									},
									WrapperTypeName: "Msg_Username",
								},
								{
									FieldData: &FieldData{
										Name:        "PublicId",
										Redact:      false,
										FieldGoType: "string",
									},
									WrapperTypeName: "Msg_PublicId",
								},
								{
									FieldData: &FieldData{
										Name:           "InternalId",
										Redact:         true,
										RedactionValue: "0",
										FieldGoType:    "int64",
									},
									WrapperTypeName: "Msg_InternalId",
								},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := tpl.Execute(&buf, data)
		require.NoError(t, err, "Template execution should succeed")

		output := buf.String()

		// Redacted fields should have cases
		assert.Contains(t, output, "case *Msg_Username:", "Should have case for Username")
		assert.Contains(t, output, "case *Msg_InternalId:", "Should have case for InternalId")

		// Non-redacted field should NOT have a case
		assert.NotContains(t, output, "case *Msg_PublicId:", "Should NOT have case for non-redacted PublicId")
	})

	t.Run("oneof_with_no_redacted_fields_skips_switch", func(t *testing.T) {
		data := &ProtoFileData{
			Source:  "test.proto",
			Package: "testpkg",
			Imports: map[string]string{
				"redact": "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
			},
			Messages: []*MessageData{
				{
					Name:      "Msg",
					WithAlias: "Msg",
					Oneofs: []*OneofData{
						{
							Name: "PublicChoice",
							Fields: []*OneofFieldData{
								{
									FieldData: &FieldData{
										Name:        "DisplayName",
										Redact:      false,
										FieldGoType: "string",
									},
									WrapperTypeName: "Msg_DisplayName",
								},
								{
									FieldData: &FieldData{
										Name:        "Nickname",
										Redact:      false,
										FieldGoType: "string",
									},
									WrapperTypeName: "Msg_Nickname",
								},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := tpl.Execute(&buf, data)
		require.NoError(t, err, "Template execution should succeed")

		output := buf.String()

		// No type switch should be generated for oneofs with no redacted fields,
		// otherwise it would produce: switch v := x.PublicChoice.(type) {}
		// which causes "v declared and not used" compilation error.
		assert.NotContains(t, output, "switch v := x.PublicChoice.(type)", "Should NOT generate type switch for oneof with no redacted fields")
		assert.NotContains(t, output, "Msg_DisplayName", "Should NOT reference wrapper types for non-redacted oneof")
		assert.NotContains(t, output, "Msg_Nickname", "Should NOT reference wrapper types for non-redacted oneof")
	})

	t.Run("message_skip_in_oneof", func(t *testing.T) {
		data := &ProtoFileData{
			Source:  "test.proto",
			Package: "testpkg",
			Imports: map[string]string{
				"redact": "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
			},
			Messages: []*MessageData{
				{
					Name:      "Msg",
					WithAlias: "Msg",
					Oneofs: []*OneofData{
						{
							Name: "Data",
							Fields: []*OneofFieldData{
								{
									FieldData: &FieldData{
										Name:                      "SkippedMsg",
										Redact:                    true,
										IsMessage:                 true,
										EmbedSkip:                 true,
										EmbedMessageName:          "PublicData",
										EmbedMessageNameWithAlias: "PublicData",
									},
									WrapperTypeName: "Msg_SkippedMsg",
								},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := tpl.Execute(&buf, data)
		require.NoError(t, err, "Template execution should succeed")

		output := buf.String()

		assert.Contains(t, output, "case *Msg_SkippedMsg:", "Should have case for SkippedMsg")
		assert.Contains(t, output, "SkippedMsg redaction is skipped", "Should have skip comment")
	})

	t.Run("multiple_oneofs_in_one_message", func(t *testing.T) {
		data := &ProtoFileData{
			Source:  "test.proto",
			Package: "testpkg",
			Imports: map[string]string{
				"redact": "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
			},
			Messages: []*MessageData{
				{
					Name:      "MultiOneof",
					WithAlias: "MultiOneof",
					Fields: []*FieldData{
						{Name: "Id", Redact: false},
					},
					Oneofs: []*OneofData{
						{
							Name: "Contact",
							Fields: []*OneofFieldData{
								{
									FieldData: &FieldData{
										Name:           "Email",
										Redact:         true,
										RedactionValue: "`r*d@ct*d`",
										FieldGoType:    "string",
									},
									WrapperTypeName: "MultiOneof_Email",
								},
							},
						},
						{
							Name: "Payload",
							Fields: []*OneofFieldData{
								{
									FieldData: &FieldData{
										Name:           "RawData",
										Redact:         true,
										RedactionValue: "`REDACTED`",
										FieldGoType:    "string",
									},
									WrapperTypeName: "MultiOneof_RawData",
								},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := tpl.Execute(&buf, data)
		require.NoError(t, err, "Template execution should succeed")

		output := buf.String()

		// Both type switches should be present
		assert.Contains(t, output, "switch v := x.Contact.(type)", "Should have Contact type switch")
		assert.Contains(t, output, "switch v := x.Payload.(type)", "Should have Payload type switch")

		// Both cases should be present
		assert.Contains(t, output, "case *MultiOneof_Email:", "Should have Email case")
		assert.Contains(t, output, "case *MultiOneof_RawData:", "Should have RawData case")

		// Count the number of type switches
		switchCount := strings.Count(output, "switch v := x.")
		assert.Equal(t, 2, switchCount, "Should have exactly 2 type switches")
	})

	t.Run("ignored_message_skips_oneofs", func(t *testing.T) {
		data := &ProtoFileData{
			Source:  "test.proto",
			Package: "testpkg",
			Imports: map[string]string{
				"redact": "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
			},
			Messages: []*MessageData{
				{
					Name:      "IgnoredMsg",
					WithAlias: "IgnoredMsg",
					Ignore:    true,
					Oneofs: []*OneofData{
						{
							Name: "Choice",
							Fields: []*OneofFieldData{
								{
									FieldData: &FieldData{
										Name:           "OptionA",
										Redact:         true,
										RedactionValue: "`REDACTED`",
										FieldGoType:    "string",
									},
									WrapperTypeName: "IgnoredMsg_OptionA",
								},
							},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		err := tpl.Execute(&buf, data)
		require.NoError(t, err, "Template execution should succeed")

		output := buf.String()

		// Ignored message should not generate type switch
		assert.NotContains(t, output, "switch v := x.Choice.(type)", "Ignored message should skip oneofs")
		assert.Contains(t, output, "Ignoring message", "Should have ignore comment")
	})
}

// TestOneofBytesField tests bytes ([]byte) fields inside oneofs.
// In Go protobuf, bytes in a oneof wrapper struct is []byte, not *[]byte.
// Default redaction is nil; custom redaction is []byte(`value`).
func TestOneofBytesField(t *testing.T) {
	tpl := newTestTemplate(t)

	t.Run("bytes_default_nil", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Data",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "RawBytes",
								Redact:         true,
								RedactionValue: "nil",
								FieldGoType:    "[]byte",
							},
							WrapperTypeName: "Msg_RawBytes",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "case *Msg_RawBytes:")
		assert.Contains(t, output, "v.RawBytes = nil",
			"bytes field in oneof should be set to nil (not *nil, not &nil)")
	})

	t.Run("bytes_custom_value", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Data",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Secret",
								Redact:         true,
								RedactionValue: "[]byte(`redacted-bytes`)",
								FieldGoType:    "[]byte",
							},
							WrapperTypeName: "Msg_Secret",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "case *Msg_Secret:")
		assert.Contains(t, output, "v.Secret = []byte(`redacted-bytes`)",
			"bytes field should get a custom []byte literal assignment")
	})

	t.Run("bytes_empty_slice", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Payload",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Blob",
								Redact:         true,
								RedactionValue: "[]byte(``)",
								FieldGoType:    "[]byte",
							},
							WrapperTypeName: "Msg_Blob",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "v.Blob = []byte(``)",
			"bytes field should accept an empty []byte value")
	})
}

// TestOneofEnumField tests enum fields inside oneofs.
// In Go protobuf, enum values in a oneof wrapper are just the enum type
// which is an int32 alias. Redaction sets them to a numeric value.
func TestOneofEnumField(t *testing.T) {
	tpl := newTestTemplate(t)

	t.Run("enum_default_zero", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Value",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Status",
								Redact:         true,
								RedactionValue: "0",
								FieldGoType:    "",
							},
							WrapperTypeName: "Msg_Status",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "case *Msg_Status:")
		assert.Contains(t, output, "v.Status = 0",
			"enum field in oneof should default to 0 (first enum value)")
	})

	t.Run("enum_custom_value", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Value",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Priority",
								Redact:         true,
								RedactionValue: "2",
								FieldGoType:    "",
							},
							WrapperTypeName: "Msg_Priority",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "v.Priority = 2",
			"enum field should accept a custom numeric value")
	})
}

// TestOneofFloatDoubleFields tests float32 and float64 fields inside oneofs.
func TestOneofFloatDoubleFields(t *testing.T) {
	tpl := newTestTemplate(t)

	t.Run("float32_field", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Measurement",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Temperature",
								Redact:         true,
								RedactionValue: "3.14",
								FieldGoType:    "float32",
							},
							WrapperTypeName: "Msg_Temperature",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "case *Msg_Temperature:")
		assert.Contains(t, output, "v.Temperature = 3.14",
			"float32 field should get float literal assignment")
	})

	t.Run("float64_field", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Measurement",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Distance",
								Redact:         true,
								RedactionValue: "6.28",
								FieldGoType:    "float64",
							},
							WrapperTypeName: "Msg_Distance",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "v.Distance = 6.28")
	})

	t.Run("float_zero_default", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Measurement",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Weight",
								Redact:         true,
								RedactionValue: "0",
								FieldGoType:    "float32",
							},
							WrapperTypeName: "Msg_Weight",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "v.Weight = 0",
			"float field default redaction is 0, not 0.0")
	})
}

// TestOneofBoolField tests bool fields inside oneofs.
func TestOneofBoolField(t *testing.T) {
	tpl := newTestTemplate(t)

	t.Run("bool_false", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Flag",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "IsActive",
								Redact:         true,
								RedactionValue: "false",
								FieldGoType:    "bool",
							},
							WrapperTypeName: "Msg_IsActive",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "case *Msg_IsActive:")
		assert.Contains(t, output, "v.IsActive = false")
	})

	t.Run("bool_true_custom", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Flag",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Confirmed",
								Redact:         true,
								RedactionValue: "true",
								FieldGoType:    "bool",
							},
							WrapperTypeName: "Msg_Confirmed",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "v.Confirmed = true")
	})
}

// TestOneofAllNumericVariants tests every protobuf numeric type variant in a
// single oneof to make sure they all generate correct assignments.
// Covers: uint32, uint64, sint32, sint64, fixed32, fixed64, sfixed32, sfixed64.
func TestOneofAllNumericVariants(t *testing.T) {
	tpl := newTestTemplate(t)

	fields := []*OneofFieldData{
		{FieldData: &FieldData{Name: "ValUint32", Redact: true, RedactionValue: "32", FieldGoType: "uint32"}, WrapperTypeName: "Msg_ValUint32"},
		{FieldData: &FieldData{Name: "ValUint64", Redact: true, RedactionValue: "64", FieldGoType: "uint64"}, WrapperTypeName: "Msg_ValUint64"},
		{FieldData: &FieldData{Name: "ValSint32", Redact: true, RedactionValue: "32", FieldGoType: "int32"}, WrapperTypeName: "Msg_ValSint32"},
		{FieldData: &FieldData{Name: "ValSint64", Redact: true, RedactionValue: "64", FieldGoType: "int64"}, WrapperTypeName: "Msg_ValSint64"},
		{FieldData: &FieldData{Name: "ValFixed32", Redact: true, RedactionValue: "32", FieldGoType: "uint32"}, WrapperTypeName: "Msg_ValFixed32"},
		{FieldData: &FieldData{Name: "ValFixed64", Redact: true, RedactionValue: "64", FieldGoType: "uint64"}, WrapperTypeName: "Msg_ValFixed64"},
		{FieldData: &FieldData{Name: "ValSfixed32", Redact: true, RedactionValue: "32", FieldGoType: "int32"}, WrapperTypeName: "Msg_ValSfixed32"},
		{FieldData: &FieldData{Name: "ValSfixed64", Redact: true, RedactionValue: "64", FieldGoType: "int64"}, WrapperTypeName: "Msg_ValSfixed64"},
	}

	output := renderOneof(t, tpl, &MessageData{
		Name:      "Msg",
		WithAlias: "Msg",
		Oneofs:    []*OneofData{{Name: "Numbers", Fields: fields}},
	})

	for _, f := range fields {
		t.Run(f.WrapperTypeName, func(t *testing.T) {
			assert.Contains(t, output, "case *"+f.WrapperTypeName+":",
				"Should have case for wrapper type")
			assert.Contains(t, output, "v."+f.Name+" = "+f.RedactionValue,
				"Should assign redaction value through v")
		})
	}

	// There should be exactly 8 case statements in the single switch
	caseCount := strings.Count(output, "case *Msg_Val")
	assert.Equal(t, 8, caseCount, "All 8 numeric variants should produce a case")
}

// TestOneofCrossPackageMessage tests message fields whose Go type lives in a
// different package and needs an import alias (e.g. &pb.Profile{}).
func TestOneofCrossPackageMessage(t *testing.T) {
	tpl := newTestTemplate(t)

	t.Run("message_empty_with_alias", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Target",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:                      "ExtProfile",
								Redact:                    true,
								IsMessage:                 true,
								RedactionValue:            "&pb.Profile{}",
								EmbedMessageName:          "Profile",
								EmbedMessageNameWithAlias: "pb.Profile",
							},
							WrapperTypeName: "Msg_ExtProfile",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "case *Msg_ExtProfile:")
		assert.Contains(t, output, "v.ExtProfile = &pb.Profile{}",
			"Empty message redaction should use the aliased type")
		// Must NOT contain redact.Apply for a non-NestedEmbedCall, non-EmbedSkip message
		assert.NotContains(t, output, "redact.Apply(v.ExtProfile)")
	})

	t.Run("message_nested_apply_with_alias", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Target",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:                      "ExtSettings",
								Redact:                    true,
								IsMessage:                 true,
								NestedEmbedCall:           true,
								EmbedMessageName:          "Settings",
								EmbedMessageNameWithAlias: "otherpkg.Settings",
							},
							WrapperTypeName: "Msg_ExtSettings",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "redact.Apply(v.ExtSettings)",
			"Nested redaction call should go through v, not x")
		// Must NOT contain a direct assignment like v.ExtSettings = ...
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "v.ExtSettings =") {
				t.Error("Should not have direct assignment for NestedEmbedCall message")
			}
		}
	})

	t.Run("message_nil_with_alias", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "Msg",
			WithAlias: "Msg",
			Oneofs: []*OneofData{
				{
					Name: "Target",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:                      "ExtData",
								Redact:                    true,
								IsMessage:                 true,
								RedactionValue:            "nil",
								EmbedMessageName:          "Data",
								EmbedMessageNameWithAlias: "ext.Data",
							},
							WrapperTypeName: "Msg_ExtData",
						},
					},
				},
			},
		})

		assert.Contains(t, output, "v.ExtData = nil",
			"Message set to nil should assign nil through v")
	})
}

// TestOneofMixedWithOptionalFields ensures that proto3 optional (synthetic
// oneof) fields stay in regular Fields while real oneofs go into Oneofs.
// The generated code for optional scalars uses pointer tmp-variables while
// oneof fields use the type-switch pattern — both must coexist.
func TestOneofMixedWithOptionalFields(t *testing.T) {
	tpl := newTestTemplate(t)

	output := renderOneof(t, tpl, &MessageData{
		Name:      "Mixed",
		WithAlias: "Mixed",
		// proto3 optional fields land in Fields with IsOptional=true
		Fields: []*FieldData{
			{Name: "Id", Redact: false, FieldGoType: "string"},
			{
				Name:           "OptionalEmail",
				Redact:         true,
				RedactionValue: "`r*d@ct*d`",
				FieldGoType:    "string",
				IsOptional:     true,
			},
			{
				Name:           "OptionalAge",
				Redact:         true,
				RedactionValue: "0",
				FieldGoType:    "int32",
				IsOptional:     true,
			},
			{
				// proto3 optional bytes — NOT a pointer, stays []byte
				Name:           "OptionalSig",
				Redact:         true,
				RedactionValue: "nil",
				FieldGoType:    "[]byte",
				IsOptional:     false, // bytes are never pointer-optional
			},
		},
		// Real oneof
		Oneofs: []*OneofData{
			{
				Name: "Contact",
				Fields: []*OneofFieldData{
					{
						FieldData: &FieldData{
							Name:           "Phone",
							Redact:         true,
							RedactionValue: "`XXX`",
							FieldGoType:    "string",
						},
						WrapperTypeName: "Mixed_Phone",
					},
					{
						FieldData: &FieldData{
							Name:           "Fax",
							Redact:         true,
							RedactionValue: "`000`",
							FieldGoType:    "string",
						},
						WrapperTypeName: "Mixed_Fax",
					},
				},
			},
		},
	})

	// --- optional fields should use the tmp-pointer pattern ---
	assert.Contains(t, output, "OptionalEmailTmp := `r*d@ct*d`",
		"Optional string should use temp variable")
	assert.Contains(t, output, "x.OptionalEmail = &OptionalEmailTmp",
		"Optional string should assign through pointer")

	assert.Contains(t, output, "OptionalAgeTmp := int32(0)",
		"Optional int32 should cast to go type in temp variable")
	assert.Contains(t, output, "x.OptionalAge = &OptionalAgeTmp",
		"Optional int32 should assign through pointer")

	// bytes optional is NOT a pointer — direct assignment
	assert.Contains(t, output, "x.OptionalSig = nil",
		"bytes optional should be a direct assignment, not pointer")
	assert.NotContains(t, output, "OptionalSigTmp",
		"bytes optional must NOT use a tmp variable")

	// --- oneof fields should use the type-switch pattern ---
	assert.Contains(t, output, "switch v := x.Contact.(type)")
	assert.Contains(t, output, "case *Mixed_Phone:")
	assert.Contains(t, output, "v.Phone = `XXX`")
	assert.Contains(t, output, "case *Mixed_Fax:")
	assert.Contains(t, output, "v.Fax = `000`")

	// oneof fields must NOT appear as direct x. assignments
	assert.NotContains(t, output, "x.Phone =")
	assert.NotContains(t, output, "x.Fax =")
}

// TestOneofToNilMessageSkipsOneofs verifies that a message with the ToNil
// option skips all oneof redaction (same as ToEmpty and Ignore).
func TestOneofToNilMessageSkipsOneofs(t *testing.T) {
	tpl := newTestTemplate(t)

	t.Run("to_nil_message", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "SensitiveMsg",
			WithAlias: "SensitiveMsg",
			ToNil:     true,
			Oneofs: []*OneofData{
				{
					Name: "Secret",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Key",
								Redact:         true,
								RedactionValue: "nil",
								FieldGoType:    "[]byte",
							},
							WrapperTypeName: "SensitiveMsg_Key",
						},
					},
				},
			},
		})

		assert.NotContains(t, output, "switch v := x.Secret.(type)",
			"ToNil message should not generate oneof type switches")
		assert.Contains(t, output, "Message will be set to nil")
	})

	t.Run("to_empty_message", func(t *testing.T) {
		output := renderOneof(t, tpl, &MessageData{
			Name:      "EmptyMsg",
			WithAlias: "EmptyMsg",
			ToEmpty:   true,
			Oneofs: []*OneofData{
				{
					Name: "Choice",
					Fields: []*OneofFieldData{
						{
							FieldData: &FieldData{
								Name:           "Token",
								Redact:         true,
								RedactionValue: "`REDACTED`",
								FieldGoType:    "string",
							},
							WrapperTypeName: "EmptyMsg_Token",
						},
					},
				},
			},
		})

		assert.NotContains(t, output, "switch v := x.Choice.(type)",
			"ToEmpty message should not generate oneof type switches")
		assert.Contains(t, output, "Message will be set to empty")
	})
}

// TestOneofEveryScalarType renders a oneof that contains every protobuf scalar
// type to verify the template handles them all.  This is the "kitchen sink"
// test that catches regressions for any type.
func TestOneofEveryScalarType(t *testing.T) {
	tpl := newTestTemplate(t)

	// Each entry: field name, redaction value, Go type, expected output
	types := []struct {
		name    string
		value   string
		goType  string
		wantOut string // exact "v.<name> = <value>" we expect
	}{
		{"Int32Val", "32", "int32", "v.Int32Val = 32"},
		{"Int64Val", "64", "int64", "v.Int64Val = 64"},
		{"Uint32Val", "32", "uint32", "v.Uint32Val = 32"},
		{"Uint64Val", "64", "uint64", "v.Uint64Val = 64"},
		{"Sint32Val", "32", "int32", "v.Sint32Val = 32"},
		{"Sint64Val", "64", "int64", "v.Sint64Val = 64"},
		{"Fixed32Val", "32", "uint32", "v.Fixed32Val = 32"},
		{"Fixed64Val", "64", "uint64", "v.Fixed64Val = 64"},
		{"Sfixed32Val", "32", "int32", "v.Sfixed32Val = 32"},
		{"Sfixed64Val", "64", "int64", "v.Sfixed64Val = 64"},
		{"FloatVal", "3.2", "float32", "v.FloatVal = 3.2"},
		{"DoubleVal", "6.4", "float64", "v.DoubleVal = 6.4"},
		{"BoolVal", "false", "bool", "v.BoolVal = false"},
		{"StringVal", "`REDACTED`", "string", "v.StringVal = `REDACTED`"},
		{"BytesVal", "[]byte(`secret`)", "[]byte", "v.BytesVal = []byte(`secret`)"},
		{"EnumVal", "2", "", "v.EnumVal = 2"},
	}

	var fields []*OneofFieldData
	for _, tt := range types {
		fields = append(fields, &OneofFieldData{
			FieldData: &FieldData{
				Name:           tt.name,
				Redact:         true,
				RedactionValue: tt.value,
				FieldGoType:    tt.goType,
			},
			WrapperTypeName: "Msg_" + tt.name,
		})
	}

	output := renderOneof(t, tpl, &MessageData{
		Name:      "Msg",
		WithAlias: "Msg",
		Oneofs:    []*OneofData{{Name: "Val", Fields: fields}},
	})

	assert.Contains(t, output, "switch v := x.Val.(type)")

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, output, "case *Msg_"+tt.name+":")
			assert.Contains(t, output, tt.wantOut,
				"Generated output must contain the exact assignment")
		})
	}

	// Verify correct case count
	caseCount := strings.Count(output, "case *Msg_")
	assert.Equal(t, len(types), caseCount,
		"Should have exactly one case per scalar type")
}

// TestOneofMessageAllVariants tests every message-field variant that can appear
// inside a oneof: apply (nested), nil, empty, empty-with-alias, skip.
func TestOneofMessageAllVariants(t *testing.T) {
	tpl := newTestTemplate(t)

	output := renderOneof(t, tpl, &MessageData{
		Name:      "Msg",
		WithAlias: "Msg",
		Oneofs: []*OneofData{
			{
				Name: "Target",
				Fields: []*OneofFieldData{
					// 1. NestedEmbedCall (apply)
					{
						FieldData: &FieldData{
							Name:                      "Applied",
							Redact:                    true,
							IsMessage:                 true,
							NestedEmbedCall:           true,
							EmbedMessageName:          "Profile",
							EmbedMessageNameWithAlias: "Profile",
						},
						WrapperTypeName: "Msg_Applied",
					},
					// 2. Nil
					{
						FieldData: &FieldData{
							Name:                      "Nulled",
							Redact:                    true,
							IsMessage:                 true,
							RedactionValue:            "nil",
							EmbedMessageName:          "Settings",
							EmbedMessageNameWithAlias: "Settings",
						},
						WrapperTypeName: "Msg_Nulled",
					},
					// 3. Empty (same package)
					{
						FieldData: &FieldData{
							Name:                      "Emptied",
							Redact:                    true,
							IsMessage:                 true,
							RedactionValue:            "&Config{}",
							EmbedMessageName:          "Config",
							EmbedMessageNameWithAlias: "Config",
						},
						WrapperTypeName: "Msg_Emptied",
					},
					// 4. Empty (cross-package with alias)
					{
						FieldData: &FieldData{
							Name:                      "ExtEmptied",
							Redact:                    true,
							IsMessage:                 true,
							RedactionValue:            "&ext.Config{}",
							EmbedMessageName:          "Config",
							EmbedMessageNameWithAlias: "ext.Config",
						},
						WrapperTypeName: "Msg_ExtEmptied",
					},
					// 5. Skip
					{
						FieldData: &FieldData{
							Name:                      "Skipped",
							Redact:                    true,
							IsMessage:                 true,
							EmbedSkip:                 true,
							EmbedMessageName:          "PublicData",
							EmbedMessageNameWithAlias: "PublicData",
						},
						WrapperTypeName: "Msg_Skipped",
					},
				},
			},
		},
	})

	// 1. apply
	assert.Contains(t, output, "case *Msg_Applied:")
	assert.Contains(t, output, "redact.Apply(v.Applied)")

	// 2. nil
	assert.Contains(t, output, "case *Msg_Nulled:")
	assert.Contains(t, output, "v.Nulled = nil")

	// 3. empty same pkg
	assert.Contains(t, output, "case *Msg_Emptied:")
	assert.Contains(t, output, "v.Emptied = &Config{}")

	// 4. empty cross pkg
	assert.Contains(t, output, "case *Msg_ExtEmptied:")
	assert.Contains(t, output, "v.ExtEmptied = &ext.Config{}",
		"Cross-package empty struct must use the aliased type")

	// 5. skip
	assert.Contains(t, output, "case *Msg_Skipped:")
	assert.Contains(t, output, "Skipped redaction is skipped")
	// skip must not have an assignment or redact.Apply
	assert.NotContains(t, output, "v.Skipped = ")
	assert.NotContains(t, output, "redact.Apply(v.Skipped)")
}

// TestOneofRegularFieldsNotAffected ensures that adding a oneof does not
// change how regular (non-oneof) fields are generated.
func TestOneofRegularFieldsNotAffected(t *testing.T) {
	tpl := newTestTemplate(t)

	output := renderOneof(t, tpl, &MessageData{
		Name:      "Msg",
		WithAlias: "Msg",
		Fields: []*FieldData{
			{Name: "Password", Redact: true, RedactionValue: `"REDACTED"`, FieldGoType: "string"},
			{Name: "Score", Redact: true, RedactionValue: "0", FieldGoType: "int32"},
			{
				Name:                      "Info",
				Redact:                    true,
				IsMessage:                 true,
				NestedEmbedCall:           true,
				EmbedMessageName:          "Info",
				EmbedMessageNameWithAlias: "Info",
			},
		},
		Oneofs: []*OneofData{
			{
				Name: "Choice",
				Fields: []*OneofFieldData{
					{
						FieldData:       &FieldData{Name: "A", Redact: true, RedactionValue: "`a`", FieldGoType: "string"},
						WrapperTypeName: "Msg_A",
					},
				},
			},
		},
	})

	// Regular fields use x. prefix
	assert.Contains(t, output, `x.Password = "REDACTED"`)
	assert.Contains(t, output, "x.Score = 0")
	assert.Contains(t, output, "redact.Apply(x.Info)")

	// Oneof field uses v. prefix inside switch
	assert.Contains(t, output, "switch v := x.Choice.(type)")
	assert.Contains(t, output, "v.A = `a`")
	// Oneof field must NOT appear as x.A
	assert.NotContains(t, output, "x.A =")
}
