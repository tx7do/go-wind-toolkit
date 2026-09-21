package database

import (
	"fmt"
	"net/url"
	"strings"
)

// BuildDSN 根据配置智能构建 DSN
func BuildDSN(cfg DBConfig) (string, error) {
	// 优先使用自定义 DSN（高级模式）
	if cfg.UseDSN && cfg.DSN != "" {
		return cfg.DSN, nil
	}

	// 验证必要字段
	if cfg.Type == "" {
		return "", fmt.Errorf("数据库类型不能为空")
	}
	if cfg.Type != DbTypeSQLite && cfg.Host == "" {
		return "", fmt.Errorf("主机地址不能为空")
	}

	// 根据数据库类型构建 DSN
	switch cfg.Type {
	case DbTypeMySQL:
		return buildMySQLDSN(cfg)
	case DbTypePostgreSQL:
		return buildPostgresDSN(cfg)
	case DbTypeSQLite:
		return buildSQLiteDSN(cfg)
	case DbTypeOracle:
		return buildOracleDSN(cfg)
	default:
		return "", fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}
}

// MySQL DSN: user:pass@tcp(host:port)/db?params
func buildMySQLDSN(cfg DBConfig) (string, error) {
	if cfg.Username == "" {
		return "", fmt.Errorf("MySQL 用户名不能为空")
	}
	// 密码可为空（如本地免密登录）
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 3306
	}

	// 构建参数
	params := []string{"charset=utf8mb4", "parseTime=true", "loc=Local"}
	if cfg.SSL {
		params = append(params, "tls=skip-verify") // 生产环境应使用真实证书
	} else {
		params = append(params, "tls=false")
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		cfg.Username,
		cfg.Password, // 密码仅在此处拼接
		host,
		port,
		cfg.Database,
		strings.Join(params, "&"),
	), nil
}

// PostgreSQL DSN: postgres://user:pass@host:port/db?sslmode=disable
//
// 用户名/口令/库名一律交给 url.URL 编码。手写 url.QueryEscape 是错的:它按
// application/x-www-form-urlencoded 把空格编成 "+",而 userinfo 与 path 里的 "+"
// 是字面量,含空格的口令会被原样送给服务端。
func buildPostgresDSN(cfg DBConfig) (string, error) {
	if cfg.Username == "" {
		return "", fmt.Errorf("PostgreSQL 用户名不能为空")
	}
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 5432
	}

	sslMode := "disable"
	if cfg.SSL {
		sslMode = "require"
	}
	return buildURLDSN("postgres", cfg, host, port, url.Values{
		"sslmode":  {sslMode},
		"timezone": {"Asia/Shanghai"},
	})
}

// Oracle DSN
func buildOracleDSN(cfg DBConfig) (string, error) {
	if cfg.Username == "" {
		return "", fmt.Errorf("oracle 用户名不能为空")
	}
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 1521
	}
	serviceName := cfg.Database
	if serviceName == "" {
		serviceName = "ORCL"
	}

	cfg.Database = serviceName
	return buildURLDSN("oracle", cfg, host, port, nil)
}

// buildURLDSN 用 net/url 而不是字符串拼接生成 `<scheme>://user:pass@host:port/db?params`:
// userinfo、path、query 各自的合法字符集不同,自己拼迟早漏掉一个。
func buildURLDSN(scheme string, cfg DBConfig, host string, port int, params url.Values) (string, error) {
	u := &url.URL{
		Scheme: scheme,
		User:   url.UserPassword(cfg.Username, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", host, port),
	}
	if cfg.Database != "" {
		u.Path = "/" + cfg.Database
	}
	if len(params) > 0 {
		// Values.Encode 按键排序,同一份配置每次得到同一个 DSN。
		u.RawQuery = params.Encode()
	}
	return u.String(), nil
}

// SQLite DSN
func buildSQLiteDSN(cfg DBConfig) (string, error) {
	if cfg.DBPath == "" {
		return "", fmt.Errorf("SQLite 数据库路径不能为空")
	}
	// 支持内存数据库
	if cfg.DBPath == ":memory:" {
		return "file::memory:?cache=shared", nil
	}
	return fmt.Sprintf("file:%s?cache=shared&mode=rwc&_fk=1", cfg.DBPath), nil
}
