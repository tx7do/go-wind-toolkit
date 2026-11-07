package internal

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/schema"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

type Text struct {
	*ConvertOptions
}

func NewText(i *ConvertOptions) (*Text, error) {
	return &Text{
		ConvertOptions: i,
	}, nil
}

// InspectSchema 把 SQL 文本解析为 atlas schema:表装配与列语义已收敛到
// schemasource.ParseDDLText(与 entimport 共用同一实现),这里只保留
// 列类型 → proto 字符串映射的 sink。include/exclude 过滤由 SchemaTables
// 统一走 schemasource.FilterTables。
func (t *Text) InspectSchema(sqlContent string, s *schema.Schema) (*schema.Schema, error) {
	return schemasource.ParseDDLText(sqlContent, nil, s)
}

func (t *Text) SchemaTables(_ context.Context) ([]*TableData, error) {
	// 加载 SQL 文本
	sqlText := schemasource.LoadSQLFromFile(t.schemaPath)
	if sqlText == "" {
		return nil, fmt.Errorf("无法加载 SQL 文件: %v", t.schemaPath)
	}

	var s schema.Schema
	_, err := t.InspectSchema(sqlText, &s)
	if err != nil {
		return nil, err
	}

	// If no tables were parsed successfully, return an error
	if len(s.Tables) == 0 {
		return nil, fmt.Errorf("无效的 SQL: 无法解析任何有效的表")
	}

	// include/exclude 过滤统一走 schemasource.FilterTables。
	tables := schemasource.FilterTables(s.Tables, t.includedTables, t.excludedTables)

	return schemaTables(t.fieldType, tables)
}

func (t *Text) fieldType(sqlType string) (f string) {
	if f = MySQLFieldType(sqlType); f != "" {
		return
	}

	f = PostgresFieldType(sqlType)

	return
}
