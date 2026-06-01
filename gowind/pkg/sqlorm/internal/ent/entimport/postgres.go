package entimport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	"entgo.io/contrib/schemast"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

const (
	pSmallInt = "smallint" // smallint - 2 bytes small-range integer -32768 to +32767.
	pInteger  = "integer"  // integer - 4 bytes typical choice for integer	-2147483648 to +2147483647.
	pBigInt   = "bigint"   // bigint - 8 bytes large-range integer	-9223372036854775808 to 9223372036854775807.
)

// Postgres implements SchemaImporter for PostgreSQL databases.
type Postgres struct {
	*ImportOptions
}

// NewPostgreSQL - returns a new *Postgres.
func NewPostgreSQL(i *ImportOptions) (SchemaImporter, error) {
	return &Postgres{
		ImportOptions: i,
	}, nil
}

// SchemaMutations implements SchemaImporter.
func (p *Postgres) SchemaMutations(ctx context.Context) ([]schemast.Mutator, error) {
	inspectOptions := &schema.InspectOptions{
		Tables: p.tables,
	}
	s, err := p.driver.InspectSchema(ctx, p.driver.SchemaName, inspectOptions)
	if err != nil {
		return nil, err
	}
	tables := s.Tables
	if p.excludedTables != nil {
		tables = nil
		excludedTableNames := make(map[string]bool)
		for _, t := range p.excludedTables {
			excludedTableNames[t] = true
		}
		// filter out tables that are in excludedTables:
		for _, t := range s.Tables {
			if !excludedTableNames[t.Name] {
				tables = append(tables, t)
			}
		}
	}
	return schemaMutations(p.field, tables)
}

func (p *Postgres) field(column *schema.Column) (f ent.Field, err error) {
	name := column.Name
	log.Printf("[entimport/postgres] column: %s, Type: %T, Raw: %s", name, column.Type.Type, column.Type.Raw)
	switch typ := column.Type.Type.(type) {
	case *schema.BinaryType:
		f = field.Bytes(name)
	case *schema.BoolType:
		f = field.Bool(name)
	case *schema.DecimalType:
		f = p.convertDecimal(typ, name)
	case *schema.EnumType:
		f = field.Enum(name).Values(typ.Values...)
	case *schema.FloatType:
		f = p.convertFloat(typ, name)
	case *schema.IntegerType:
		f = p.convertInteger(typ, name)
	case *schema.JSONType:
		f = field.JSON(name, json.RawMessage{})
	case *schema.StringType:
		f = field.String(name)
	case *schema.TimeType:
		f = field.Time(name)

	case *postgres.SerialType:
		f = p.convertSerial(typ, name)
	case *postgres.UUIDType:
		f = field.UUID(name, uuid.New())

	default:
		return nil, fmt.Errorf("entimport: unsupported type %q for column %v", typ, column.Name)
	}
	applyColumnAttributes(f, column)
	return f, err
}

// decimal, numeric - user-specified precision, exact up to 131072 digits before the decimal point;
// up to 16383 digits after the decimal point.
// real - 4 bytes variable-precision, inexact 6 decimal digits precision.
// double -	8 bytes	variable-precision, inexact	15 decimal digits precision.
func (p *Postgres) convertFloat(typ *schema.FloatType, name string) (f ent.Field) {
	switch typ.T {
	case postgres.TypeReal:
		return field.Float32(name)
	case postgres.TypeDouble, postgres.TypeFloat8:
		return field.Float(name)
	case postgres.TypeFloat4:
		return field.Float32(name)
	default:
		return field.Float(name)
	}
}

// convertDecimal handles decimal/numeric types.
// numeric/decimal with precision and scale.
func (p *Postgres) convertDecimal(typ *schema.DecimalType, name string) ent.Field {
	if typ.Precision <= 18 {
		// 小精度用 float64 足够
		return field.Float(name)
	}
	// 大精度 numeric 使用 string 存储
	return field.Float(name)
}

func (p *Postgres) convertInteger(typ *schema.IntegerType, name string) (f ent.Field) {
	switch typ.T {
	case pSmallInt:
		f = field.Int16(name)
	case pInteger, "int":
		f = field.Int32(name)
	case pBigInt:
		f = field.Int(name)
	default:
		// 其他整数类型回退到 int64
		f = field.Int(name).
			SchemaType(map[string]string{
				dialect.Postgres: typ.T,
			})
	}
	return f
}

// smallserial- 2 bytes - small autoincrementing integer 1 to 32767
// serial - 4 bytes autoincrementing integer 1 to 2147483647
// bigserial - 8 bytes large autoincrementing integer	1 to 9223372036854775807
func (p *Postgres) convertSerial(typ *postgres.SerialType, name string) ent.Field {
	return field.Uint(name).
		SchemaType(map[string]string{
			dialect.Postgres: typ.T, // Override Postgres.
		})
}
