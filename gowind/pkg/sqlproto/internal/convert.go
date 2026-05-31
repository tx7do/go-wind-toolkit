package internal

import (
	"fmt"
	"log"

	"entgo.io/ent"
	"entgo.io/ent/dialect"

	"ariga.io/atlas/sql/schema"

	_ "github.com/lib/pq"
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

	default:
		return nil, fmt.Errorf("sqlproto: unsupported dialect %q", i.driver.Dialect)
	}

	return si, err
}

// applyColumnAttributes adds column attributes to a given ent field.
func applyColumnAttributes(f ent.Field, col *schema.Column) {
	desc := f.Descriptor()
	desc.Optional = col.Type.Null
	for _, attr := range col.Attrs {
		if a, ok := attr.(*schema.Comment); ok {
			desc.Comment = a.Text
		}
	}
}

// Note: at this moment ent doesn't support fields on m2m relations.
func isJoinTable(table *schema.Table) bool {
	if table.PrimaryKey == nil || len(table.PrimaryKey.Parts) != 2 || len(table.ForeignKeys) != 2 {
		return false
	}
	// Make sure that the foreign key columns exactly match primary key column.
	for _, fk := range table.ForeignKeys {
		if len(fk.Columns) != 1 {
			return false
		}
		if fk.Columns[0] != table.PrimaryKey.Parts[0].C && fk.Columns[0] != table.PrimaryKey.Parts[1].C {
			return false
		}
	}
	return true
}

func schemaTables(fnc fieldTypeFunc, tables []*schema.Table) ([]*TableData, error) {
	tableDatas := make([]*TableData, 0)
	joinTables := make(map[string]*schema.Table)
	for _, table := range tables {
		if isJoinTable(table) {
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

	for _, column := range table.Columns {
		//log.Println(column.Name)

		fieldData := FieldData{
			Name: column.Name,
			Type: fnc(column.Type.Raw),
			Null: column.Type.Null,
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
