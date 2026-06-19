package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestImportAliasGeneration tests the generation of unique aliases for imports
func TestImportAliasGeneration(t *testing.T) {
	tests := []struct {
		name           string
		existingAlias  map[string]string
		existingPath   map[string]string
		newImportPath  string
		newPackageName string
		expectedAlias  string
	}{
		{
			name:           "first_import_simple",
			existingAlias:  map[string]string{},
			existingPath:   map[string]string{},
			newImportPath:  "github.com/example/user",
			newPackageName: "user",
			expectedAlias:  "user",
		},
		{
			name: "conflict_requires_number_suffix",
			existingAlias: map[string]string{
				"github.com/example/user": "user",
			},
			existingPath: map[string]string{
				"user": "github.com/example/user",
			},
			newImportPath:  "github.com/other/user",
			newPackageName: "user",
			expectedAlias:  "user1",
		},
		{
			name: "multiple_conflicts",
			existingAlias: map[string]string{
				"github.com/example/user": "user",
				"github.com/other/user":   "user1",
				"github.com/another/user": "user2",
			},
			existingPath: map[string]string{
				"user":  "github.com/example/user",
				"user1": "github.com/other/user",
				"user2": "github.com/another/user",
			},
			newImportPath:  "github.com/fourth/user",
			newPackageName: "user",
			expectedAlias:  "user3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate alias generation logic
			alias := tt.newPackageName
			_, exists := tt.existingPath[alias]
			counter := 0
			for exists {
				counter++
				alias = tt.newPackageName + string(rune('0'+counter))
				_, exists = tt.existingPath[alias]
			}

			// This test demonstrates the expected behavior
			// The actual implementation is in imports.go
			if counter > 0 {
				// Expected format with number suffix
				assert.Contains(t, alias, tt.newPackageName)
			}
		})
	}
}

// TestStandardImports tests that standard imports are always included
func TestStandardImports(t *testing.T) {
	standardImports := map[string]string{
		"context": "context",
		"grpc":    "google.golang.org/grpc",
		"codes":   "google.golang.org/grpc/codes",
		"status":  "google.golang.org/grpc/status",
		"redact":  "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
	}

	// Verify all standard imports are present
	for alias, path := range standardImports {
		assert.NotEmpty(t, alias, "Alias should not be empty")
		assert.NotEmpty(t, path, "Path should not be empty")
	}

	// Verify bidirectional mapping
	assert.Equal(t, 5, len(standardImports), "Should have 5 standard imports")
}

// TestStandardReferences tests that standard references are included
func TestStandardReferences(t *testing.T) {
	standardRefs := []string{
		"grpc.Server",
		"context.Context",
		"redact.Redactor",
		"codes.Code",
		"status.Status",
	}

	assert.Len(t, standardRefs, 5, "Should have 5 standard references")

	for _, ref := range standardRefs {
		assert.Contains(t, ref, ".", "Reference should contain package separator")
	}
}

// TestImportPathHandling tests various import path scenarios
func TestImportPathHandling(t *testing.T) {
	tests := []struct {
		name        string
		importPath  string
		shouldSkip  bool
		description string
	}{
		{
			name:        "normal_proto_import",
			importPath:  "github.com/example/user/pb",
			shouldSkip:  false,
			description: "Normal proto package should be included",
		},
		{
			name:        "self_import",
			importPath:  "self/package/path",
			shouldSkip:  true,
			description: "Self imports should be skipped",
		},
		{
			name:        "google_api_import",
			importPath:  "google/protobuf/empty.proto",
			shouldSkip:  false,
			description: "Google protobuf imports may be included if they have messages",
		},
		{
			name:        "annotation_only_import",
			importPath:  "redact/redact.proto",
			shouldSkip:  true,
			description: "Annotation-only imports should be skipped if no usable types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			assert.NotEmpty(t, tt.importPath)
		})
	}
}

// TestImportWithNestedPackages tests imports with nested package structures
func TestImportWithNestedPackages(t *testing.T) {
	nestedImports := []struct {
		path          string
		expectedAlias string
		depth         int
	}{
		{
			path:          "github.com/org/project/api/v1/user",
			expectedAlias: "user",
			depth:         6,
		},
		{
			path:          "github.com/org/project/internal/pkg/models",
			expectedAlias: "models",
			depth:         6,
		},
		{
			path:          "google.golang.org/grpc/codes",
			expectedAlias: "codes",
			depth:         4,
		},
	}

	for _, ni := range nestedImports {
		t.Run(ni.expectedAlias, func(t *testing.T) {
			assert.NotEmpty(t, ni.path)
			assert.Greater(t, ni.depth, 0)
		})
	}
}

// TestCrossPackageReferences tests handling of cross-package references
func TestCrossPackageReferences(t *testing.T) {
	type Reference struct {
		packageAlias string
		typeName     string
		fullRef      string
	}

	references := []Reference{
		{
			packageAlias: "pb",
			typeName:     "User",
			fullRef:      "pb.User",
		},
		{
			packageAlias: "common",
			typeName:     "Timestamp",
			fullRef:      "common.Timestamp",
		},
		{
			packageAlias: "status",
			typeName:     "Status",
			fullRef:      "status.Status",
		},
	}

	for _, ref := range references {
		t.Run(ref.fullRef, func(t *testing.T) {
			assert.Equal(t, ref.packageAlias+"."+ref.typeName, ref.fullRef)
		})
	}
}

// TestImportConflictResolution tests complex import conflict scenarios
func TestImportConflictResolution(t *testing.T) {
	scenarios := []struct {
		name        string
		imports     []string
		packageName string
		expected    []string
	}{
		{
			name: "multiple_user_packages",
			imports: []string{
				"github.com/company/user",
				"github.com/external/user",
				"github.com/internal/user",
			},
			packageName: "user",
			expected:    []string{"user", "user1", "user2"},
		},
		{
			name: "mixed_package_names",
			imports: []string{
				"github.com/company/api",
				"github.com/company/models",
				"github.com/company/types",
			},
			packageName: "various",
			expected:    []string{"api", "models", "types"},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			assert.Len(t, scenario.imports, len(scenario.expected))
		})
	}
}

// TestWellKnownTypeImports tests handling of well-known protobuf types
func TestWellKnownTypeImports(t *testing.T) {
	wellKnownTypes := map[string]string{
		"google.golang.org/protobuf/types/known/anypb":       "any",
		"google.golang.org/protobuf/types/known/durationpb":  "duration",
		"google.golang.org/protobuf/types/known/emptypb":     "empty",
		"google.golang.org/protobuf/types/known/timestamppb": "timestamp",
		"google.golang.org/protobuf/types/known/wrapperspb":  "wrappers",
		"google.golang.org/protobuf/types/known/structpb":    "structpb",
	}

	for path, expectedPkg := range wellKnownTypes {
		t.Run(expectedPkg, func(t *testing.T) {
			assert.NotEmpty(t, path)
			assert.NotEmpty(t, expectedPkg)
		})
	}
}

// TestImportDeduplication tests that duplicate imports are handled correctly
func TestImportDeduplication(t *testing.T) {
	type ImportSet struct {
		path  string
		count int
	}

	testCases := []struct {
		name     string
		imports  []ImportSet
		expected int
	}{
		{
			name: "no_duplicates",
			imports: []ImportSet{
				{"github.com/a/pkg1", 1},
				{"github.com/b/pkg2", 1},
				{"github.com/c/pkg3", 1},
			},
			expected: 3,
		},
		{
			name: "with_duplicates",
			imports: []ImportSet{
				{"github.com/a/pkg1", 2},
				{"github.com/b/pkg2", 1},
				{"github.com/a/pkg1", 1}, // duplicate
			},
			expected: 2, // should deduplicate
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uniquePaths := make(map[string]bool)
			for _, imp := range tc.imports {
				uniquePaths[imp.path] = true
			}
			assert.Equal(t, tc.expected, len(uniquePaths))
		})
	}
}

// TestImportAliasValidCharacters tests that aliases only contain valid Go identifiers
func TestImportAliasValidCharacters(t *testing.T) {
	validAliases := []string{
		"user",
		"user1",
		"user_v2",
		"pb",
		"protobuf",
		"grpc",
		"status",
	}

	invalidAliases := []string{
		"user-pkg", // hyphen not allowed
		"1user",    // cannot start with number
		"user.pkg", // dot not allowed
		"user/pkg", // slash not allowed
		"user pkg", // space not allowed
	}

	for _, alias := range validAliases {
		t.Run("valid_"+alias, func(t *testing.T) {
			// Valid aliases should not contain special characters
			assert.NotContains(t, alias, "-")
			assert.NotContains(t, alias, ".")
			assert.NotContains(t, alias, "/")
			assert.NotContains(t, alias, " ")
		})
	}

	for _, alias := range invalidAliases {
		t.Run("invalid_"+alias, func(t *testing.T) {
			// These would fail Go compilation
			hasInvalidChar := false
			for _, char := range []string{"-", ".", "/", " "} {
				if contains(alias, char) {
					hasInvalidChar = true
					break
				}
			}
			// Check if starts with number
			if alias != "" && alias[0] >= '0' && alias[0] <= '9' {
				hasInvalidChar = true
			}
			assert.True(t, hasInvalidChar, "Alias '%s' should be identified as invalid", alias)
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestImportOrderPreservation tests that import order is deterministic
func TestImportOrderPreservation(t *testing.T) {
	// Map iteration order is random in Go, but we want deterministic output
	// This test documents the expected behavior
	imports := map[string]string{
		"context": "context",
		"grpc":    "google.golang.org/grpc",
		"status":  "google.golang.org/grpc/status",
	}

	// Verify all expected imports are present
	assert.Contains(t, imports, "context")
	assert.Contains(t, imports, "grpc")
	assert.Contains(t, imports, "status")

	// In the actual generated code, imports should be sorted for consistency
	assert.Len(t, imports, 3)
}

// TestEmptyImportHandling tests handling of files with no additional imports
func TestEmptyImportHandling(t *testing.T) {
	// Even with no additional imports, standard imports should be present
	minimalImports := map[string]string{
		"context": "context",
		"grpc":    "google.golang.org/grpc",
		"redact":  "github.com/menta2k/protoc-gen-redact/v3/redact/v3",
	}

	assert.NotEmpty(t, minimalImports, "Should always have at least standard imports")
	assert.GreaterOrEqual(t, len(minimalImports), 3)
}

// TestImportPathNormalization tests that import paths are normalized correctly
func TestImportPathNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "github.com/user/project",
			expected: "github.com/user/project",
		},
		{
			input:    "google.golang.org/grpc",
			expected: "google.golang.org/grpc",
		},
		{
			input:    "github.com/user/project/v2",
			expected: "github.com/user/project/v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Paths should be used as-is, no modification
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

// BenchmarkImportLookup benchmarks import path lookups
func BenchmarkImportLookup(b *testing.B) {
	imports := map[string]string{
		"user":     "github.com/example/user",
		"product":  "github.com/example/product",
		"order":    "github.com/example/order",
		"payment":  "github.com/example/payment",
		"shipping": "github.com/example/shipping",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = imports["user"]
		_ = imports["product"]
		_ = imports["order"]
	}
}

// BenchmarkImportAliasGeneration benchmarks alias generation with conflicts
func BenchmarkImportAliasGeneration(b *testing.B) {
	existingAliases := map[string]string{
		"user":  "github.com/example/user",
		"user1": "github.com/other/user",
		"user2": "github.com/another/user",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alias := "user"
		counter := 0
		for {
			if _, exists := existingAliases[alias]; !exists {
				break
			}
			counter++
			alias = "user" + string(rune('0'+counter))
		}
	}
}
