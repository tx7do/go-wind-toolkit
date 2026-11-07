package entimport

import (
	"encoding/json"
	"fmt"
	"log"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"

	"github.com/google/uuid"
)

// textField 把 atlas 列转换为 ent 字段——DDL 文本驱动专用:DDL 里的类型
// 可能来自 MySQL 或 Postgres 方言,故为两方言映射的并集
// (mysql 的类型全集 + postgres 的 SerialType/UUIDType 扩展)。
func textField(column *schema.Column) (f ent.Field, err error) {
	name := column.Name
	log.Printf("[entimport/text] column: %s, Type: %T, Raw: %s", name, column.Type.Type, column.Type.Raw)
	switch typ := column.Type.Type.(type) {
	case *schema.BinaryType:
		f = field.Bytes(name)
	case *schema.BoolType:
		f = field.Bool(name)
	case *schema.DecimalType:
		f = textConvertDecimal(typ, name)
	case *schema.EnumType:
		f = field.Enum(name).Values(typ.Values...)
	case *schema.FloatType:
		f = textConvertFloat(typ, name)
	case *schema.IntegerType:
		f = textConvertInteger(typ, name)
	case *schema.JSONType:
		f = field.JSON(name, json.RawMessage{})
	case *schema.StringType:
		f = field.String(name)
	case *schema.TimeType:
		f = field.Time(name)

	case *postgres.SerialType:
		f = textConvertSerial(typ, name)
	case *postgres.UUIDType:
		f = field.UUID(name, uuid.New())

	default:
		return nil, fmt.Errorf("entimport: unsupported type %q for column %v", typ, column.Name)
	}
	applyColumnAttributes(f, column)
	return f, err
}

func textConvertFloat(typ *schema.FloatType, name string) (f ent.Field) {
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

func textConvertInteger(typ *schema.IntegerType, name string) (f ent.Field) {
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

func textConvertDecimal(typ *schema.DecimalType, name string) ent.Field {
	return field.Float(name)
}

// smallserial- 2 bytes - small autoincrementing integer 1 to 32767
// serial - 4 bytes autoincrementing integer 1 to 2147483647
// bigserial - 8 bytes large autoincrementing integer	1 to 9223372036854775807
func textConvertSerial(typ *postgres.SerialType, name string) ent.Field {
	return field.Uint(name).
		SchemaType(map[string]string{
			dialect.Postgres: typ.T, // Override Postgres.
		})
}
