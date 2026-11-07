package entimport

import (
	"context"
	"encoding/json"
	"fmt"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"

	"entgo.io/contrib/schemast"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

const (
	mTinyInt   = "tinyint"   // MYSQL_TYPE_TINY
	mSmallInt  = "smallint"  // MYSQL_TYPE_SHORT
	mInt       = "int"       // MYSQL_TYPE_LONG
	mMediumInt = "mediumint" // MYSQL_TYPE_INT24
	mBigInt    = "bigint"    // MYSQL_TYPE_LONGLONG
)

// MySQL holds the schema import options and an Atlas inspector instance
type MySQL struct {
	*ImportOptions
}

// NewMySQL - create aמ import structure for MySQL.
func NewMySQL(i *ImportOptions) (*MySQL, error) {
	return &MySQL{
		ImportOptions: i,
	}, nil
}

// SchemaMutations implements SchemaImporter.
func (m *MySQL) SchemaMutations(ctx context.Context) ([]schemast.Mutator, error) {
	inspectOptions := &schema.InspectOptions{
		Tables: m.tables,
	}
	s, err := m.driver.InspectSchema(ctx, m.driver.SchemaName, inspectOptions)
	if err != nil {
		return nil, err
	}
	// exclude 过滤统一走 schemasource.FilterTables(atlas 只吃 include,
	// InspectOptions.Tables 已按 include 过滤,这里不再重复)。
	tables := schemasource.FilterTables(s.Tables, nil, m.excludedTables)
	return schemaMutations(m.field, tables)
}

func (m *MySQL) field(column *schema.Column) (f ent.Field, err error) {
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
		f = m.convertFloat(typ, name)
	case *schema.IntegerType:
		f = m.convertInteger(typ, name)
	case *schema.JSONType:
		f = field.JSON(name, json.RawMessage{})
	case *schema.StringType:
		f = field.String(name)
	case *schema.TimeType:
		f = field.Time(name)
	default:
		return nil, fmt.Errorf("entimport: unsupported type %q for column %v", typ, column.Name)
	}
	applyColumnAttributes(f, column)
	return f, err
}

func (m *MySQL) convertFloat(typ *schema.FloatType, name string) (f ent.Field) {
	// Precision from 0 to 23 results in a 4-byte single-precision FLOAT column.
	// Precision from 24 to 53 results in an 8-byte double-precision DOUBLE column:
	// https://dev.mysql.com/doc/refman/8.0/en/floating-point-types.html
	if typ.T == mysql.TypeDouble {
		return field.Float(name)
	}
	return field.Float32(name)
}

func (m *MySQL) convertInteger(typ *schema.IntegerType, name string) (f ent.Field) {
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
		f = field.Int(name)
	case mBigInt:
		f = field.Int64(name)
	}
	return f
}
