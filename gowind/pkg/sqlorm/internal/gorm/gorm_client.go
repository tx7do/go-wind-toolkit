package gorm

import (
	"fmt"
	"strings"

	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"

	"gorm.io/gorm"
	"gorm.io/rawsql"
)

// NewGormClient 创建数据库客户端
func NewGormClient(drv, dsn string) (*gorm.DB, error) {
	var driver gorm.Dialector
	switch drv {
	default:
		fallthrough
	case "mysql":
		driver = mysql.Open(dsn)
		break
	case "postgres":
		driver = postgres.Open(dsn)
		break
	case "clickhouse":
		driver = clickhouse.Open(dsn)
		break
	case "sqlite":
		driver = sqlite.Open(dsn)
		break
	case "sqlserver":
		driver = sqlserver.Open(dsn)
		break
	}

	client, err := gorm.Open(driver, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("gormimport: failed to open database: %w", err)
	}

	return client, nil
}

// NewRawSqlClient 从 SQL 文本创建 gorm.DB（无需连接真实数据库）
// 适用于 gorm.io/gen 从建表语句生成模型
func NewRawSqlClient(sqlContent string) (*gorm.DB, error) {
	if strings.TrimSpace(sqlContent) == "" {
		return nil, fmt.Errorf("gormimport: SQL content is empty")
	}

	db, err := gorm.Open(rawsql.New(rawsql.Config{
		SQL: []string{sqlContent},
	}))
	if err != nil {
		return nil, fmt.Errorf("gormimport: failed to open rawsql: %w", err)
	}

	return db, nil
}
