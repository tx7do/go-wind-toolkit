package internal

import (
	"context"
)

// FieldData 字段数据
type FieldData struct {
	Name    string // 字段名
	Type    string // 字段类型
	Null    bool   // 是否允许为 NULL
	Comment string // 字段注释
}

// TableData 表数据
type TableData struct {
	Name      string      // 表名
	Comment   string      // 表注释
	Charset   string      // 字符集
	Collation string      // 排序规则
	Fields    []FieldData // 字段数据
}

func (t TableData) WithComment() bool {
	return t.Comment != ""
}

type SchemaConverter interface {
	SchemaTables(context.Context) ([]*TableData, error)
}

type fieldTypeFunc func(sqlType string) string
