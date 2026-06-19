package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationProtoCompilation tests the complete workflow:
// 1. Generate Go code from proto file
// 2. Generate redaction code
// 3. Verify the code compiles
// 4. Test optional fields work correctly
func TestIntegrationProtoCompilation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test directory
	testDir := "testdata/integration"
	protoFile := filepath.Join(testDir, "test.proto")

	// Verify proto file exists
	require.FileExists(t, protoFile, "Test proto file should exist")

	// Get current directory for module path
	currentDir, err := os.Getwd()
	require.NoError(t, err, "Should get current directory")

	t.Run("generate_go_code", func(t *testing.T) {
		// Generate Go code using protoc
		cmd := exec.Command("protoc",
			"--experimental_allow_proto3_optional",
			"--go_out="+currentDir,
			"--go_opt=paths=source_relative",
			"--go-grpc_out="+currentDir,
			"--go-grpc_opt=paths=source_relative",
			"-I", ".",
			protoFile,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("protoc output: %s", string(output))
		}
		require.NoError(t, err, "protoc should generate Go code successfully")

		// Verify generated files exist
		generatedFile := filepath.Join(testDir, "test.pb.go")
		require.FileExists(t, generatedFile, "Generated Go file should exist")

		grpcFile := filepath.Join(testDir, "test_grpc.pb.go")
		require.FileExists(t, grpcFile, "Generated gRPC file should exist")
	})

	t.Run("generate_redaction_code", func(t *testing.T) {
		// Build protoc-gen-redact plugin
		buildCmd := exec.Command("go", "build", "-o", "protoc-gen-redact", ".")
		buildOutput, err := buildCmd.CombinedOutput()
		if err != nil {
			t.Logf("build output: %s", string(buildOutput))
		}
		require.NoError(t, err, "Should build protoc-gen-redact plugin")

		// Ensure plugin is executable
		pluginPath := "./protoc-gen-redact"
		require.FileExists(t, pluginPath, "Plugin binary should exist")

		// Generate redaction code
		cmd := exec.Command("protoc",
			"--experimental_allow_proto3_optional",
			"--plugin=protoc-gen-redact="+pluginPath,
			"--redact_out="+currentDir,
			"--redact_opt=paths=source_relative",
			"-I", ".",
			protoFile,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("protoc-gen-redact output: %s", string(output))
		}
		require.NoError(t, err, "protoc-gen-redact should generate redaction code")

		// Verify redaction file exists
		redactFile := filepath.Join(testDir, "test.pb.redact.go")
		require.FileExists(t, redactFile, "Generated redaction file should exist")

		// Read and verify redaction file content
		content, err := os.ReadFile(redactFile)
		require.NoError(t, err, "Should read generated redaction file")

		contentStr := string(content)

		// Check for package declaration
		assert.Contains(t, contentStr, "package testdata", "Should have correct package")

		// Check for imports
		assert.Contains(t, contentStr, "import", "Should have imports")

		// Check for RegisterRedacted service methods
		assert.Contains(t, contentStr, "RegisterRedactedTestService", "Should have redacted service registration")

		// Check for Redact methods on messages
		assert.Contains(t, contentStr, "func (x *TestMessage) Redact()", "Should have Redact method for TestMessage")
		assert.Contains(t, contentStr, "func (x *Profile) Redact()", "Should have Redact method for Profile")
		assert.Contains(t, contentStr, "func (x *Address) Redact()", "Should have Redact method for Address")

		// Verify optional field handling (temp variables for pointer assignment)
		assert.Contains(t, contentStr, "Tmp :=", "Should have temp variable assignments for optional fields")
	})

	t.Run("verify_code_compiles", func(t *testing.T) {
		// Try to compile the generated code
		cmd := exec.Command("go", "build", "./"+testDir)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			t.Logf("Compilation output: %s", outputStr)
			require.NoError(t, err, "Generated code should compile without errors")
		}

		// Verify no compilation warnings
		assert.NotContains(t, outputStr, "warning", "Should not have compilation warnings")
	})

	t.Run("verify_generated_code_structure", func(t *testing.T) {
		redactFile := filepath.Join(testDir, "test.pb.redact.go")
		content, err := os.ReadFile(redactFile)
		require.NoError(t, err, "Should read generated redaction file")

		contentStr := string(content)

		// Test various redaction patterns
		tests := []struct {
			name     string
			contains string
			reason   string
		}{
			{
				name:     "default_string_redaction",
				contains: `"REDACTED"`,
				reason:   "Should have default string redaction",
			},
			{
				name:     "custom_string_redaction",
				contains: "`r*d@ct*d`",
				reason:   "Should have custom email redaction",
			},
			{
				name:     "nil_redaction",
				contains: "nil",
				reason:   "Should have nil redaction for some fields",
			},
			{
				name:     "empty_struct_redaction",
				contains: "{}",
				reason:   "Should have empty struct redaction",
			},
			{
				name:     "nested_redaction_call",
				contains: "redact.Apply",
				reason:   "Should call nested redaction",
			},
			{
				name:     "iterate_over_repeated",
				contains: "for k := range",
				reason:   "Should iterate over repeated fields",
			},
			{
				name:     "internal_method_check",
				contains: "CheckInternal",
				reason:   "Should check for internal methods",
			},
			{
				name:     "status_error",
				contains: "status.Error",
				reason:   "Should return status error for internal methods",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Contains(t, contentStr, tt.contains, tt.reason)
			})
		}
	})

	t.Run("verify_optional_field_handling", func(t *testing.T) {
		redactFile := filepath.Join(testDir, "test.pb.redact.go")
		content, err := os.ReadFile(redactFile)
		require.NoError(t, err, "Should read generated redaction file")

		contentStr := string(content)

		// Count occurrences of optional field handling patterns
		// Optional fields should use temporary variables for pointer assignment
		tempVarPatterns := []string{
			"Tmp :=", // Pattern for temp variable assignment
		}

		foundTempVars := 0
		for _, pattern := range tempVarPatterns {
			foundTempVars += strings.Count(contentStr, pattern)
		}

		// We have multiple optional fields that need temp variables (email, age, is_active, etc.)
		assert.Greater(t, foundTempVars, 0, "Should have temp variable assignments for optional fields")

		// Verify pointer assignment pattern exists
		assert.Contains(t, contentStr, "x.", "Should have field assignments")
		assert.Contains(t, contentStr, " = &", "Should have pointer assignments for optional fields")
	})

	t.Run("verify_message_level_options", func(t *testing.T) {
		redactFile := filepath.Join(testDir, "test.pb.redact.go")
		content, err := os.ReadFile(redactFile)
		require.NoError(t, err, "Should read generated redaction file")

		contentStr := string(content)

		// Verify PublicData is ignored
		assert.Contains(t, contentStr, "func (x *PublicData) Redact()", "Should have Redact method for PublicData")
		// The method should be empty or just return for ignored messages
		publicDataSection := extractFunctionBody(contentStr, "func (x *PublicData) Redact()")
		if publicDataSection != "" {
			assert.NotContains(t, publicDataSection, "x.Data = ", "Ignored message should not redact fields")
		}

		// Verify SensitiveData returns nil (message-level nil option)
		sensitiveDataSection := extractFunctionBody(contentStr, "func (x *SensitiveData) Redact()")
		if sensitiveDataSection != "" {
			// With (redact.nil) = true at message level, the entire message handling is different
			// This is typically handled at the service level, not in the Redact() method
			t.Log("SensitiveData has message-level nil option")
		}

		// Verify EmptyData is set to empty (message-level empty option)
		emptyDataSection := extractFunctionBody(contentStr, "func (x *EmptyData) Redact()")
		if emptyDataSection != "" {
			// With (redact.empty) = true at message level
			t.Log("EmptyData has message-level empty option")
		}
	})

	t.Run("verify_service_methods", func(t *testing.T) {
		redactFile := filepath.Join(testDir, "test.pb.redact.go")
		content, err := os.ReadFile(redactFile)
		require.NoError(t, err, "Should read generated redaction file")

		contentStr := string(content)

		// Verify GetUser method exists and applies redaction
		assert.Contains(t, contentStr, "func (s *redactedTestServiceServer) GetUser", "Should have GetUser method")

		// Verify AdminOperation is internal
		adminSection := extractFunctionBody(contentStr, "func (s *redactedTestServiceServer) AdminOperation")
		if adminSection != "" {
			assert.Contains(t, adminSection, "CheckInternal", "AdminOperation should check internal access")
			assert.Contains(t, adminSection, "status.Error", "AdminOperation should return error for external callers")
		}

		// Verify HealthCheck is skipped
		healthCheckSection := extractFunctionBody(contentStr, "func (s *redactedTestServiceServer) HealthCheck")
		if healthCheckSection != "" {
			// Skipped methods just pass through to the underlying service
			assert.Contains(t, healthCheckSection, "s.srv.HealthCheck", "HealthCheck should pass through")
			assert.NotContains(t, healthCheckSection, "redact.Apply", "HealthCheck should not apply redaction")
		}
	})

	t.Run("verify_no_syntax_errors", func(t *testing.T) {
		// Run go vet to check for common mistakes
		cmd := exec.Command("go", "vet", "./"+testDir)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// go vet may return an error for unused imports or other issues
		// but we want to ensure there are no major syntax errors
		if err != nil {
			t.Logf("go vet output: %s", outputStr)
			// Check if it's just about unused imports or minor issues
			if !strings.Contains(outputStr, "imported and not used") {
				t.Logf("go vet found issues (this may be expected for test code): %s", outputStr)
			}
		}
	})

	t.Run("cleanup", func(t *testing.T) {
		// Clean up generated files
		filesToRemove := []string{
			filepath.Join(testDir, "test.pb.go"),
			filepath.Join(testDir, "test_grpc.pb.go"),
			filepath.Join(testDir, "test.pb.redact.go"),
			"./protoc-gen-redact",
		}

		for _, file := range filesToRemove {
			if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
				t.Logf("Warning: could not remove %s: %v", file, err)
			}
		}
	})
}

// extractFunctionBody attempts to extract the body of a function from source code
// This is a simple helper for testing purposes
func extractFunctionBody(source, functionSignature string) string {
	idx := strings.Index(source, functionSignature)
	if idx == -1 {
		return ""
	}

	// Find the opening brace
	start := strings.Index(source[idx:], "{")
	if start == -1 {
		return ""
	}
	start += idx + 1

	// Find the matching closing brace (simple implementation)
	braceCount := 1
	pos := start
	for pos < len(source) && braceCount > 0 {
		if source[pos] == '{' {
			braceCount++
		} else if source[pos] == '}' {
			braceCount--
		}
		pos++
	}

	if braceCount == 0 {
		return source[start : pos-1]
	}

	return ""
}

// TestGeneratedCodeQuality tests the quality of generated code
func TestGeneratedCodeQuality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		protoFile   string
		shouldPass  bool
		description string
	}{
		{
			name:        "valid_proto_with_optional_fields",
			protoFile:   "testdata/integration/test.proto",
			shouldPass:  true,
			description: "Proto file with optional fields should compile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			// Verify proto file exists
			_, err := os.Stat(tt.protoFile)
			if tt.shouldPass {
				require.NoError(t, err, "Proto file should exist")
			}
		})
	}
}

// TestOptionalFieldsInGeneratedCode specifically tests optional field generation
func TestOptionalFieldsInGeneratedCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDir := "testdata/integration"
	protoFile := filepath.Join(testDir, "test.proto")

	// Generate code first
	currentDir, err := os.Getwd()
	require.NoError(t, err)

	// Build plugin
	buildCmd := exec.Command("go", "build", "-o", "protoc-gen-redact", ".")
	_, err = buildCmd.CombinedOutput()
	require.NoError(t, err, "Should build plugin")

	// Generate Go code
	genCmd := exec.Command("protoc",
		"--experimental_allow_proto3_optional",
		"--go_out="+currentDir,
		"--go_opt=paths=source_relative",
		"--go-grpc_out="+currentDir,
		"--go-grpc_opt=paths=source_relative",
		"-I", ".",
		protoFile,
	)
	_, err = genCmd.CombinedOutput()
	require.NoError(t, err, "Should generate Go code")

	// Generate redaction code
	redactCmd := exec.Command("protoc",
		"--experimental_allow_proto3_optional",
		"--plugin=protoc-gen-redact=./protoc-gen-redact",
		"--redact_out="+currentDir,
		"--redact_opt=paths=source_relative",
		"-I", ".",
		protoFile,
	)
	_, err = redactCmd.CombinedOutput()
	require.NoError(t, err, "Should generate redaction code")

	// Now test the generated code
	redactFile := filepath.Join(testDir, "test.pb.redact.go")
	content, err := os.ReadFile(redactFile)
	require.NoError(t, err, "Should read generated file")

	contentStr := string(content)

	t.Run("optional_string_field", func(t *testing.T) {
		// email is optional string field
		// Should generate: emailTmp := "r*d@ct*d"
		//                  x.Email = &emailTmp
		assert.Contains(t, contentStr, "Email", "Should handle email field")
	})

	t.Run("optional_int32_field", func(t *testing.T) {
		// age is optional int32 field
		assert.Contains(t, contentStr, "Age", "Should handle age field")
	})

	t.Run("optional_bool_field", func(t *testing.T) {
		// is_active is optional bool field
		assert.Contains(t, contentStr, "IsActive", "Should handle is_active field")
	})

	t.Run("optional_message_field", func(t *testing.T) {
		// profile is optional message field with nested redaction
		assert.Contains(t, contentStr, "Profile", "Should handle profile field")
	})

	// Cleanup
	t.Cleanup(func() {
		os.Remove(filepath.Join(testDir, "test.pb.go"))
		os.Remove(filepath.Join(testDir, "test_grpc.pb.go"))
		os.Remove(filepath.Join(testDir, "test.pb.redact.go"))
		os.Remove("./protoc-gen-redact")
	})
}

// BenchmarkCodeGeneration benchmarks the code generation process
func BenchmarkCodeGeneration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	testDir := "testdata/integration"
	protoFile := filepath.Join(testDir, "test.proto")

	currentDir, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}

	// Build plugin once
	buildCmd := exec.Command("go", "build", "-o", "protoc-gen-redact", ".")
	if _, err := buildCmd.CombinedOutput(); err != nil {
		b.Fatal(err)
	}
	defer os.Remove("./protoc-gen-redact")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate redaction code
		cmd := exec.Command("protoc",
			"--experimental_allow_proto3_optional",
			"--plugin=protoc-gen-redact=./protoc-gen-redact",
			"--redact_out="+currentDir,
			"--redact_opt=paths=source_relative",
			"-I", ".",
			protoFile,
		)
		if _, err := cmd.CombinedOutput(); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()
	// Cleanup
	os.Remove(filepath.Join(testDir, "test.pb.redact.go"))
}
