package internal

import (
	"context"
	"fmt"
	"log"

	"entgo.io/ent/dialect"

	"ariga.io/atlas/sql/schema"

	_ "github.com/lib/pq"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

func NewConvert(opts ...ConvertOption) (SchemaConverter, error) {
	i := &ConvertOptions{}
	for _, apply := range opts {
		apply(i)
	}

	var fnc fieldTypeFunc
	switch i.driver.Dialect {
	case dialect.MySQL:
		fnc = MySQLFieldType
	case dialect.Postgres:
		fnc = PostgresFieldType
	case "text":
		fnc = textFieldType
	case schemasource.DialectEntSchema, schemasource.DialectGormSchema:
		fnc = goSchemaFieldType
	default:
		return nil, fmt.Errorf("sqlproto: unsupported dialect %q", i.driver.Dialect)
	}

	return &converterFunc{opts: i, fnc: fnc}, nil
}

// converterFunc 是四种方言(mysql/postgres/text/ent|gorm 源码)共用的
// SchemaConverter 泛型实现:驱动带 Inspector(数据库直连与 Go 源码源)时走
// atlas 内省;schemaPath 持有 SQL 正文(text 驱动,mux 的 textProvider 不带
// Inspector)时走 ParseDDLText。方言差异只剩类型映射函数(fieldTypeFunc),
// 骨架不再四写。
type converterFunc struct {
	opts *ConvertOptions
	fnc  fieldTypeFunc
}

// SchemaTables implements SchemaConverter.
func (c *converterFunc) SchemaTables(ctx context.Context) ([]*TableData, error) {
	if c.opts.driver.Inspector != nil {
		s, err := c.opts.driver.InspectSchema(ctx, c.opts.driver.SchemaName, &schema.InspectOptions{
			Tables: c.opts.includedTables,
		})
		if err != nil {
			return nil, fmt.Errorf("sqlproto: inspect schema: %w", err)
		}
		// exclude 过滤统一走 schemasource.FilterTables(include 已由
		// InspectOptions.Tables 在 inspect 期过滤,这里不再重复)。
		tables := schemasource.FilterTables(s.Tables, nil, c.opts.excludedTables)
		return schemaTables(c.fnc, tables)
	}

	// text 驱动:schemaPath 是 SQL 文本路径或 text:// 内联正文。
	sqlText := schemasource.LoadSQLFromFile(c.opts.schemaPath)
	if sqlText == "" {
		return nil, fmt.Errorf("无法加载 SQL 文件: %v", c.opts.schemaPath)
	}
	var s schema.Schema
	if _, err := schemasource.ParseDDLText(sqlText, nil, &s); err != nil {
		return nil, err
	}
	if len(s.Tables) == 0 {
		return nil, fmt.Errorf("无效的 SQL: 无法解析任何有效的表")
	}
	// text 路径的 include 由 ParseDDLText 的 InspectOptions 承担不了
	// (这里 opts 传 nil),与 exclude 一起统一走 FilterTables。
	tables := schemasource.FilterTables(s.Tables, c.opts.includedTables, c.opts.excludedTables)
	return schemaTables(c.fnc, tables)
}

func schemaTables(fnc fieldTypeFunc, tables []*schema.Table) ([]*TableData, error) {
	tableDatas := make([]*TableData, 0)
	joinTables := make(map[string]*schema.Table)
	for _, table := range tables {
		if schemasource.IsJoinTable(table) {
			joinTables[table.Name] = table
			continue
		}

		log.Println("表:", table.Name)

		node, err := convertTable(fnc, table)
		if err != nil {
			return nil, fmt.Errorf("entimport: issue with table %v: %w", table.Name, err)
		}

		tableDatas = append(tableDatas, node)
	}

	return tableDatas, nil
}

func convertTable(fnc fieldTypeFunc, table *schema.Table) (*TableData, error) {
	var tableData TableData

	tableData.Name = table.Name

	for _, attr := range table.Attrs {
		switch a := attr.(type) {
		case *schema.Comment:
			tableData.Comment = a.Text
			//fmt.Println("schema.Comment", comment)

		case *schema.Charset:
			//fmt.Println("schema.Charset", a.V)
			tableData.Charset = a.V

		case *schema.Collation:
			//fmt.Println("schema.Collation", a.V)
			tableData.Collation = a.V
		}
	}

	// 构建主键集合
	pkSet := make(map[string]bool)
	if table.PrimaryKey != nil {
		for _, part := range table.PrimaryKey.Parts {
			pkSet[part.C.Name] = true
		}
	}

	for _, column := range table.Columns {
		//log.Println(column.Name)

		fieldData := FieldData{
			Name:         column.Name,
			Type:         fnc(column.Type.Raw),
			SqlType:      schemasource.ParseSQLType(column.Type.Raw).Key(),
			Null:         column.Type.Null,
			IsPrimaryKey: pkSet[column.Name],
		}

		if fieldData.Type == "" {
			// 兜底成 string 不等于类型正确:int64 的精度、二进制的字节串都会在这个
			// 看不见的转换里丢失,所以必须留下能定位到表与列的告警。
			log.Printf("sqlproto: 表 %s 的列 %s 类型 %q 没有对应的 Proto 映射,按 string 处理", table.Name, column.Name, column.Type.Raw)
			fieldData.Type = "string"
		}

		for _, attr := range column.Attrs {
			switch a := attr.(type) {
			case *schema.Comment:
				fieldData.Comment = a.Text
			}
		}

		tableData.Fields = append(tableData.Fields, fieldData)
	}

	return &tableData, nil
}
