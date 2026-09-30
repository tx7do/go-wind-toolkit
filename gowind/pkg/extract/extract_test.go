package extract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

// ==============================
// DetectOrmType
// ==============================

func TestDetectOrmType_Ent(t *testing.T) {
	tmpDir := t.TempDir()
	entSchemaPath := filepath.Join(tmpDir, "internal", "data", "ent", "schema")
	err := os.MkdirAll(entSchemaPath, 0o755)
	assert.Nil(t, err)

	result := DetectOrmType(tmpDir)
	assert.Equal(t, "ent", result)
}

func TestDetectOrmType_Gorm(t *testing.T) {
	tmpDir := t.TempDir()
	gormSchemaPath := filepath.Join(tmpDir, "internal", "data", "gorm", "schema")
	err := os.MkdirAll(gormSchemaPath, 0o755)
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
	err := os.MkdirAll(entSchemaPath, 0o755)
	assert.Nil(t, err)
	gormSchemaPath := filepath.Join(tmpDir, "internal", "data", "gorm", "schema")
	err = os.MkdirAll(gormSchemaPath, 0o755)
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

	assert.True(t, pkg.IsFileExists(tmpFile))
}

func TestIsFileExists_False_NotExist(t *testing.T) {
	assert.False(t, pkg.IsFileExists("/nonexistent/file.go"))
}

func TestIsFileExists_False_IsDir(t *testing.T) {
	tmpDir := t.TempDir()
	assert.False(t, pkg.IsFileExists(tmpDir))
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

// ==============================
// importsInternalData / targetWiringContextFor
// ==============================

func TestImportsInternalData(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		wants bool
	}{
		{"真实 import 路径", "package service\n\nimport (\n\t\"github.com/example/myproject/app/admin/service/internal/data\"\n)\n", true},
		{"data 子包", "import \"mod/app/admin/service/internal/data/ent\"\n", true},
		{"BFF 型:只引服务客户端", "import (\n\tuserV1 \"github.com/example/myproject/app/user/service/api/v1\"\n)\n", false},
		{"注释里提到 internal/data 而非 import", "// 本文件不使用 internal/data\nvar x = 1\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wants, importsInternalData(tt.src))
		})
	}
}

func newExtractorAt(t *testing.T, root string) *Extractor {
	t.Helper()
	return NewExtractor(Options{
		RootPath:      root,
		ModulePath:    "github.com/example/myproject",
		ProjectName:   "myproject",
		SourceService: "admin",
		TargetService: "user",
		OrmType:       "ent",
	})
}

// writeTargetServiceFile 在目标服务的 internal/service 下落地一个模型服务文件。
func writeTargetServiceFile(t *testing.T, root, model, content string) {
	t.Helper()
	dir := filepath.Join(root, "app", "user", "service", "internal", "service")
	assert.Nil(t, os.MkdirAll(dir, 0o755))
	assert.Nil(t, os.WriteFile(filepath.Join(dir, model+"_service.go"), []byte(content), 0o644))
}

func TestTargetWiringContextFor_RepoStyleServiceUsesRepo(t *testing.T) {
	root := t.TempDir()
	writeTargetServiceFile(t, root, "role", `package service

import (
	"github.com/example/myproject/app/user/service/internal/data"
)

var _ = data.RoleRepo(nil)
`)

	assert.False(t, newExtractorAt(t, root).targetWiringContextFor("Role").useClient)
}

func TestTargetWiringContextFor_BffStyleServiceUsesClient(t *testing.T) {
	root := t.TempDir()
	writeTargetServiceFile(t, root, "role", `package service

import (
	roleV1 "github.com/example/myproject/app/role/service/api/v1"
)

var _ = roleV1.RoleService{}
`)

	assert.True(t, newExtractorAt(t, root).targetWiringContextFor("Role").useClient)
}

func TestTargetWiringContextFor_MissingServiceFileDefaultsToClient(t *testing.T) {
	assert.True(t, newExtractorAt(t, t.TempDir()).targetWiringContextFor("Role").useClient)
}
