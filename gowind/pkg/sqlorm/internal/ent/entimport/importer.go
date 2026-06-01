package entimport

import (
	"context"
	"errors"
	"fmt"
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
		return fmt.Errorf("entimport: failed to create import driver: %w", err)
	}
	defer func(drv *mux.ImportDriver) {
		if drv != nil {
			_ = drv.Close()
		}
	}(drv)

	i, err := NewImport(
		WithTables(includeTables),
		WithExcludedTables(excludeTables),
		WithDriver(drv),
		WithSchemaPath(normalizedDSN),
	)
	if err != nil {
		return fmt.Errorf("entimport: create importer failed: %w", err)
	}

	mutations, err := i.SchemaMutations(ctx)
	if err != nil {
		return fmt.Errorf("entimport: schema import failed: %w", err)
	}

	if err = WriteSchema(mutations, WithSchemaPath(*schemaPath)); err != nil {
		return fmt.Errorf("entimport: schema writing failed: %w", err)
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
