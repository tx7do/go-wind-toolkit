package internal

import (
	"context"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// MySQL到Protobuf的类型映射
var mysqlTypeMapping = map[string]string{
	// 整数类型（有符号）
	"TINYINT":   "int32",
	"SMALLINT":  "int32",
	"MEDIUMINT": "int32",
	"INT":       "int32",
	"INTEGER":   "int32",
	"BIGINT":    "int64",

	// 整数类型（无符号）
	"TINYINT UNSIGNED":   "uint32",
	"SMALLINT UNSIGNED":  "uint32",
	"MEDIUMINT UNSIGNED": "uint32",
	"INT UNSIGNED":       "uint32",
	"BIGINT UNSIGNED":    "uint64",

	// 浮点类型
	"FLOAT":   "float",
	"DOUBLE":  "double",
	"DECIMAL": "string", // Protobuf没有直接对应类型，通常用string表示

	// 字符串类型
	"CHAR":       "string",
	"VARCHAR":    "string",
	"TINYTEXT":   "string",
	"TEXT":       "string",
	"MEDIUMTEXT": "string",
	"LONGTEXT":   "string",

	// 二进制类型
	"BINARY":     "bytes",
	"VARBINARY":  "bytes",
	"TINYBLOB":   "bytes",
	"BLOB":       "bytes",
	"MEDIUMBLOB": "bytes",
	"LONGBLOB":   "bytes",

	// 日期和时间类型
	"DATE":      "string",
	"TIME":      "string",
	"DATETIME":  "google.protobuf.Timestamp",
	"TIMESTAMP": "google.protobuf.Timestamp",
	"YEAR":      "int32",

	// 布尔类型
	"BOOLEAN": "bool",
	"BOOL":    "bool",

	// JSON类型
	"JSON":  "string",
	"JSONB": "string", // MySQL 5.7.8+

	// 位类型
	"BIT": "int64",

	// 空间类型
	"GEOMETRY":           "bytes",
	"POINT":              "bytes",
	"LINESTRING":         "bytes",
	"POLYGON":            "bytes",
	"MULTIPOINT":         "bytes",
	"MULTILINESTRING":    "bytes",
	"MULTIPOLYGON":       "bytes",
	"GEOMETRYCOLLECTION": "bytes",

	// 其他类型
	"ENUM": "string", // 通常映射为字符串
	"SET":  "string", // 通常映射为字符串
}

// MySQL holds the schema import options and an Atlas inspector instance
type MySQL struct {
	*ConvertOptions
}

// NewMySQL - create aמ import structure for MySQL.
func NewMySQL(i *ConvertOptions) (*MySQL, error) {
	return &MySQL{
		ConvertOptions: i,
	}, nil
}

func (m *MySQL) SchemaTables(ctx context.Context) ([]*TableData, error) {
	inspectOptions := &schema.InspectOptions{
		Tables: m.includedTables,
	}
	s, err := m.driver.InspectSchema(ctx, m.driver.SchemaName, inspectOptions)
	if err != nil {
		return nil, err
	}

	tables := s.Tables
	if m.excludedTables != nil {
		tables = nil
		excludedTableNames := make(map[string]bool)
		for _, t := range m.excludedTables {
			excludedTableNames[t] = true
		}
		// filter out includedTables that are in excludedTables:
		for _, t := range s.Tables {
			if !excludedTableNames[t.Name] {
				tables = append(tables, t)
			}
		}
	}

	return schemaTables(MySQLFieldType, tables)
}

func MySQLFieldType(sqlType string) (f string) {
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
