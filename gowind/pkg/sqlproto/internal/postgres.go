package internal

import (
	"context"
	"strings"

	"ariga.io/atlas/sql/schema"

	_ "github.com/lib/pq"
)

// PostgreSQL到Protobuf的类型映射
var postgresqlTypeMapping = map[string]string{
	// 整数类型
	"SMALLINT":    "int32",
	"INT2":        "int32",
	"SMALLSERIAL": "int32", // 2字节自增
	"SERIAL2":     "int32",
	"INTEGER":     "int32",
	"INT":         "int32",
	"INT4":        "int32",
	"SERIAL":      "int32",
	"SERIAL4":     "int32",
	"BIGINT":      "int64",
	"INT8":        "int64",
	"BIGSERIAL":   "int64",
	"SERIAL8":     "int64",

	// 浮点类型
	"REAL":             "float",
	"FLOAT4":           "float",
	"FLOAT":            "float",
	"DOUBLE PRECISION": "double",
	"FLOAT8":           "double",
	"NUMERIC":          "string", // Protobuf没有直接对应类型，通常用string表示
	"DECIMAL":          "string",

	// 字符串类型
	"CHAR":              "string",
	"CHARACTER":         "string",
	"VARCHAR":           "string",
	"CHARACTER VARYING": "string",
	"TEXT":              "string",
	"BPCHAR":            "string",

	// 二进制类型
	"BYTEA": "bytes",

	// 日期和时间类型
	"DATE":        "string",
	"TIME":        "string",
	"TIMETZ":      "string",
	"TIMESTAMP":   "google.protobuf.Timestamp",
	"TIMESTAMPTZ": "google.protobuf.Timestamp",
	"INTERVAL":    "string",

	// 布尔类型
	"BOOLEAN": "bool",
	"BOOL":    "bool",

	// 网络地址类型
	"CIDR":    "string",
	"INET":    "string",
	"MACADDR": "string",

	// JSON类型
	"JSON":  "string",
	"JSONB": "string",

	// UUID类型
	"UUID": "string",

	// 位串类型
	"BIT":         "bytes",
	"BIT VARYING": "bytes",
	"VARBIT":      "bytes",

	// XML类型
	"XML": "string",

	// 货币类型
	"MONEY": "string",

	// 几何类型
	"POINT":   "string",
	"LINE":    "string",
	"LSEG":    "string",
	"BOX":     "string",
	"PATH":    "string",
	"POLYGON": "string",
	"CIRCLE":  "string",

	// 全文搜索类型
	"TSVECTOR": "string",
	"TSQUERY":  "string",

	// 数组类型（PostgreSQL特有，通常映射为JSON字符串）
	"ARRAY": "string",
}

// Postgres implements SchemaConverter for PostgreSQL databases.
type Postgres struct {
	*ConvertOptions
}

// NewPostgreSQL - returns a new *Postgres.
func NewPostgreSQL(i *ConvertOptions) (SchemaConverter, error) {
	return &Postgres{
		ConvertOptions: i,
	}, nil
}

func (p *Postgres) SchemaTables(ctx context.Context) ([]*TableData, error) {
	inspectOptions := &schema.InspectOptions{
		Tables: p.includedTables,
	}
	s, err := p.driver.InspectSchema(ctx, p.driver.SchemaName, inspectOptions)
	if err != nil {
		return nil, err
	}
	tables := s.Tables
	if p.excludedTables != nil {
		tables = nil
		excludedTableNames := make(map[string]bool)
		for _, t := range p.excludedTables {
			excludedTableNames[t] = true
		}
		// filter out includedTables that are in excludedTables:
		for _, t := range s.Tables {
			if !excludedTableNames[t.Name] {
				tables = append(tables, t)
			}
		}
	}
	return schemaTables(PostgresFieldType, tables)
}

func PostgresFieldType(sqlType string) (f string) {
	sqlType = strings.ToUpper(sqlType)

	// 去除类型声明中的括号部分，例如 "VARCHAR(255)" -> "VARCHAR"
	baseType := strings.SplitN(sqlType, "(", 2)[0]
	baseType = strings.TrimSpace(strings.ToUpper(baseType))

	// 查找映射
	if protoType, exists := postgresqlTypeMapping[baseType]; exists {
		return protoType
	}

	// 处理多词类型名，如 "TIMESTAMP WITHOUT TIME ZONE" -> "TIMESTAMP"
	// "TIMESTAMP WITH TIME ZONE" -> "TIMESTAMPTZ"
	// "CHARACTER VARYING" 已经在映射表中
	firstWord := strings.SplitN(baseType, " ", 2)[0]
	switch baseType {
	case "TIMESTAMP WITHOUT TIME ZONE":
		return postgresqlTypeMapping["TIMESTAMP"]
	case "TIMESTAMP WITH TIME ZONE":
		return postgresqlTypeMapping["TIMESTAMPTZ"]
	case "TIME WITH TIME ZONE":
		return postgresqlTypeMapping["TIMETZ"]
	case "BIT VARYING":
		return postgresqlTypeMapping["BIT VARYING"]
	case "DOUBLE PRECISION":
		return postgresqlTypeMapping["DOUBLE PRECISION"]
	}

	// 最后尝试只用第一个单词匹配
	if firstWord != baseType {
		if protoType, exists := postgresqlTypeMapping[firstWord]; exists {
			return protoType
		}
	}

	return ""
}
