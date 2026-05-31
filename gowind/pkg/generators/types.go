package generators

import (
	"context"

	"github.com/tx7do/go-utils/stringcase"
)

// ProtoField 字段数据
type ProtoField struct {
	Name    string // 字段名
	Type    string // 字段类型
	Null    bool   // 是否允许为 NULL
	Comment string // 字段注释
	Number  int    // 字段编号
}

// DataField 数据库字段定义
type DataField struct {
	Name    string // 字段名
	Type    string // 字段类型
	Comment string // 字段注释
}

type DataFieldArray []DataField

func (f DataField) CamelName() string {
	return stringcase.LowerCamelCase(f.Name)
}

func (f DataField) PascalName() string {
	return stringcase.UpperCamelCase(f.Name)
}

func (f DataField) SnakeName() string {
	return stringcase.SnakeCase(f.Name)
}

func (f DataField) EntPascalName() string {
	return SnakeToPascalPlus(f.Name)
}

func (f DataField) EntSetNillableFunc() string {
	return MakeEntSetNillableFunc(f.Name)
}

// TableData 表数据
type TableData struct {
	Name      string       // 表名
	Comment   string       // 表注释
	Charset   string       // 字符集
	Collation string       // 排序规则
	Fields    []ProtoField // 字段数据
}

func (t TableData) WithComment() bool {
	return t.Comment != ""
}

type SchemaConverter interface {
	SchemaTables(context.Context) ([]*TableData, error)
}

type fieldTypeFunc func(sqlType string) string
