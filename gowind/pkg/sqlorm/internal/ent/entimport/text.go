package entimport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	"entgo.io/contrib/schemast"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"

	"github.com/google/uuid"

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

// InspectSchema 把 SQL 文本解析为 atlas schema:表装配与列语义已收敛到
// schemasource.ParseDDLText(与 sqlproto 共用同一实现),这里保留
// SchemaImporter 接口形态,只做 sink(列 → ent.Field)。
func (t *Text) InspectSchema(ctx context.Context, sqlContent string, opts *schema.InspectOptions, s *schema.Schema) (*schema.Schema, error) {
	return schemasource.ParseDDLText(sqlContent, opts, s)
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

	// exclude 过滤统一走 schemasource.FilterTables(include 已由
	// ParseDDLText 的 InspectOptions.Tables 过滤)。
	tables := schemasource.FilterTables(s.Tables, nil, t.excludedTables)

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
