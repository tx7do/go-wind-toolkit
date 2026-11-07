package schemasource

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"ariga.io/atlas/sql/schema"

	ddlparser "github.com/tx7do/go-utils/ddl_parser"
)

// ParseDDLText 把 CREATE TABLE 脚本离线解析为 atlas schema.Schema——
// sqlproto 与 sqlorm(entimport) 两条生成管线共用的 DDL 文本内省实现,
// 取两侧原有实现的语义并集:
//
//   - 表装配(原 entimport 超集):表级 CONSTRAINT ... PRIMARY KEY 归一化、
//     带引号/schema 限定的表名清洗、COMMENT ON COLUMN/TABLE 独立语句的
//     注释提取、InspeateOptions.Tables 的 include 过滤(atlas Inspector 同构);
//   - 列语义(原 sqlproto 超集):unsigned 回原文重解析且保留在 Raw 里
//     (Raw 是下游字符串映射的唯一输入)、PRIMARY KEY 列强制 NOT NULL。
//
// 两侧消费面已核实:entimport 只读 Type.Null(不读 Raw);sqlproto 读
// Type.Raw 与 Type.Null。对该并集,entimport 侧的行为变化仅为 PK 列
// Optional=false(SQL 语义上主键本就非空,属正确性修复);sqlproto 侧
// 对干净 MySQL DDL 零漂移,对 Postgres 风格 DDL 从解析残缺变为正确。
//
// s 为 nil 时内部分配;传入现有 *schema.Schema 时把解析出的表追加进去
// (Table.Schema 反指 ParseDDLText 内部的 schema,调用方不应依赖该指针)。
func ParseDDLText(sqlContent string, opts *schema.InspectOptions, s *schema.Schema) (*schema.Schema, error) {
	if strings.TrimSpace(sqlContent) == "" {
		return nil, fmt.Errorf("schemasource: SQL 文本为空,无法解析")
	}
	if s == nil {
		s = &schema.Schema{}
	}

	// ddl_parser 只识别以 PRIMARY KEY 开头的分片,`CONSTRAINT <名> PRIMARY KEY`
	// 会被当作约束整体跳过,导致 Postgres 风格 DDL 丢失主键,解析前先归一化。
	tables, err := ddlparser.ParseCreateTables(normalizeTableConstraints(sqlContent))
	if err != nil {
		return nil, fmt.Errorf("schemasource: 解析 DDL 失败: %v", err)
	}

	// Postgres 的注释是独立语句,不在 CREATE TABLE 内。
	columnComments, tableComments := extractCommentOn(sqlContent)

	// include 过滤与数据库直连路径的 atlas InspectOptions.Tables 行为一致
	// (nil/空 = 全部表);按清洗后的末端表名精确匹配。
	var includeTables map[string]bool
	if opts != nil && len(opts.Tables) > 0 {
		includeTables = make(map[string]bool, len(opts.Tables))
		for _, name := range opts.Tables {
			includeTables[cleanTableName(name)] = true
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
			table.Attrs = append(table.Attrs, &schema.Comment{Text: tableComment})
		}
		if tbl.Collation != "" {
			table.Attrs = append(table.Attrs, &schema.Collation{V: tbl.Collation})
		}
		if tbl.Charset != "" {
			table.Attrs = append(table.Attrs, &schema.Charset{V: tbl.Charset})
		}

		for _, col := range tbl.Columns {
			colType, err := ddlColumnType(name, col, sqlContent)
			if err != nil {
				log.Printf("schemasource: 表 %s 的列 %s 解析失败: %v", name, col.Name, err)
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
					Parts: []*schema.IndexPart{{C: column}},
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

// ddlColumnType 把一列的声明解析为 atlas 列类型。
//
// ddl_parser 不把 unsigned 带进 col.Type(如 INT UNSIGNED 只返回 "int"),
// 需要回原文按词边界确认该列声明为 unsigned 后再重新解析一次;成功时
// Raw 必须带上 " unsigned"——Raw 是下游字符串映射(sqlproto 的
// MySQLFieldType 等)的唯一输入,unsigned 留在里面才不会把无符号整型
// 映射成有符号。PRIMARY KEY 列恒为 NOT NULL。
func ddlColumnType(tableName string, col ddlparser.ColumnDef, sqlContent string) (*schema.ColumnType, error) {
	parsedType, err := ParseType(col.Type)
	if err != nil {
		return nil, err
	}

	raw := col.Type
	if intType, ok := parsedType.(*schema.IntegerType); ok && !intType.Unsigned {
		if ColumnIsUnsigned(tableName, col.Name, col.Type, sqlContent) {
			unsignedType, uErr := ParseType(col.Type + " unsigned")
			if uErr == nil {
				if uIntType, ok := unsignedType.(*schema.IntegerType); ok && uIntType.Unsigned {
					parsedType = uIntType
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

// constraintPKRe 匹配表级 `CONSTRAINT <名> PRIMARY KEY`。
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
	// commentOnColumnRe 匹配 `COMMENT ON COLUMN <表路径>.<列> IS '<文本>'`。
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
