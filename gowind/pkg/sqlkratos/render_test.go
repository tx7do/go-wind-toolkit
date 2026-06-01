package sqlkratos

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

// ==============================
// NewGenerator
// ==============================

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	assert.NotNil(t, g)
	assert.NotNil(t, g.goGenerator)
	assert.NotNil(t, g.yamlGenerator)
	assert.NotNil(t, g.makefileGenerator)
	assert.NotNil(t, g.protoGenerator)
}

// ==============================
// WriteWireSetCode
// ==============================

func TestWriteWireSetCode(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "server")

	err := g.WriteWireSetCode(outputPath, "github.com/example/myproject", "user", "server", "Server", []string{"rest", "grpc"})
	assert.Nil(t, err)

	wireSetPath := filepath.Join(outputPath, "providers", "wire_set.go")
	assert.FileExists(t, wireSetPath)

	data, err := os.ReadFile(wireSetPath)
	assert.Nil(t, err)
	content := string(data)
	assert.Contains(t, content, "server.NewRestServer")
	assert.Contains(t, content, "server.NewGrpcServer")
}

func TestWriteWireSetCode_EmptyServices(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "data")

	err := g.WriteWireSetCode(outputPath, "github.com/example/myproject", "user", "data", "Client", []string{"Ent"})
	assert.Nil(t, err)
}

// ==============================
// WriteServerPackageCode
// ==============================

func TestWriteServerPackageCode_Grpc(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "server")
	err := os.MkdirAll(outputPath, os.ModePerm)
	assert.Nil(t, err)

	services := map[string]string{"user": "user", "role": "user"}
	err = g.WriteServerPackageCode(outputPath, "github.com/example/myproject", "grpc", "user", services)
	assert.Nil(t, err)

	grpcFile := filepath.Join(outputPath, "grpc_server.go")
	assert.FileExists(t, grpcFile)

	data, err := os.ReadFile(grpcFile)
	assert.Nil(t, err)
	content := string(data)
	assert.Contains(t, content, "userV1.RegisterUserServiceServer")
	assert.Contains(t, content, "userV1.RegisterRoleServiceServer")
}

func TestWriteServerPackageCode_Rest(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "server")
	err := os.MkdirAll(outputPath, os.ModePerm)
	assert.Nil(t, err)

	services := map[string]string{"user": "user", "role": "user"}
	err = g.WriteServerPackageCode(outputPath, "github.com/example/myproject", "rest", "user", services)
	assert.Nil(t, err)

	restFile := filepath.Join(outputPath, "rest_server.go")
	assert.FileExists(t, restFile)

	data, err := os.ReadFile(restFile)
	assert.Nil(t, err)
	content := string(data)
	assert.Contains(t, content, "userV1.RegisterUserServiceHTTPServer")
	assert.Contains(t, content, "userV1.RegisterRoleServiceHTTPServer")
}

func TestWriteServerPackageCode_UnsupportedType(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()

	err := g.WriteServerPackageCode(tmpDir, "github.com/example/myproject", "kafka", "user", nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unsupported service type")
}

// ==============================
// WriteDataPackageCode
// ==============================

func TestWriteDataPackageCode_Ent(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "data")

	fields := []generators.DataField{
		{Name: "id", Type: "int64", IsPrimaryKey: true},
		{Name: "name", Type: "string"},
	}

	err := g.WriteDataPackageCode(outputPath, "ent", "github.com/example/myproject", "user", "user", "user", "v1", fields)
	assert.Nil(t, err)
}

func TestWriteDataPackageCode_Gorm(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "data")

	fields := []generators.DataField{
		{Name: "id", Type: "int64", IsPrimaryKey: true},
		{Name: "name", Type: "string"},
	}

	err := g.WriteDataPackageCode(outputPath, "gorm", "github.com/example/myproject", "user", "user", "user", "v1", fields)
	assert.Nil(t, err)
}

func TestWriteDataPackageCode_UnsupportedOrm(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "data")

	err := g.WriteDataPackageCode(outputPath, "sqlx", "github.com/example/myproject", "user", "user", "user", "v1", nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unsupported orm")
}

func TestWriteDataPackageCode_EmptyFields(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "data")

	err := g.WriteDataPackageCode(outputPath, "ent", "github.com/example/myproject", "user", "user", "user", "v1", []generators.DataField{})
	assert.Nil(t, err)
}

func TestWriteDataPackageCode_EmptyTypeFields(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "data")

	fields := []generators.DataField{
		{Name: "id", Type: "", IsPrimaryKey: true},
	}

	err := g.WriteDataPackageCode(outputPath, "ent", "github.com/example/myproject", "user", "user", "user", "v1", fields)
	assert.Nil(t, err)
}

// ==============================
// WriteServicePackageCode
// ==============================

func TestWriteServicePackageCode(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "service")

	err := g.WriteServicePackageCode(
		outputPath,
		"github.com/example/myproject", "user",
		"role",
		"user", "admin", "v1",
		true, true,
	)
	assert.Nil(t, err)

	svcFile := filepath.Join(outputPath, "role_service.go")
	assert.FileExists(t, svcFile)
}

func TestWriteServicePackageCode_RestService(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "internal", "service")

	err := g.WriteServicePackageCode(
		outputPath,
		"github.com/example/myproject", "user",
		"role",
		"user", "admin", "v1",
		false, false,
	)
	assert.Nil(t, err)
}

// ==============================
// WriteMainCode
// ==============================

func TestWriteMainCode(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "cmd", "server")

	err := g.WriteMainCode(outputPath, "github.com/example/myproject", "user", []string{"rest", "grpc"})
	assert.Nil(t, err)

	mainFile := filepath.Join(outputPath, "main.go")
	assert.FileExists(t, mainFile)

	data, err := os.ReadFile(mainFile)
	assert.Nil(t, err)
	content := string(data)
	assert.Contains(t, content, "package main")
}

// ==============================
// WriteWireCode
// ==============================

func TestWriteWireCode(t *testing.T) {
	g := NewGenerator()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "app", "user", "service", "cmd", "server")

	err := g.WriteWireCode(outputPath, "github.com/example/myproject", "user")
	assert.Nil(t, err)

	wireFile := filepath.Join(outputPath, "wire.go")
	assert.FileExists(t, wireFile)
}
