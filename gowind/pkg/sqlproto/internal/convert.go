package internal

import (
	"fmt"
	"log"
	"strings"

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

		log.Println("***********", table.Name)

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
			SqlType:      strings.ToUpper(strings.TrimSpace(strings.SplitN(column.Type.Raw, "(", 2)[0])),
			Null:         column.Type.Null,
			IsPrimaryKey: pkSet[column.Name],
		}

		if fieldData.Type == "" {
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
