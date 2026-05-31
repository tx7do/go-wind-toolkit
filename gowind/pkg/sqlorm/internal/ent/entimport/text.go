package entimport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	ddlparser "github.com/tx7do/go-utils/ddl_parser"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"

	"entgo.io/contrib/schemast"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

type Text struct {
	*ImportOptions
}

func NewText(i *ImportOptions) (*Text, error) {
	return &Text{
		ImportOptions: i,
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

	return &schema.ColumnType{
		Type: parsedType,
		Raw:  col.Type,
		Null: col.Nullable,
	}, nil
}

func (t *Text) InspectSchema(ctx context.Context, sqlContent string, opts *schema.InspectOptions, s *schema.Schema) (*schema.Schema, error) {
	if sqlContent == "" {
		return nil, fmt.Errorf("SQL 文本为空，无法解析")
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
			log.Printf("列名: %v, 类型: %v, NULL: %v\n", col.Name, col.Type, col.Nullable)

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

// SchemaMutations 实现 SchemaImporter 接口，用于解析 SQL 文本
func (t *Text) SchemaMutations(ctx context.Context) ([]schemast.Mutator, error) {
	// 加载 SQL 文本
	sqlText := t.loadSQLFromFile()
	if sqlText == "" {
		return nil, fmt.Errorf("无法加载 SQL 文件: %v", t.schemaPath)
	}

	inspectOptions := &schema.InspectOptions{
		Tables: t.tables,
	}

	var s schema.Schema

	_, err := t.InspectSchema(ctx, sqlText, inspectOptions, &s)
	if err != nil {
		return nil, err
	}

	tables := s.Tables
	if t.excludedTables != nil {
		tables = nil
		excludedTableNames := make(map[string]bool)
		for _, t := range t.excludedTables {
			excludedTableNames[t] = true
		}
		// filter out tables that are in excludedTables:
		for _, t := range s.Tables {
			if !excludedTableNames[t.Name] {
				tables = append(tables, t)
			}
		}
	}

	return schemaMutations(t.field, tables)
}

func (t *Text) field(column *schema.Column) (f ent.Field, err error) {
	name := column.Name
	switch typ := column.Type.Type.(type) {
	case *schema.BinaryType:
		f = field.Bytes(name)
	case *schema.BoolType:
		f = field.Bool(name)
	case *schema.DecimalType:
		f = field.Float(name)
	case *schema.EnumType:
		f = field.Enum(name).Values(typ.Values...)
	case *schema.FloatType:
		f = t.convertFloat(typ, name)
	case *schema.IntegerType:
		f = t.convertInteger(typ, name)
	case *schema.JSONType:
		f = field.JSON(name, json.RawMessage{})
	case *schema.StringType:
		f = field.String(name)
	case *schema.TimeType:
		f = field.Time(name)

	case *postgres.SerialType:
		f = t.convertSerial(typ, name)
	case *postgres.UUIDType:
		f = field.UUID(name, uuid.New())

	default:
		return nil, fmt.Errorf("entimport: unsupported type %q for column %v", typ, column.Name)
	}
	applyColumnAttributes(f, column)
	return f, err
}

func (t *Text) convertFloat(typ *schema.FloatType, name string) (f ent.Field) {
	// Precision from 0 to 23 results in a 4-byte single-precision FLOAT column.
	// Precision from 24 to 53 results in an 8-byte double-precision DOUBLE column:
	// https://dev.mysql.com/doc/refman/8.0/en/floating-point-types.html
	switch typ.T {
	case mysql.TypeDouble:
		return field.Float(name)

	case mysql.TypeFloat:
		if typ.Precision == 0 {
			// If precision and scale are not specified, use Float32.
			return field.Float32(name)
		}
		// If precision is specified, use Float64.
		if typ.Precision > 23 {
			return field.Float(name)
		}

	case mysql.TypeReal:
		// MySQL's REAL is an alias for FLOAT, so we treat it as such.
		if typ.Precision == 0 {
			// If precision and scale are not specified, use Float32.
			return field.Float32(name)
		}
		// If precision is specified, use Float64.
		if typ.Precision > 23 {
			return field.Float(name)
		}
	}

	switch typ.T {
	case postgres.TypeReal:
		return field.Float32(name)
	case postgres.TypeDouble:
		return field.Float(name)
	case postgres.TypeFloat8:
		return field.Float(name)
	case postgres.TypeFloat4:
		return field.Float32(name)
	case postgres.TypeFloat:
		return field.Float(name)
	}

	return field.Float32(name)
}

func (t *Text) convertInteger(typ *schema.IntegerType, name string) (f ent.Field) {
	if typ.Unsigned {
		switch typ.T {
		case mTinyInt:
			f = field.Uint8(name)
		case mSmallInt:
			f = field.Uint16(name)
		case mMediumInt:
			f = field.Uint32(name)
		case mInt:
			f = field.Uint32(name)
		case mBigInt:
			f = field.Uint64(name)
		}
		return f
	}

	switch typ.T {
	case mTinyInt:
		f = field.Int8(name)
	case mSmallInt:
		f = field.Int16(name)
	case mMediumInt:
		f = field.Int32(name)
	case mInt:
		f = field.Int32(name)
	case mBigInt:
		// Int64 is not used on purpose.
		f = field.Int(name)

	case pInteger:
		f = field.Int32(name)
	}

	return f
}

// smallserial- 2 bytes - small autoincrementing integer 1 to 32767
// serial - 4 bytes autoincrementing integer 1 to 2147483647
// bigserial - 8 bytes large autoincrementing integer	1 to 9223372036854775807
func (t *Text) convertSerial(typ *postgres.SerialType, name string) ent.Field {
	return field.Uint(name).
		SchemaType(map[string]string{
			dialect.Postgres: typ.T, // Override Postgres.
		})
}
