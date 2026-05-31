package entimport

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlorm/internal/ent/mux"
)

// Importer imports the schema from the database specified by the DSN and writes it to the schemaPath.
func Importer(ctx context.Context, dsn, schemaPath *string, includeTables, excludeTables []string) error {
	if schemaPath == nil {
		return errors.New("entimport: schema path is nil")
	}
	if dsn == nil {
		return errors.New("entimport: dsn is nil")
	}

	_ = os.MkdirAll(*schemaPath, os.ModePerm)

	// Normalize the DSN to ensure it has a valid scheme
	normalizedDSN := normalizeDSN(*dsn)

	drv, err := mux.Default.OpenImport(normalizedDSN)
	if err != nil {
		log.Fatalf("entimport: failed to create import driver - %v", err)
	}
	defer func(drv *mux.ImportDriver) {
		//if drv != nil {
		//	err = drv.Close()
		//	if err != nil {
		//		log.Fatalf("entimport: failed to close import driver - %v", err)
		//	}
		//}
	}(drv)

	i, err := NewImport(
		WithTables(includeTables),
		WithExcludedTables(excludeTables),
		WithDriver(drv),
		WithSchemaPath(normalizedDSN),
	)
	if err != nil {
		log.Fatalf("entimport: create importer failed: %v", err)
	}

	mutations, err := i.SchemaMutations(ctx)
	if err != nil {
		log.Fatalf("entimport: schema import failed - %v", err)
	}

	if err = WriteSchema(mutations, WithSchemaPath(*schemaPath)); err != nil {
		log.Fatalf("entimport: schema writing failed - %v", err)
	}

	return nil
}

// normalizeDSN normalizes the DSN to ensure it has a valid scheme.
// If the input is a file path, it will be prefixed with "file://".
// If it's SQL text content, it will be prefixed with "text://".
// If it already has a scheme (mysql://, postgres://, etc.), it's returned as-is.
func normalizeDSN(dsn string) string {
	// Check if it already has a scheme
	if strings.Contains(dsn, "://") {
		return dsn
	}

	// Check if it's a file path
	if _, err := os.Stat(dsn); err == nil {
		return "file://" + dsn
	}

	// Treat it as SQL text content
	return "text://" + dsn
}
