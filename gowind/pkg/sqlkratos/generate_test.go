package sqlkratos

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMySQLAddr 是本包集成测试假定的本机 MySQL 地址。
// DSN 契约见 schemasource.Open:按 "://" 切分后剩余部分原样交给 sql.Open,
// 所以 mysql:// 后面必须是 go-sql-driver 的原生 DSN(user:pass@tcp(host:port)/db),
// 写成 mysql://localhost/test 会被驱动当成协议名,报
// "default addr for network 'localhost' unknown"。
const testMySQLAddr = "127.0.0.1:3306"

// requireLocalMySQL 探活本机 MySQL,不可达则跳过:这些用例断言的是"有库时能跑通",
// 没有库时应当跳过而不是失败。
func requireLocalMySQL(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", testMySQLAddr, 500*time.Millisecond)
	if err != nil {
		t.Skipf("no MySQL reachable at %s: %v", testMySQLAddr, err)
	}
	_ = conn.Close()
}

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

	// entimport 写 schema 前会 schemast.Load 该目录,要求它位于 Go module 内
	// (真实向导的输出目录本就是用户已 `go mod init` 过的项目),夹具需复现这一前置条件。
	initCmd := exec.Command("go", "mod", "init", "test-project")
	initCmd.Dir = tmpDir
	if initOut, initErr := initCmd.CombinedOutput(); initErr != nil {
		t.Fatalf("go mod init failed: %s", initOut)
	}

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

	// 验证 proto 文件存在。生产路径写在 <OutputPath>/api/protos/<proto包>/v1/*.proto
	// (见 generator.go 的 protoPath),包策略会多一层目录,所以递归找。
	var protoFiles []string
	require.NoError(t, filepath.WalkDir(filepath.Join(tmpDir, "api", "protos"),
		func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() && strings.HasSuffix(p, ".proto") {
				protoFiles = append(protoFiles, p)
			}
			return nil
		}))
	assert.NotEmpty(t, protoFiles, "Proto files should be generated")
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

// driver 只在 source 需要补 scheme 时才参与判定。GUI 的 ent:// 与 SQL 文本导入路径
// 会留空或传 sqlite/oracle,这些 source 自带类型信息,不能被 driver 校验误伤。
func TestEnsureDSNScheme_IgnoresDriverForSelfDescribingSources(t *testing.T) {
	cases := []struct {
		name   string
		source string
		driver string
		want   string
	}{
		{
			name:   "ent source with empty driver",
			source: "ent://app/core/service/internal/data/ent/schema",
			driver: "",
			want:   "ent://app/core/service/internal/data/ent/schema",
		},
		{
			name:   "sqlite driver with ddl text",
			source: "CREATE TABLE users (id bigint NOT NULL);",
			driver: "sqlite",
			want:   "CREATE TABLE users (id bigint NOT NULL);",
		},
		{
			name:   "bare source with empty driver",
			source: "some_bare_source",
			driver: "",
			want:   "some_bare_source",
		},
		{
			name:   "mysql bare dsn gains scheme",
			source: "root@tcp(localhost:3306)/test",
			driver: "mysql",
			want:   "mysql://root@tcp(localhost:3306)/test",
		},
		{
			name:   "postgresql key-value dsn converts",
			source: "host=localhost port=5432 user=postgres dbname=mydb",
			driver: "postgresql",
			want:   "postgres://postgres@localhost:5432/mydb",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ensureDSNScheme(c.source, c.driver)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// 需要补 scheme 却给了本管线没有 provider 的驱动时必须立刻报错:
// 此前会静默走到底并返回 nil,调用方以为生成成功。
func TestEnsureDSNScheme_RejectsUnsupportedDriver(t *testing.T) {
	for _, driver := range []string{"invalid_driver", "sqlite", "oracle"} {
		t.Run(driver, func(t *testing.T) {
			got, err := ensureDSNScheme("some_bare_source", driver)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported driver")
			assert.Contains(t, err.Error(), driver)
			assert.Empty(t, got)
		})
	}
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

// TestGenerate_GenerateOnlyProto 测试仅生成 Proto —— 需要真实 MySQL,无库时跳过。
func TestGenerate_GenerateOnlyProto(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	requireLocalMySQL(t)

	ctx := context.Background()
	tmpDir := t.TempDir()

	opts := GeneratorOptions{
		Driver:          "mysql",
		Source:          "mysql://root:pass@tcp(" + testMySQLAddr + ")/test?parseTime=True",
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
