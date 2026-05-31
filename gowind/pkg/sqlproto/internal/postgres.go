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
	"SMALLINT":  "int32",
	"INT2":      "int32",
	"INTEGER":   "int32",
	"INT":       "int32",
	"INT4":      "int32",
	"BIGINT":    "int64",
	"INT8":      "int64",
	"SERIAL":    "int32",
	"SERIAL4":   "int32",
	"BIGSERIAL": "int64",
	"SERIAL8":   "int64",

	// 浮点类型
	"REAL":             "float",
	"FLOAT4":           "float",
	"DOUBLE PRECISION": "double",
	"FLOAT8":           "double",
	"NUMERIC":          "string", // Protobuf没有直接对应类型，通常用string表示

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
	"TIMESTAMP":   "string",
	"TIMESTAMPTZ": "string",
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
	if protoType, exists := mysqlTypeMapping[baseType]; exists {
		return protoType
	}

	return ""
}
