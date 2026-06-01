package sqlproto

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto/internal"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto/internal/mux"
)

type TableDataArray []*internal.TableData

// Convert converts the database schema into a protocol buffer definition.
func Convert(
	ctx context.Context,
	dsn, outputPath *string,
	moduleName, sourceModuleName, moduleVersion *string,
	serviceType *string,
	strategy string,
	customPackages map[string]string,
	includeTables, excludeTables []string,
	exportProto bool,
) (TableDataArray, error) {
	if outputPath == nil {
		return nil, errors.New("sqlproto: proto file output path is nil")
	}
	if dsn == nil {
		return nil, errors.New("sqlproto: dsn is nil")
	}
	if moduleName == nil {
		return nil, errors.New("sqlproto: proto module is nil")
	}

	_ = os.MkdirAll(*outputPath, os.ModePerm)

	// Normalize the DSN to ensure it has a valid scheme
	normalizedDSN := normalizeDSN(*dsn)

	convertDriver, err := mux.Default.OpenConvert(normalizedDSN)
	if err != nil {
		return nil, fmt.Errorf("sqlproto: failed to create import driver: %w", err)
	}
	defer func() {
		if convertDriver != nil {
			_ = convertDriver.Close()
		}
	}()

	i, err := internal.NewConvert(
		internal.WithIncludedTables(includeTables),
		internal.WithExcludedTables(excludeTables),
		internal.WithDriver(convertDriver),
		internal.WithSchemaPath(normalizedDSN),
	)
	if err != nil {
		return nil, fmt.Errorf("sqlproto: create importer failed: %w", err)
	}

	tableDatas, err := i.SchemaTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("sqlproto: schema import failed: %w", err)
	}

	if exportProto {
		if err = WriteServicesProto(
			*outputPath,
			*serviceType,
			strategy,
			*moduleName, *sourceModuleName, *moduleVersion,
			tableDatas,
			customPackages,
		); err != nil {
			return nil, fmt.Errorf("sqlproto: schema writing failed: %w", err)
		}
	}

	return tableDatas, nil
}

// normalizeDSN normalizes the DSN to ensure it has a valid scheme.
// If it already has a scheme (mysql://, postgres://, etc.), it's returned as-is.
// If it's a file path, it will be prefixed with "file://".
// Otherwise, it's treated as SQL text content and prefixed with "text://".
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
