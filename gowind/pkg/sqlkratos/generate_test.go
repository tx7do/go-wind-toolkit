package sqlkratos

import (
	"context"
	"database/sql"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
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
	if !localMySQLReachable() {
		t.Skipf("no MySQL reachable at %s", testMySQLAddr)
	}
}

// skipIfLocalMySQL 把守反向前提:断言"连不上库时必须报 dial 错"的用例,一旦本机真有
// MySQL 在监听,报错就变成认证/协议错而不含 "dial tcp",断言便无端转红。
func skipIfLocalMySQL(t *testing.T) {
	t.Helper()
	if localMySQLReachable() {
		t.Skipf("something is listening on %s, the connection-failure assertion no longer holds", testMySQLAddr)
	}
}

func localMySQLReachable() bool {
	conn, err := net.DialTimeout("tcp", testMySQLAddr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// protoFileContents 递归收集 root 下的 .proto 文件内容,键为文件名。包策略会在 v1
// 之前多开目录层级，所以按文件名取，不写死路径。
func protoFileContents(t *testing.T, root string) map[string]string {
	t.Helper()

	files := map[string]string{}
	require.NoError(t, filepath.WalkDir(root, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && strings.HasSuffix(p, ".proto") {
			body, readErr := os.ReadFile(p)
			if readErr != nil {
				return readErr
			}
			files[filepath.Base(p)] = string(body)
		}
		return nil
	}))
	return files
}

// protoBasenames 递归收集 root 下的 .proto 文件名并排序。包策略会在 v1 之前多开
// 目录层级，所以按文件名断言，不断言路径。
func protoBasenames(t *testing.T, root string) []string {
	t.Helper()

	var names []string
	for name := range protoFileContents(t, root) {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestGenerate_MySQL_DDL 测试 MySQL 连接串走的是 schemasource 拨号路径:断言的是
// "连不上库时报错",因此本机一旦真有 MySQL 在监听,报错会变成认证/协议错而不是
// dial tcp,该前提不再成立,直接跳过。
func TestGenerate_MySQL_DDL(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}
	skipIfLocalMySQL(t)

	ctx := context.Background()
	tmpDir := t.TempDir()

	dsn := "mysql://root:password@tcp(" + testMySQLAddr + ")/test?charset=utf8mb4&parseTime=True&loc=Local"
	opts := GeneratorOptions{
		Driver:           "mysql",
		Source:           dsn,
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
		IncludedTables:   []string{"users", "orders"},
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
	assert.NotEmpty(t, protoBasenames(t, filepath.Join(tmpDir, "api", "protos")),
		"Proto files should be generated")
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

// TestGenerate_TableFilter 固定住表过滤契约:IncludedTables 为空 = 全部表
// (见 sqlproto/internal/options.go 的 "all if empty"),非空 = 只导入列出的表。
//
// 替代原 TestGenerate_EmptyTables:那条用 mysql://localhost/test 当 source、
// 又不设任何 Generate* 开关,Generate 全程 no-op 返回 nil,assert.NoError 恒真。
func TestGenerate_TableFilter(t *testing.T) {
	const ddl = `CREATE TABLE users (
		id bigint(20) NOT NULL AUTO_INCREMENT,
		name varchar(100) NOT NULL,
		PRIMARY KEY (id)
	);

	CREATE TABLE orders (
		id bigint(20) NOT NULL AUTO_INCREMENT,
		total decimal(10,2) NOT NULL,
		PRIMARY KEY (id)
	);`

	for _, c := range []struct {
		name    string
		include []string
		want    []string
	}{
		{name: "empty include list yields every table", include: nil, want: []string{"order.proto", "user.proto"}},
		{name: "explicit include list narrows to that table", include: []string{"users"}, want: []string{"user.proto"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			// 只生成 proto:不连库、不跑 ent 代码生成,所以不需要 -short 豁免也不需要 go.mod。
			opts := GeneratorOptions{
				Source:         ddl,
				OrmType:        "ent",
				GenerateProto:  true,
				Servers:        []string{"grpc"},
				ProjectName:    "test",
				ServiceName:    "core",
				ModuleName:     "core",
				ModuleVersion:  "v1",
				OutputPath:     t.TempDir(),
				IncludedTables: c.include,
			}

			require.NoError(t, Generate(context.Background(), opts))
			assert.Equal(t, c.want, protoBasenames(t, filepath.Join(opts.OutputPath, "api", "protos")))
		})
	}
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
		Driver:           "mysql",
		Source:           "mysql://root:pass@tcp(" + testMySQLAddr + ")/test?parseTime=True",
		OrmType:          "ent",
		UseRepo:          false,
		GenerateProto:    true,
		GenerateORM:      false,
		GenerateData:     false,
		GenerateService:  false,
		GenerateServer:   false,
		GenerateMain:     false,
		GenerateConfig:   false,
		GenerateMakefile: false,
		Servers:          []string{"grpc"},
		ProjectName:      "test",
		ServiceName:      "core",
		OutputPath:       tmpDir,
	}

	err := Generate(ctx, opts)
	// 预期会成功
	assert.NoError(t, err)
}

// testMySQLDB 是本包 MySQL 集成用例专用的库。单独建库而不是往 `test` 里塞表,是为了
// 让 DROP 只落在测试私有的对象上——开发者的 `test` 库里可能真有同名业务表。
const testMySQLDB = "gowind_sqlkratos_it"

// testMySQLAdminDSN 是不带库名的管理连接(root:pass@tcp(host:port))。CI 的
// services:mysql 容器就是这个口令;本机密码不同时用 GOWIND_TEST_MYSQL_ROOT_DSN
// 指过去,否则用例会从"跳过"变成"红"。
func testMySQLAdminDSN() string {
	if v := os.Getenv("GOWIND_TEST_MYSQL_ROOT_DSN"); v != "" {
		return v
	}
	return "root:pass@tcp(" + testMySQLAddr + ")"
}

// seedMySQLUsersTable 重建 testMySQLDB 并在其中建一张 users 表,返回后即清理。
// 语句里的库名/表名都是本包常量,不接受外部输入。
func seedMySQLUsersTable(t *testing.T) {
	t.Helper()

	// 结尾的 "/" 是必需的:go-sql-driver 拒绝没有库名分隔符的 DSN
	// ("invalid DSN: missing the slash separating the database name"),而这里要在
	// 建库之前就能连上,所以不指定任何库。
	db, err := sql.Open("mysql", testMySQLAdminDSN()+"/")
	require.NoError(t, err, "open admin connection")
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Cleanup(func() {
		// 自带超时:库若在跑中途失去响应,清理不该把测试吊死。
		dropCtx, cancelDrop := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelDrop()
		if _, derr := db.ExecContext(dropCtx, "DROP DATABASE IF EXISTS "+testMySQLDB); derr != nil {
			t.Logf("清理测试库 %s 失败: %v", testMySQLDB, derr)
		}
	})

	for _, stmt := range []string{
		"DROP DATABASE IF EXISTS " + testMySQLDB,
		"CREATE DATABASE " + testMySQLDB,
		"CREATE TABLE " + testMySQLDB + ".users (" +
			"id bigint NOT NULL AUTO_INCREMENT, " +
			"name varchar(100) NOT NULL, " +
			"PRIMARY KEY (id))",
	} {
		_, err = db.ExecContext(ctx, stmt)
		require.NoError(t, err, "seed statement failed: %s", stmt)
	}
}

// TestGenerate_MySQLSeededTable 让 MySQL provider 这条路真正被断言:
// TestGenerate_GenerateOnlyProto 连的是空的 `test` 库,Generate 在
// generator.go 的 len(tables)==0 处直接返回 nil,assert.NoError 恒真。这里自建
// 专用库并建一张 users 表,断言 information_schema 探测出来的表确实变成了
// user.proto 且列映射成 int64 id / string name。
func TestGenerate_MySQLSeededTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	requireLocalMySQL(t)
	seedMySQLUsersTable(t)

	opts := GeneratorOptions{
		Driver:        "mysql",
		Source:        "mysql://" + testMySQLAdminDSN() + "/" + testMySQLDB + "?parseTime=True",
		OrmType:       "ent",
		GenerateProto: true,
		Servers:       []string{"grpc"},
		ProjectName:   "test",
		ServiceName:   "core",
		ModuleName:    "core",
		ModuleVersion: "v1",
		OutputPath:    t.TempDir(),
	}

	require.NoError(t, Generate(context.Background(), opts))

	files := protoFileContents(t, filepath.Join(opts.OutputPath, "api", "protos"))
	require.Equal(t, []string{"user.proto"}, protoBasenames(t, filepath.Join(opts.OutputPath, "api", "protos")),
		"MySQL 探测到的表应生成 user.proto")

	content := files["user.proto"]
	// 断言前先打印,红的时候一次就能看清真实内容,不用再来一轮 CI。
	t.Logf("生成的 user.proto:\n%s", content)
	assert.Contains(t, content, "optional int64 id = 1")
	assert.Contains(t, content, "optional string name = 2")
}
