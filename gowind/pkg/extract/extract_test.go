package extract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==============================
// DetectOrmType
// ==============================

func TestDetectOrmType_Ent(t *testing.T) {
	tmpDir := t.TempDir()
	entSchemaPath := filepath.Join(tmpDir, "internal", "data", "ent", "schema")
	err := os.MkdirAll(entSchemaPath, os.ModePerm)
	assert.Nil(t, err)

	result := DetectOrmType(tmpDir)
	assert.Equal(t, "ent", result)
}

func TestDetectOrmType_Gorm(t *testing.T) {
	tmpDir := t.TempDir()
	gormSchemaPath := filepath.Join(tmpDir, "internal", "data", "gorm", "schema")
	err := os.MkdirAll(gormSchemaPath, os.ModePerm)
	assert.Nil(t, err)

	result := DetectOrmType(tmpDir)
	assert.Equal(t, "gorm", result)
}

func TestDetectOrmType_None(t *testing.T) {
	tmpDir := t.TempDir()

	result := DetectOrmType(tmpDir)
	assert.Equal(t, "", result)
}

func TestDetectOrmType_EntPriorityOverGorm(t *testing.T) {
	tmpDir := t.TempDir()
	entSchemaPath := filepath.Join(tmpDir, "internal", "data", "ent", "schema")
	err := os.MkdirAll(entSchemaPath, os.ModePerm)
	assert.Nil(t, err)
	gormSchemaPath := filepath.Join(tmpDir, "internal", "data", "gorm", "schema")
	err = os.MkdirAll(gormSchemaPath, os.ModePerm)
	assert.Nil(t, err)

	result := DetectOrmType(tmpDir)
	assert.Equal(t, "ent", result)
}

func TestDetectOrmType_NonExistentPath(t *testing.T) {
	result := DetectOrmType("/nonexistent/path/to/service")
	assert.Equal(t, "", result)
}

// ==============================
// injectBeforeMarker
// ==============================

func TestInjectBeforeMarker(t *testing.T) {
	content := "func foo() (*grpc.Server, error) {"
	result, err := injectBeforeMarker(content, ") (*grpc.Server", "\troleService *service.RoleService,")
	assert.Nil(t, err)
	assert.Contains(t, result, "roleService *service.RoleService,")
	assert.Contains(t, result, ") (*grpc.Server")
}

func TestInjectBeforeMarker_NotFound(t *testing.T) {
	content := "func foo() error {"
	_, err := injectBeforeMarker(content, ") (*grpc.Server", "\troleService,")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "marker")
}

func TestInjectBeforeMarker_EmptyContent(t *testing.T) {
	_, err := injectBeforeMarker("", "marker", "line")
	assert.NotNil(t, err)
}

// ==============================
// isFileExists / isDirExists
// ==============================

func TestIsFileExists_True(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(tmpFile, []byte("package test"), 0644)
	assert.Nil(t, err)

	assert.True(t, isFileExists(tmpFile))
}

func TestIsFileExists_False_NotExist(t *testing.T) {
	assert.False(t, isFileExists("/nonexistent/file.go"))
}

func TestIsFileExists_False_IsDir(t *testing.T) {
	tmpDir := t.TempDir()
	assert.False(t, isFileExists(tmpDir))
}

func TestIsDirExists_True(t *testing.T) {
	tmpDir := t.TempDir()
	assert.True(t, isDirExists(tmpDir))
}

func TestIsDirExists_False_NotExist(t *testing.T) {
	assert.False(t, isDirExists("/nonexistent/dir"))
}

func TestIsDirExists_False_IsFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(tmpFile, []byte("package test"), 0644)
	assert.Nil(t, err)

	assert.False(t, isDirExists(tmpFile))
}

// ==============================
// copyAndReplaceImport
// ==============================

func TestCopyAndReplaceImport(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "source.go")
	srcContent := `package data

import "github.com/example/myproject/app/admin/service/internal/data"
import pb "github.com/example/myproject/api/gen/go/admin/service/v1"
`
	err := os.WriteFile(srcFile, []byte(srcContent), 0644)
	assert.Nil(t, err)

	dstFile := filepath.Join(tmpDir, "target.go")

	e := NewExtractor(Options{
		ModulePath:    "github.com/example/myproject",
		SourceService: "admin",
		TargetService: "user",
	})

	err = e.copyAndReplaceImport(srcFile, dstFile)
	assert.Nil(t, err)

	data, err := os.ReadFile(dstFile)
	assert.Nil(t, err)

	result := string(data)
	assert.Contains(t, result, "app/user/service/internal/data")
	assert.NotContains(t, result, "app/admin/service")
	assert.Contains(t, result, "api/gen/go/user/service/v1")
	assert.NotContains(t, result, "api/gen/go/admin/service")
}

func TestCopyAndReplaceImport_SourceNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	e := NewExtractor(Options{
		ModulePath:    "github.com/example/myproject",
		SourceService: "admin",
		TargetService: "user",
	})

	err := e.copyAndReplaceImport(
		filepath.Join(tmpDir, "nonexistent.go"),
		filepath.Join(tmpDir, "target.go"),
	)
	assert.NotNil(t, err)
}

// ==============================
// removeProvider
// ==============================

func TestRemoveProvider(t *testing.T) {
	tmpDir := t.TempDir()
	providerFile := filepath.Join(tmpDir, "wire_set.go")
	content := `package data

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	data.NewRoleRepo,
	data.NewUserRepo,
)
`
	err := os.WriteFile(providerFile, []byte(content), 0644)
	assert.Nil(t, err)

	e := NewExtractor(Options{
		ModulePath:    "github.com/example/myproject",
		SourceService: "admin",
		TargetService: "user",
	})

	err = e.removeProvider(providerFile, "data.NewRoleRepo")
	assert.Nil(t, err)

	data, err := os.ReadFile(providerFile)
	assert.Nil(t, err)

	result := string(data)
	assert.NotContains(t, result, "NewRoleRepo")
	assert.Contains(t, result, "NewUserRepo")
}

func TestRemoveProvider_FileNotExist(t *testing.T) {
	e := NewExtractor(Options{
		ModulePath:    "github.com/example/myproject",
		SourceService: "admin",
		TargetService: "user",
	})

	err := e.removeProvider("/nonexistent/wire_set.go", "data.NewRoleRepo")
	assert.Nil(t, err) // should not error on non-existent file
}
