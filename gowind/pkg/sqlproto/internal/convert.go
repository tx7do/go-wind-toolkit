package internal

import (
	"fmt"
	"log"

	"entgo.io/ent/dialect"

	"ariga.io/atlas/sql/schema"

	_ "github.com/lib/pq"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

func NewConvert(opts ...ConvertOption) (SchemaConverter, error) {
	var (
		si  SchemaConverter
		err error
	)
	i := &ConvertOptions{}
	for _, apply := range opts {
		apply(i)
	}

	switch i.driver.Dialect {
	case dialect.MySQL:
		si, err = NewMySQL(i)
		if err != nil {
			return nil, err
		}

	case dialect.Postgres:
		si, err = NewPostgreSQL(i)
		if err != nil {
			return nil, err
		}

	case "text":
		si, err = NewText(i)
		if err != nil {
			return nil, err
		}

	case schemasource.DialectEntSchema, schemasource.DialectGormSchema:
		si, err = NewGoSchema(i)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("sqlproto: unsupported dialect %q", i.driver.Dialect)
	}

	return si, err
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
