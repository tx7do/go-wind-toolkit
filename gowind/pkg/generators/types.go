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
	Name         string // 字段名
	Type         string // 字段类型
	Comment      string // 字段注释
	Null         bool   // 是否允许为 NULL
	IsPrimaryKey bool   // 是否为主键
}

type DataFieldArray []DataField

// HasTimestampField 检查字段数组中是否包含 Timestamp 类型的字段
func (f DataFieldArray) HasTimestampField() bool {
	for _, field := range f {
		if field.IsTimestampType() {
			return true
		}
	}
	return false
}

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

const (
	// ProtoTypeTimestamp 表示 google.protobuf.Timestamp 类型
	ProtoTypeTimestamp = "google.protobuf.Timestamp"
)

// IsTimestampType 判断字段是否为 Timestamp 类型
func (f DataField) IsTimestampType() bool {
	return f.Type == ProtoTypeTimestamp
}

func (f DataField) EntSetNillableFunc() string {
	if f.IsTimestampType() {
		return MakeEntSetNillableFuncWithTransfer(f.Name, "timeutil.TimestamppbToTime")
	}
	return MakeEntSetNillableFunc(f.Name)
}

// EntCreateSetFunc 根据字段是否可为 NULL 以及类型选择合适的 setter：
// NOT NULL + Timestamp → SetXxx(timeutil.TimestamppbToTime(req.Data.GetXxx()))
// NOT NULL             → SetXxx(req.Data.GetXxx())
// NULL     + Timestamp → SetNillableXxx(timeutil.TimestamppbToTime(req.Data.Xxx))
// NULL                 → SetNillableXxx(req.Data.Xxx)
func (f DataField) EntCreateSetFunc() string {
	if f.Null {
		return f.EntSetNillableFunc()
	}
	if f.IsTimestampType() {
		return MakeEntSetFuncWithTransfer(f.Name, "timeutil.TimestamppbToTime")
	}
	return MakeEntSetFunc(f.Name)
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
