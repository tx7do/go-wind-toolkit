package sqlkratos

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerate_MySQL_DDL 测试从 MySQL DDL 生成完整代码
func TestGenerate_MySQL_DDL(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	dsn := "mysql://root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	opts := GeneratorOptions{
		Driver:          "mysql",
		Source:          dsn,
		OrmType:         "ent",
		UseRepo:         true,
		GenerateProto:   true,
		GenerateORM:     true,
		GenerateData:    true,
		GenerateService: true,
		GenerateServer:  true,
		GenerateMain:    true,
		GenerateConfig:  true,
		GenerateMakefile: true,
		Servers:         []string{"grpc"},
		ProjectName:     "test-project",
		ServiceName:     "core",
		ModuleName:      "core",
		ModuleVersion:   "v1",
		OutputPath:      tmpDir,
		IncludedTables:  []string{"users", "orders"},
	}

	err := Generate(ctx, opts)
	// 由于没有真实数据库，预期会失败，但验证错误信息是否合理
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dial tcp") // MySQL 连接错误
}

// TestGenerate_DDLText 测试从 DDL 文本生成（不连接数据库）
func TestGenerate_DDLText(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	// DDL 文本作为 source（text:// 协议）
	ddl := `CREATE TABLE users (
		id bigint(20) NOT NULL AUTO_INCREMENT,
		name varchar(100) NOT NULL,
		email varchar(255) NOT NULL,
		created_at datetime NOT NULL,
		PRIMARY KEY (id)
	);

	CREATE TABLE orders (
		id bigint(20) NOT NULL AUTO_INCREMENT,
		user_id bigint(20) NOT NULL,
		total decimal(10,2) NOT NULL,
		status varchar(20) NOT NULL,
		created_at datetime NOT NULL,
		PRIMARY KEY (id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	opts := GeneratorOptions{
		Driver:           "mysql",
		Source:           ddl, // DDL 文本
		OrmType:          "ent",
		UseRepo:          true,
		GenerateProto:    true,
		GenerateORM:      true,
		GenerateData:     true,
		GenerateService:  true,
		GenerateServer:   true,
		GenerateMain:     true,
		GenerateConfig:   true,
		GenerateMakefile: true,
		Servers:          []string{"grpc"},
		ProjectName:      "test-project",
		ServiceName:      "core",
		ModuleName:       "core",
		ModuleVersion:    "v1",
		OutputPath:       tmpDir,
	}

	err := Generate(ctx, opts)
	// 预期会成功生成 schema 文件
	assert.NoError(t, err, "DDL text generation should succeed")

	// 验证生成的文件存在
	schemaPath := filepath.Join(tmpDir, "app", "core", "service", "internal", "data", "ent", "schema")
	entries, err := os.ReadDir(schemaPath)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "Schema files should be generated")

	// 验证 proto 文件存在
	protoPath := filepath.Join(tmpDir, "app", "core", "service", "api", "v1")
	entries, err = os.ReadDir(protoPath)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "Proto files should be generated")
}

// TestGenerate_InvalidDriver 测试无效驱动的处理
func TestGenerate_InvalidDriver(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	opts := GeneratorOptions{
		Driver:      "invalid_driver",
		Source:      "test",
		OrmType:     "ent",
		ProjectName: "test",
		ServiceName: "core",
		OutputPath:  tmpDir,
	}

	err := Generate(ctx, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported driver")
}

// TestGenerate_EmptyTables 测试空表列表的处理
func TestGenerate_EmptyTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	opts := GeneratorOptions{
		Driver:        "mysql",
		Source:        "mysql://localhost/test",
		OrmType:       "ent",
		ProjectName:   "test",
		ServiceName:   "core",
		OutputPath:    tmpDir,
		IncludedTables: []string{}, // 空表列表
	}

	err := Generate(ctx, opts)
	// 预期会成功（即使没有表也要生成基础结构）
	assert.NoError(t, err)
}

// TestGenerate_GenerateOnlyProto 测试仅生成 Proto
func TestGenerate_GenerateOnlyProto(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	opts := GeneratorOptions{
		Driver:          "mysql",
		Source:          "mysql://localhost/test",
		OrmType:         "ent",
		UseRepo:         false,
		GenerateProto:   true,
		GenerateORM:     false,
		GenerateData:    false,
		GenerateService: false,
		GenerateServer:  false,
		GenerateMain:    false,
		GenerateConfig:  false,
		GenerateMakefile: false,
		Servers:         []string{"grpc"},
		ProjectName:     "test",
		ServiceName:     "core",
		OutputPath:      tmpDir,
	}

	err := Generate(ctx, opts)
	// 预期会成功
	assert.NoError(t, err)
}
