package internal

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/schema"
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

	tables := s.Tables
	if len(gs.excludedTables) > 0 {
		excluded := make(map[string]bool, len(gs.excludedTables))
		for _, t := range gs.excludedTables {
			excluded[t] = true
		}
		filtered := make([]*schema.Table, 0, len(tables))
		for _, t := range tables {
			if !excluded[t.Name] {
				filtered = append(filtered, t)
			}
		}
		tables = filtered
	}

	return schemaTables(gs.fieldType, tables)
}

// fieldType 依次尝试 MySQL/PostgreSQL 映射;源码解析产物的类型文本为 MySQL 风格。
func (gs *GoSchema) fieldType(sqlType string) string {
	if f := MySQLFieldType(sqlType); f != "" {
		return f
	}
	return PostgresFieldType(sqlType)
}
