package sqlproto

import (
	"context"
	"errors"
	"log"
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
		log.Fatalf("sqlproto: failed to create import driver - %v", err)
		return nil, err
	}
	defer func() {
		//if convertDriver != nil {
		//	if err = convertDriver.Close(); err != nil {
		//		log.Printf("sqlproto: warning - failed to close driver: %v", err)
		//	}
		//}
	}()

	i, err := internal.NewConvert(
		internal.WithIncludedTables(includeTables),
		internal.WithExcludedTables(excludeTables),
		internal.WithDriver(convertDriver),
		internal.WithSchemaPath(normalizedDSN),
	)
	if err != nil {
		log.Fatalf("sqlproto: create importer failed: %v", err)
		return nil, err
	}

	tableDatas, err := i.SchemaTables(ctx)
	if err != nil {
		log.Fatalf("sqlproto: schema import failed - %v", err)
		return nil, err
	}

	if exportProto {
		if err = WriteServicesProto(
			*outputPath,
			*serviceType,
			*moduleName, *sourceModuleName, *moduleVersion,
			tableDatas,
		); err != nil {
			log.Fatalf("sqlproto: schema writing failed - %v", err)
		}
	}

	return tableDatas, nil
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
