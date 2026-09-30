package schemasource

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/dialect"
)

func TestNormalizeDSN(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "dump.sql")
	if err := os.WriteFile(existing, []byte("CREATE TABLE a (id INT);\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name    string
		dsn     string
		want    string
		wantErr string
	}{
		{name: "scheme passthrough", dsn: "mysql://user:pass@tcp(localhost:3306)/db", want: "mysql://user:pass@tcp(localhost:3306)/db"},
		{name: "trailing newline from shell substitution is trimmed", dsn: "text://CREATE TABLE a (id INT);\n", want: "text://CREATE TABLE a (id INT);"},
		{name: "existing file gains file scheme", dsn: existing, want: "file://" + existing},
		{name: "inline sql text gains text scheme", dsn: "CREATE TABLE a (id INT);", want: "text://CREATE TABLE a (id INT);"},
		{name: "indented sql text gains text scheme", dsn: "  ALTER TABLE a ADD COLUMN b INT;", want: "text://ALTER TABLE a ADD COLUMN b INT;"},
		{name: "dump comment header gains text scheme", dsn: "/*![40101 SET NAMES utf8mb4]*/;\nCREATE TABLE a (id INT);", want: "text:///*![40101 SET NAMES utf8mb4]*/;\nCREATE TABLE a (id INT);"},

		// 原生 DSN 是 go-sql-driver 的文档写法,也是 sqlkratos 那条管线喂进来的形状。
		// 它以前会被兜底成 text://,解析出 0 张表还返回 nil——用户只看到空产物。
		{name: "mysql native dsn gains mysql scheme", dsn: "root:pass@tcp(localhost:3306)/testdb?parseTime=true", want: "mysql://root:pass@tcp(localhost:3306)/testdb?parseTime=true"},
		{name: "mysql native dsn without password", dsn: "root@tcp(127.0.0.1:3307)/test", want: "mysql://root@tcp(127.0.0.1:3307)/test"},
		{name: "mysql unix socket dsn", dsn: "user:pw@unix(/tmp/mysql.sock)/db", want: "mysql://user:pw@unix(/tmp/mysql.sock)/db"},

		// 认不出来就不能再猜成 SQL 正文:报错并说明支持哪些写法。
		{name: "postgres key-value dsn is rejected", dsn: "host=localhost port=5432 user=postgres dbname=mydb", wantErr: "无法识别数据源"},
		{name: "bare word is rejected", dsn: "some_bare_source", wantErr: "无法识别数据源"},
		{name: "missing file is rejected", dsn: filepath.Join(t.TempDir(), "nope.sql"), wantErr: "无法识别数据源"},
		{name: "empty is rejected", dsn: "   \n", wantErr: "数据源为空"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDSN(tt.dsn)
			t.Logf("NormalizeDSN(%q) = %q, err = %v", tt.dsn, got, err)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("应当报错 %q,却返回了 %q", tt.wantErr, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("报错内容 = %q, want 包含 %q", err.Error(), tt.wantErr)
				}
				if got != "" {
					t.Fatalf("报错时不该给出结果: %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeDSN(%q) 报错: %v", tt.dsn, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeDSN(%q) = %q; want %q", tt.dsn, got, tt.want)
			}
		})
	}
}

// TestNormalizeDSN_MysqlNativeReachesMysqlProvider 原生 DSN 补完 scheme 后必须真的落到
// mysql provider,而不是停在字符串层面——scheme 与 provider 的拼接方式由 Mux.Open 决定。
func TestNormalizeDSN_MysqlNativeReachesMysqlProvider(t *testing.T) {
	var opened string
	mux := New()
	mux.RegisterProvider(func(dsn string) (*Driver, error) {
		opened = dsn
		return &Driver{Dialect: dialect.MySQL, SchemaName: "testdb"}, nil
	}, "mysql")

	normalized, err := NormalizeDSN("root@tcp(localhost:3306)/testdb")
	if err != nil {
		t.Fatal(err)
	}
	drv, err := mux.Open(normalized)
	if err != nil {
		t.Fatalf("Open(%q): %v", normalized, err)
	}
	defer drv.Close()

	t.Logf("provider 收到 %q", opened)
	if want := "root@tcp(localhost:3306)/testdb"; opened != want {
		t.Errorf("provider 收到 %q, want %q(scheme 之后不能多出或少掉字符)", opened, want)
	}
}

// TestNormalizeDSN_DDLWithDSNLookalikeComment 建表语句的注释里带着一个 @tcp(...)/db 时,
// 仍然得按正文处理:go-sql-driver 的解析器对这种串是宽松的,会把整段 DDL 当成"账号"。
func TestNormalizeDSN_DDLWithDSNLookalikeComment(t *testing.T) {
	ddl := "CREATE TABLE `x` (id INT) COMMENT='@tcp(fake)/db'"
	got, err := NormalizeDSN(ddl)
	t.Logf("NormalizeDSN = %q, err = %v", got, err)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "text://") {
		t.Errorf("带连接串样注释的 DDL 被当成了 DSN: %s", got)
	}
}

func TestLoadSQLFromFile(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "dump.sql")
	const content = "CREATE TABLE a (id INT);\n"
	if err := os.WriteFile(existing, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name string
		path string
		want string
	}{
		{name: "existing file yields its content", path: existing, want: content},
		// 生产路径给的是 NormalizeDSN 之后的值:裸文件路径会被补成 file://,
		// 所以判文件之前必须先剥 scheme,否则路径本身会被当成 SQL 文本。
		{name: "file scheme yields the file content", path: "file://" + existing, want: content},
		{name: "text scheme is stripped", path: "text://CREATE TABLE a (id INT);", want: "CREATE TABLE a (id INT);"},
		{name: "inline sql without scheme is returned as-is", path: "CREATE TABLE a (id INT);", want: "CREATE TABLE a (id INT);"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := LoadSQLFromFile(tt.path); got != tt.want {
				t.Fatalf("LoadSQLFromFile(%q) = %q; want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestParseType(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want any
	}{
		{raw: "int", want: &schema.IntegerType{}},
		{raw: "varchar(100)", want: &schema.StringType{}},
		{raw: "wtf$%$#", want: &schema.UnsupportedType{}},
	} {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseType(tt.raw)
			if err != nil {
				t.Fatalf("ParseType(%q): %v", tt.raw, err)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Fatalf("ParseType(%q) = %T; want %T", tt.raw, got, tt.want)
			}
		})
	}
}
