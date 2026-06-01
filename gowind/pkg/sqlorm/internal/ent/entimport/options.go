package entimport

import (
	"context"
	"errors"

	"ariga.io/atlas/sql/schema"

	"entgo.io/contrib/schemast"
	"entgo.io/ent"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlorm/internal/ent/mux"
)

const (
	header         = "Code generated " + "by entimport, DO NOT EDIT."
	to     edgeDir = iota
	from
)

var joinTableErr = errors.New("entimport: join tables must be inspected with ref tables - append `tables` flag")

type (
	edgeDir int

	// relOptions are the options passed down to the functions that creates a relation.
	relOptions struct {
		uniqueEdgeToChild    bool
		recursive            bool
		uniqueEdgeFromParent bool
		refName              string
		edgeField            string
	}

	// fieldFunc receives an Atlas column and converts it to an Ent field.
	fieldFunc func(column *schema.Column) (f ent.Field, err error)

	// SchemaImporter is the interface that wraps the SchemaMutations method.
	SchemaImporter interface {
		// SchemaMutations imports a given schema from a data source and returns a list of schemast mutators.
		SchemaMutations(context.Context) ([]schemast.Mutator, error)
	}

	// ImportOptions are the options passed on to every SchemaImporter.
	ImportOptions struct {
		tables         []string
		excludedTables []string
		schemaPath     string
		driver         *mux.ImportDriver
	}

	// ImportOption allows for managing import configuration using functional options.
	ImportOption func(*ImportOptions)
)

// WithSchemaPath provides a DSN (data source name) for reading the schema and tables from.
func WithSchemaPath(path string) ImportOption {
	return func(i *ImportOptions) {
		i.schemaPath = path
	}
}

// WithTables limits the schema import to a set of given tables (by all tables are imported)
func WithTables(tables []string) ImportOption {
	return func(i *ImportOptions) {
		i.tables = tables
	}
}

// WithExcludedTables supplies the set of tables to exclude.
func WithExcludedTables(tables []string) ImportOption {
	return func(i *ImportOptions) {
		i.excludedTables = tables
	}
}

// WithDriver provides an import driver to be used by SchemaImporter.
func WithDriver(drv *mux.ImportDriver) ImportOption {
	return func(i *ImportOptions) {
		i.driver = drv
	}
}
