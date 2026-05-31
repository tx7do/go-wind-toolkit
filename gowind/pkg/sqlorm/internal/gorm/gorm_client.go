package gorm

import (
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"

	"gorm.io/gorm"
)

// NewGormClient 创建数据库客户端
func NewGormClient(drv, dsn string) *gorm.DB {
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
		return nil
	}

	return client
}
