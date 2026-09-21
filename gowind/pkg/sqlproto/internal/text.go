package internal

import (
	"context"
	"fmt"
	"log"

	"ariga.io/atlas/sql/schema"

	ddlparser "github.com/tx7do/go-utils/ddl_parser"

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

func (t *Text) toColumnType(tableName string, col ddlparser.ColumnDef, sqlContent string) (*schema.ColumnType, error) {
	parsedType, err := schemasource.ParseType(col.Type)
	if err != nil {
		return nil, err
	}

	// ddl_parser 不把 unsigned 带进 col.Type(如 INT UNSIGNED 只返回 "int"),
	// 需要回原文确认该列声明后再按 unsigned 重新解析一次。
	raw := col.Type
	if intType, ok := parsedType.(*schema.IntegerType); ok && !intType.Unsigned {
		if schemasource.ColumnIsUnsigned(tableName, col.Name, col.Type, sqlContent) {
			unsignedType, uErr := schemasource.ParseType(col.Type + " unsigned")
			if uErr == nil {
				if uIntType, ok := unsignedType.(*schema.IntegerType); ok && uIntType.Unsigned {
					parsedType = uIntType
					// Raw 是下游字符串映射(MySQLFieldType)的唯一输入,unsigned 必须留在里面。
					raw = col.Type + " unsigned"
				}
			}
		}
	}

	// PRIMARY KEY columns are always NOT NULL
	isNullable := col.Nullable
	if col.PrimaryKey {
		isNullable = false
	}

	return &schema.ColumnType{
		Type: parsedType,
		Raw:  raw,
		Null: isNullable,
	}, nil
}

func (t *Text) InspectSchema(sqlContent string, s *schema.Schema) (*schema.Schema, error) {
	if sqlContent == "" {
		return nil, fmt.Errorf("SQL 内容为空，无法解析")
	}
	if s == nil {
		s = &schema.Schema{}
	}

	// 解析 SQL 文本
	tables, err := ddlparser.ParseCreateTables(sqlContent)
	if err != nil {
		return nil, fmt.Errorf("解析失败: %v", err)
	}

	for _, tbl := range tables {
		table := &schema.Table{
			Name:   tbl.Name,
			Schema: s,
		}

		if tbl.Comment != "" {
			table.Attrs = append(table.Attrs, &schema.Comment{
				Text: tbl.Comment,
			})
		}
		if tbl.Collation != "" {
			table.Attrs = append(table.Attrs, &schema.Collation{
				V: tbl.Collation,
			})
		}
		if tbl.Charset != "" {
			table.Attrs = append(table.Attrs, &schema.Charset{
				V: tbl.Charset,
			})
		}

		for _, col := range tbl.Columns {
			colType, err := t.toColumnType(tbl.Name, col, sqlContent)
			if err != nil {
				log.Printf("sqlproto: 表 %s 的列 %s 解析失败: %v\n", tbl.Name, col.Name, err)
				continue
			}

			column := &schema.Column{
				Name: col.Name,
				Type: colType,
			}
			if col.Comment != "" {
				column.SetComment(col.Comment)
			}
			if col.Default != "" {
				column.SetDefault(&schema.NamedDefault{Expr: &schema.Literal{V: col.Default}})
			}

			if col.PrimaryKey {
				table.PrimaryKey = &schema.Index{
					Table: table,
					Name:  col.Name,
					Parts: []*schema.IndexPart{
						{
							C: column,
						},
					},
				}
			}

			table.Columns = append(table.Columns, column)
		}

		for _, idx := range tbl.Indexes {
			table.Indexes = append(table.Indexes, &schema.Index{
				Table: table,
				Name:  idx,
			})
		}

		s.Tables = append(s.Tables, table)
	}

	return s, nil
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

	tables := s.Tables

	// Filter by includedTables if specified
	if t.includedTables != nil && len(t.includedTables) > 0 {
		includedTableNames := make(map[string]bool)
		for _, tableName := range t.includedTables {
			includedTableNames[tableName] = true
		}
		tables = nil
		for _, table := range s.Tables {
			if includedTableNames[table.Name] {
				tables = append(tables, table)
			}
		}
	}

	// Filter out excludedTables
	if t.excludedTables != nil && len(t.excludedTables) > 0 {
		excludedTableNames := make(map[string]bool)
		for _, tableName := range t.excludedTables {
			excludedTableNames[tableName] = true
		}
		filteredTables := make([]*schema.Table, 0)
		for _, table := range tables {
			if !excludedTableNames[table.Name] {
				filteredTables = append(filteredTables, table)
			}
		}
		tables = filteredTables
	}

	return schemaTables(t.fieldType, tables)
}

func (t *Text) fieldType(sqlType string) (f string) {
	if f = MySQLFieldType(sqlType); f != "" {
		return
	}

	f = PostgresFieldType(sqlType)

	return
}
