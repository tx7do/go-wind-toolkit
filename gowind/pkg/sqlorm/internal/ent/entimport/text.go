package entimport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/google/uuid"
	ddlparser "github.com/tx7do/go-utils/ddl_parser"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	"entgo.io/contrib/schemast"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

type Text struct {
	*ImportOptions
}

func NewText(i *ImportOptions) (*Text, error) {
	return &Text{
		ImportOptions: i,
	}, nil
}

func (t *Text) toColumnType(col ddlparser.ColumnDef, sqlContent string) (*schema.ColumnType, error) {
	parsedType, err := schemasource.ParseType(col.Type)
	if err != nil {
		return nil, err
	}

	// ddl_parser 不将 unsigned 纳入 col.Type（如 INT UNSIGNED 只返回 "int"），
	// 需要从原始 SQL 中检测列是否含 unsigned，并重新解析。
	if intType, ok := parsedType.(*schema.IntegerType); ok && !intType.Unsigned {
		if isColumnUnsigned(col.Name, col.Type, sqlContent) {
				unsignedType, uErr := schemasource.ParseType(col.Type + " unsigned")
			if uErr == nil {
				if uIntType, ok := unsignedType.(*schema.IntegerType); ok && uIntType.Unsigned {
					parsedType = uIntType
				}
			}
		}
	}

	return &schema.ColumnType{
		Type: parsedType,
		Raw:  col.Type,
		Null: col.Nullable,
	}, nil
}

// isColumnUnsigned 检测原始 SQL 中指定列是否包含 UNSIGNED 关键字
func isColumnUnsigned(colName, colType, sqlContent string) bool {
	// 在 SQL 内容中查找列名，检测该行是否含 unsigned
	sqlLower := strings.ToLower(sqlContent)
	colNameLower := strings.ToLower(colName)

	// 查找列名出现的位置
	idx := 0
	for {
		pos := strings.Index(sqlLower[idx:], colNameLower)
		if pos == -1 {
			break
		}
		absPos := idx + pos
		// 从列名开始找到下一个逗号或行尾，作为该列的定义范围
		endPos := len(sqlLower)
		if comma := strings.Index(sqlLower[absPos:], ","); comma != -1 {
			endPos = absPos + comma
		}
		colDef := sqlLower[absPos:endPos]
		// 检查该列定义中是否包含 unsigned
		if strings.Contains(colDef, " unsigned") {
			return true
		}
		idx = absPos + len(colNameLower)
	}
	return false
}

func (t *Text) InspectSchema(ctx context.Context, sqlContent string, opts *schema.InspectOptions, s *schema.Schema) (*schema.Schema, error) {
	if sqlContent == "" {
		return nil, fmt.Errorf("SQL 文本为空，无法解析")
	}
	if s == nil {
		s = &schema.Schema{}
	}

	// 解析 SQL 文本
	tables, err := ddlparser.ParseCreateTables(normalizeTableConstraints(sqlContent))
	if err != nil {
		return nil, fmt.Errorf("解析失败: %v", err)
	}

	columnComments, tableComments := extractCommentOn(sqlContent)

	// 按 InspectOptions.Tables 过滤（nil/空 = 全部表），
	// 与数据库直连路径的 atlas Inspector 行为一致
	var includeTables map[string]bool
	if opts != nil && len(opts.Tables) > 0 {
		includeTables = make(map[string]bool, len(opts.Tables))
		for _, name := range opts.Tables {
			includeTables[name] = true
		}
	}

	for _, tbl := range tables {
		name := cleanTableName(tbl.Name)
		if includeTables != nil && !includeTables[name] {
			continue
		}

		table := &schema.Table{
			Name:   name,
			Schema: s,
		}

		tableComment := tbl.Comment
		if tableComment == "" {
			tableComment = tableComments[name]
		}
		if tableComment != "" {
			table.Attrs = append(table.Attrs, &schema.Comment{
				Text: tableComment,
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

			colType, err := t.toColumnType(col, sqlContent)
			if err != nil {
				log.Printf("解析失败: %v\n", err)
				continue
			}

			column := &schema.Column{
				Name: col.Name,
				Type: colType,
			}
			colComment := col.Comment
			if colComment == "" {
				colComment = columnComments[name+"."+col.Name]
			}
			if colComment != "" {
				column.SetComment(colComment)
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
	sqlText := schemasource.LoadSQLFromFile(t.schemaPath)
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
	log.Printf("[entimport/text] column: %s, Type: %T, Raw: %s", name, column.Type.Type, column.Type.Raw)
	switch typ := column.Type.Type.(type) {
	case *schema.BinaryType:
		f = field.Bytes(name)
	case *schema.BoolType:
		f = field.Bool(name)
	case *schema.DecimalType:
		f = t.convertDecimal(typ, name)
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
		default:
			f = field.Uint64(name)
		}
		return f
	}

	switch typ.T {
	case mTinyInt:
		f = field.Int8(name)
	case mSmallInt, pInt2:
		f = field.Int16(name)
	case mMediumInt:
		f = field.Int32(name)
	case mInt, pInteger, pInt4:
		f = field.Int32(name)
	case mBigInt, pInt8:
		f = field.Int64(name)
	default:
		f = field.Int(name).
			SchemaType(map[string]string{
				dialect.Postgres: typ.T,
			})
	}

	return f
}

func (t *Text) convertDecimal(typ *schema.DecimalType, name string) ent.Field {
	return field.Float(name)
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

// constraintPKRe 匹配表级 `CONSTRAINT <名> PRIMARY KEY`。ddl_parser 只识别
// 以 PRIMARY KEY 开头的分片,CONSTRAINT 名字头会被当作约束整体跳过,
// 导致 Postgres 风格 DDL 丢失主键,解析前先归一化为 `PRIMARY KEY`。
var constraintPKRe = regexp.MustCompile(`(?is)constraint\s+(?:"[^"]*"|` + "`[^`]*`" + `|\[[^\]]*\]|\S+)\s+primary\s+key`)

func normalizeTableConstraints(sqlContent string) string {
	return constraintPKRe.ReplaceAllString(sqlContent, "PRIMARY KEY")
}

// cleanTableName 去掉表名的引号与 schema/catalog 限定,只保留末端表名。
// ddl_parser 对 "public"."addresses" 这类带引号限定名会解析出
// `public"."addresses` 这样的残缺名。
func cleanTableName(name string) string {
	name = unquoteIdentifier(name)
	if idx := strings.LastIndex(name, "."); idx != -1 {
		name = name[idx+1:]
	}
	return strings.TrimSpace(name)
}

// unquoteIdentifier 去掉标识符的引号包裹,保留限定分隔符。
func unquoteIdentifier(s string) string {
	return strings.NewReplacer("\"", "", "`", "", "[", "", "]", "").Replace(s)
}

var (
	// commentOnColumnRe 匹配 `COMMENT ON COLUMN <表路径>.<列> IS '<文本>'`,
	// Postgres 的列注释是独立语句,不在 CREATE TABLE 内。
	commentOnColumnRe = regexp.MustCompile(`(?is)comment\s+on\s+column\s+(.+?)\s+is\s+'((?:[^']|'')*)'`)
	// commentOnTableRe 匹配 `COMMENT ON TABLE <表路径> IS '<文本>'`。
	commentOnTableRe = regexp.MustCompile(`(?is)comment\s+on\s+table\s+(.+?)\s+is\s+'((?:[^']|'')*)'`)
)

// extractCommentOn 提取 COMMENT ON COLUMN/TABLE 独立语句的注释,
// 返回 列注释(key: "表.列")与表注释(key: 表名),表/列名均已去限定去引号。
func extractCommentOn(sqlContent string) (columnComments, tableComments map[string]string) {
	columnComments = make(map[string]string)
	tableComments = make(map[string]string)
	for _, m := range commentOnColumnRe.FindAllStringSubmatch(sqlContent, -1) {
		parts := strings.Split(unquoteIdentifier(m[1]), ".")
		if len(parts) < 2 {
			continue
		}
		table := parts[len(parts)-2]
		column := parts[len(parts)-1]
		columnComments[table+"."+column] = unescapeSQLString(m[2])
	}
	for _, m := range commentOnTableRe.FindAllStringSubmatch(sqlContent, -1) {
		tableComments[cleanTableName(m[1])] = unescapeSQLString(m[2])
	}
	return columnComments, tableComments
}

// unescapeSQLString 还原 SQL 字符串字面量中的 '' 转义。
func unescapeSQLString(s string) string {
	return strings.ReplaceAll(s, "''", "'")
}
