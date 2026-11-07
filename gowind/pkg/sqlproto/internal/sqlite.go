package internal

import (
	"strings"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

// sqliteFieldType SQLite 是动态类型:STORAGE CLASS 只有 NULL/INTEGER/REAL/
// TEXT/BLOB,声明类型千姿百态。先借 MySQL/PostgreSQL 映射吃掉常见声明
// (含 TINYINT(1)->bool 特判),剩下的按 storage class 兜底。
func sqliteFieldType(sqlType string) (f string) {
	if f = MySQLFieldType(sqlType); f != "" {
		return
	}
	if f = PostgresFieldType(sqlType); f != "" {
		return
	}

	switch schemasource.ParseSQLType(sqlType).Base {
	case "TEXT", "CLOB", "BLOB":
		return "string"
	case "REAL", "DOUBLE", "FLOAT":
		return "float"
	case "":
		return ""
	default:
		// 未识别的声明类型按 string 兜底,保留告警上下文(schemaTables 会打)。
		if !strings.EqualFold(sqlType, "ANY") {
			return "string"
		}
		return ""
	}
}
