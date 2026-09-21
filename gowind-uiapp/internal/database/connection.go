package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	// 数据库驱动
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/sijms/go-ora/v2"
	_ "modernc.org/sqlite"
)

// DBConnection 数据库连接封装
type DBConnection struct {
	db     *sql.DB
	config DBConfig
}

// Connect 建立数据库连接（带超时控制）
func Connect(cfg DBConfig) (*DBConnection, error) {
	// 设置超时时间
	timeout := 10 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 构建 DSN
	dsn := cfg.DSN // 默认使用配置中的 DSN
	if !cfg.UseDSN {
		// 如果不使用自定义 DSN，根据配置构建
		var err error
		dsn, err = buildDSN(cfg)
		if err != nil {
			return nil, &DBError{
				Code:    "INVALID_CONFIG",
				Message: "数据库配置无效",
				Details: err.Error(),
			}
		}
	}

	if dsn == "" {
		return nil, &DBError{
			Code:    "INVALID_CONFIG",
			Message: "DSN 为空",
			Details: "请提供有效的 DSN 或配置字段",
		}
	}

	// 打开连接
	var db *sql.DB
	var err error
	switch cfg.Type {
	case DbTypeMySQL:
		db, err = sql.Open("mysql", dsn)
	case DbTypePostgreSQL:
		db, err = sql.Open("pgx", dsn)
	case DbTypeSQLite:
		db, err = sql.Open("sqlite", dsn)
	case DbTypeOracle:
		db, err = sql.Open("oracle", dsn)
	default:
		return nil, &DBError{
			Code:    "UNSUPPORTED_DB",
			Message: fmt.Sprintf("不支持的数据库类型: %s", cfg.Type),
			Details: "支持: mysql, postgresql, sqlite, oracle",
		}
	}

	if err != nil {
		return nil, wrapDBError(err, cfg.Type)
	}

	// 设置连接池参数
	maxOpenConns := 5
	if cfg.MaxOpenConns > 0 {
		maxOpenConns = cfg.MaxOpenConns
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns / 2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, wrapDBError(err, cfg.Type)
	}

	conn := &DBConnection{
		db:     db,
		config: cfg,
	}

	return conn, nil
}

// Ping 测试连接可用性
func (conn *DBConnection) Ping(ctx context.Context) error {
	return conn.db.PingContext(ctx)
}

// Close 关闭连接
func (conn *DBConnection) Close() {
	if conn.db != nil {
		_ = conn.db.Close()
	}
}

// Exec 执行 SQL（INSERT/UPDATE/DELETE）
func (conn *DBConnection) Exec(ctx context.Context, sql string, args ...any) (sql.Result, error) {
	return conn.db.ExecContext(ctx, sql, args...)
}

// Query 执行查询（SELECT）
func (conn *DBConnection) Query(ctx context.Context, sql string, args ...any) (*sql.Rows, error) {
	return conn.db.QueryContext(ctx, sql, args...)
}

// 构建 DSN（Data Source Name）
func buildDSN(cfg DBConfig) (string, error) {
	switch cfg.Type {
	case DbTypeMySQL:
		// username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
		sslMode := "false"
		if cfg.SSL {
			sslMode = "true"
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database, sslMode), nil

	case DbTypePostgreSQL:
		// 与生成路径共用同一个构造器:转义只有一份实现才不会两边分叉。
		return buildPostgresDSN(cfg)

	case DbTypeSQLite:
		if cfg.DBPath == "" {
			return "", fmt.Errorf("SQLite 需要指定数据库文件路径")
		}
		// modernc.org/sqlite 支持的 DSN 格式
		// 可以是文件路径或 :memory: 或其他选项
		// 参考: https://pkg.go.dev/modernc.org/sqlite
		dsn := cfg.DBPath
		// 如果是内存数据库，无需额外处理
		if dsn == ":memory:" {
			return dsn, nil
		}
		// 对于文件路径，可以添加查询参数
		return fmt.Sprintf("file:%s?cache=shared&mode=rwc&_fk=1", dsn), nil

	case DbTypeOracle:
		// oracle://user:pass@host:port/service_name
		// 同 PostgreSQL:转义只留一份实现,避免两条路径对含特殊字符的口令给出不同结果。
		return buildOracleDSN(cfg)

	default:
		return "", fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// 错误包装（统一错误处理）
func wrapDBError(err error, dbType DbType) *DBError {
	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "authentication") || strings.Contains(lower, "access denied"):
		return &DBError{
			Code:    "AUTH_FAILED",
			Message: "认证失败，请检查用户名或密码",
			Details: msg,
		}
	case strings.Contains(lower, "connect: connection refused") || strings.Contains(lower, "timeout"):
		return &DBError{
			Code:    "CONN_TIMEOUT",
			Message: "连接超时，请检查主机地址和端口",
			Details: msg,
		}
	case strings.Contains(lower, "unknown database") || strings.Contains(lower, "database .* does not exist"):
		return &DBError{
			Code:    "DB_NOT_FOUND",
			Message: "数据库不存在",
			Details: msg,
		}
	case strings.Contains(lower, "driver: bad connection"):
		return &DBError{
			Code:    "BAD_CONNECTION",
			Message: "数据库连接异常",
			Details: msg,
		}
	default:
		return &DBError{
			Code:    "DB_ERROR",
			Message: fmt.Sprintf("%s 连接错误", dbType),
			Details: msg,
		}
	}
}
