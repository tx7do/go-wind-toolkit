package internal

import (
	"context"
	"fmt"
	"log"
	"os"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"

	ddlparser "github.com/tx7do/go-utils/ddl_parser"
)

type Text struct {
	*ConvertOptions
}

func NewText(i *ConvertOptions) (*Text, error) {
	return &Text{
		ConvertOptions: i,
	}, nil
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (t *Text) loadSQLFromFile() string {
	// 检查 schemaPath 是否为文件
	if !isFile(t.schemaPath) {
		// 如果不是文件，直接返回 schemaPath，认为它是一个 SQL 文本内容。
		return t.schemaPath
	}

	content, err := os.ReadFile(t.schemaPath)
	if err != nil {
		return ""
	}

	return string(content)
}

func (t *Text) ParseType(raw string) (schema.Type, error) {
	mysqlType, err := mysql.ParseType(raw)
	if err == nil {
		return mysqlType, nil
	}

	postgresType, err := postgres.ParseType(raw)
	if err == nil {
		return postgresType, nil
	}

	sqliteType, err := sqlite.ParseType(raw)
	if err == nil {
		return sqliteType, nil
	}

	return &schema.UnsupportedType{T: raw}, nil
}

func (t *Text) toColumnType(col ddlparser.ColumnDef) (*schema.ColumnType, error) {
	parsedType, err := t.ParseType(col.Type)
	if err != nil {
		return nil, err
	}

	// PRIMARY KEY columns are always NOT NULL
	isNullable := col.Nullable
	if col.PrimaryKey {
		isNullable = false
	}

	return &schema.ColumnType{
		Type: parsedType,
		Raw:  col.Type,
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
			log.Printf("列名: %v, 类型: %v\n", col.Name, col.Type)

			colType, err := t.toColumnType(col)
			if err != nil {
				log.Printf("解析失败: %v\n", err)
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
	sqlText := t.loadSQLFromFile()
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
