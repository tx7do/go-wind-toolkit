package internal

import (
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto/internal/mux"
)

type Options struct {
	Driver string `json:"driver"` // Driver name, e.g., "mysql", "postgres"
	Source string `json:"source"` // Data source name (DSN), e.g., "mysql://user:pass@tcp(localhost:3306)/dbname"

	IncludedTables []string `json:"included_tables"` // IncludedTables to inspect (all if empty)
	ExcludedTables []string `json:"excluded_tables"` // ExcludedTables to exclude from inspection

	OutputPath string `json:"output_path"` // Path to save the generated code

	SourceModule string `json:"sourceModule"` // Source module name, for REST service generate, e.g., "admin"
	Module       string `json:"module"`       // Module name for the generated code, e.g., "admin"
	Version      string `json:"version"`      // Version of the module, e.g., "v1"
	Service      string `json:"service"`      // generate service code, "rest" for REST service, "grpc" for gRPC service
}

type (
	// ConvertOptions are the options passed on to every SchemaConverter.
	ConvertOptions struct {
		includedTables []string
		excludedTables []string

		protoPath  string
		schemaPath string

		sourceModuleName string // for REST service, the source module name
		moduleName       string
		moduleVersion    string

		serviceType string

		driver *mux.ConvertDriver
	}

	// ConvertOption allows for managing import configuration using functional options.
	ConvertOption func(*ConvertOptions)
)

// WithProtoPath provides a path for writing the generated protocol buffer definitions.
func WithProtoPath(path string) ConvertOption {
	return func(i *ConvertOptions) {
		i.protoPath = path
	}
}

func WithSchemaPath(path string) ConvertOption {
	return func(i *ConvertOptions) {
		i.schemaPath = path
	}
}

func WithSourceModuleName(name string) ConvertOption {
	return func(i *ConvertOptions) {
		i.sourceModuleName = name
	}
}

func WithModuleName(name string) ConvertOption {
	return func(i *ConvertOptions) {
		i.moduleName = name
	}
}

func WithModuleVersion(ver string) ConvertOption {
	return func(i *ConvertOptions) {
		i.moduleVersion = ver
	}
}

func WithServiceType(serviceType string) ConvertOption {
	return func(i *ConvertOptions) {
		i.serviceType = serviceType
	}
}

// WithIncludedTables limits the schema import to a set of given includedTables (by all includedTables are imported)
func WithIncludedTables(tables []string) ConvertOption {
	return func(i *ConvertOptions) {
		i.includedTables = tables
	}
}

// WithExcludedTables supplies the set of includedTables to exclude.
func WithExcludedTables(tables []string) ConvertOption {
	return func(i *ConvertOptions) {
		i.excludedTables = tables
	}
}

// WithDriver provides an import driver to be used by SchemaConverter.
func WithDriver(drv *mux.ConvertDriver) ConvertOption {
	return func(i *ConvertOptions) {
		i.driver = drv
	}
}
