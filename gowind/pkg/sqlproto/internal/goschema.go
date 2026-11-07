package internal

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/schema"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

// GoSchema 转换由 Go 源码解析出的数据源(ent:// 与 gorm://):
// schemasource 的内存 Inspector 已把 ent schema / gorm model 目录还原为
// atlas 表,这里仅做 include/exclude 过滤与类型映射。
type GoSchema struct {
	*ConvertOptions
}

func NewGoSchema(i *ConvertOptions) (*GoSchema, error) {
	return &GoSchema{ConvertOptions: i}, nil
}

func (gs *GoSchema) SchemaTables(ctx context.Context) ([]*TableData, error) {
	s, err := gs.driver.InspectSchema(ctx, gs.driver.SchemaName, &schema.InspectOptions{
		Tables: gs.includedTables,
	})
	if err != nil {
		return nil, fmt.Errorf("sqlproto: inspect go source schema: %w", err)
	}

	// exclude 过滤统一走 schemasource.FilterTables(include 已由
	// memoryInspector 的 InspectOptions.Tables 过滤)。
	tables := schemasource.FilterTables(s.Tables, nil, gs.excludedTables)

	return schemaTables(gs.fieldType, tables)
}

// fieldType 依次尝试 MySQL/PostgreSQL 映射;源码解析产物的类型文本为 MySQL 风格。
func (gs *GoSchema) fieldType(sqlType string) string {
	if f := MySQLFieldType(sqlType); f != "" {
		return f
	}
	return PostgresFieldType(sqlType)
}
