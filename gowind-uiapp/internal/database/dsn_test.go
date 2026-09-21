package database

import (
	"net/url"
	"strings"
	"testing"
)

// TestBuildDSN_PostgresEscaping 用户名/口令/库名里的保留字符必须按 URL 规则编码,
// 并且要能原样读回来。
//
// 重点是空格:url.QueryEscape 走的是表单编码,空格编成 "+",而 userinfo 里的 "+"
// 是字面量字符——服务端拿到的口令就多了个加号。
func TestBuildDSN_PostgresEscaping(t *testing.T) {
	cfg := DBConfig{
		Type:     DbTypePostgreSQL,
		Host:     "db.internal",
		Port:     5432,
		Database: "my/db",
		Username: "app user",
		Password: "p@ss word+#",
		SSL:      true,
	}

	dsn, err := BuildDSN(cfg)
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	t.Logf("PostgreSQL DSN: %s", dsn)

	if strings.Contains(dsn, "+word") || strings.Contains(dsn, "app+") {
		t.Fatalf("空格被按表单编码成了 +,URL 里这会变义: %s", dsn)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("生成的 DSN 无法解析: %v", err)
	}
	if got, _ := url.PathUnescape(u.EscapedPath()); got != "/"+cfg.Database {
		t.Errorf("库名解回 %q, want %q", got, "/"+cfg.Database)
	}
	if u.Host != "db.internal:5432" {
		t.Errorf("host = %q(库名或口令里的字符改写掉了地址?)", u.Host)
	}
	if u.User.Username() != cfg.Username {
		t.Errorf("用户名 = %q, want %q", u.User.Username(), cfg.Username)
	}
	if got, _ := u.User.Password(); got != cfg.Password {
		t.Errorf("口令 = %q, want %q", got, cfg.Password)
	}
	if u.Query().Get("sslmode") != "require" {
		t.Errorf("sslmode = %q, want require", u.Query().Get("sslmode"))
	}
}

// TestBuildDSN_TwoBuildersAgree 连接路径与生成路径必须给出同一个 DSN。
// 连接路径原来自己拼了一遍且完全不转义,同一个配置在两条路径上会得到两个结果。
func TestBuildDSN_TwoBuildersAgree(t *testing.T) {
	cases := []struct {
		name string
		cfg  DBConfig
	}{
		{"postgresql", DBConfig{
			Type:     DbTypePostgreSQL,
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			Username: "postgres",
			Password: "p@ss word",
		}},
		{"oracle", DBConfig{
			Type:     DbTypeOracle,
			Host:     "localhost",
			Port:     1521,
			Database: "ORCL",
			Username: "system",
			Password: "p@ss word",
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			viaGenerate, err := BuildDSN(c.cfg)
			if err != nil {
				t.Fatalf("BuildDSN: %v", err)
			}
			viaConnect, err := buildDSN(c.cfg)
			if err != nil {
				t.Fatalf("buildDSN: %v", err)
			}
			t.Logf("DSN: %s", viaGenerate)
			if viaGenerate != viaConnect {
				t.Errorf("两个构造器分叉了:\n  生成路径 %s\n  连接路径 %s", viaGenerate, viaConnect)
			}
		})
	}
}

// TestBuildDSN_MySQL_NotURLEncoded MySQL 用的是驱动原生格式,不是 URL:
// 里面的 @ 与 : 都不该被转义(go-sql-driver 按最后一个 @ 划界、第一个 : 分列)。
func TestBuildDSN_MySQL_NotURLEncoded(t *testing.T) {
	cfg := DBConfig{
		Type:     DbTypeMySQL,
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Username: "root",
		Password: "p@ss:1",
	}

	dsn, err := BuildDSN(cfg)
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	if !strings.HasPrefix(dsn, "root:p@ss:1@tcp(localhost:3306)/testdb?") {
		t.Errorf("原生 DSN 被改写了: %s", dsn)
	}
	if strings.Contains(dsn, "%40") || strings.Contains(dsn, "%3A") {
		t.Errorf("原生格式里不该出现百分号转义: %s", dsn)
	}
}

// TestBuildDSN_PostgresParamsAreStable 查询串顺序要稳定:同一份配置每次生成的
// DSN 必须逐字节相同,否则日志与测试断言都会飘。
func TestBuildDSN_PostgresParamsAreStable(t *testing.T) {
	cfg := DBConfig{
		Type:     DbTypePostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "postgres",
		Password: "x",
	}
	first, err := BuildDSN(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		again, err := BuildDSN(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if again != first {
			t.Fatalf("第 %d 次生成不一致:\n  %s\n  %s", i+1, first, again)
		}
	}
}
