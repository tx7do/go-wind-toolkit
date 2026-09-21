package entimport

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

// Importer imports the schema from the database specified by the DSN and writes it to the schemaPath.
func Importer(ctx context.Context, dsn, schemaPath *string, includeTables, excludeTables []string) error {
	if schemaPath == nil {
		return errors.New("entimport: schema path is nil")
	}
	if dsn == nil {
		return errors.New("entimport: dsn is nil")
	}

	_ = os.MkdirAll(*schemaPath, 0o755)

	// Normalize the DSN to ensure it has a valid scheme
	normalizedDSN, err := schemasource.NormalizeDSN(*dsn)
	if err != nil {
		return err
	}

	drv, err := schemasource.Default.Open(normalizedDSN)
	if err != nil {
		return fmt.Errorf("entimport: failed to create import driver: %w", err)
	}
	defer func(drv *schemasource.Driver) {
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
