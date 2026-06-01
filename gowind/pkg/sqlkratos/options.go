package sqlkratos

type (
	GeneratorOptions struct {
		Driver string
		Source string // Data Source name (DSN), e.g., "mysql://user:pass@tcp(localhost:3306)/dbname"

		IncludedTables []string
		ExcludedTables []string

		OutputPath string

		SourceModuleName string // for REST service, the Source module name
		ModuleName       string
		ModuleVersion    string

		// ProtoPackageStrategy controls how proto packages are organized.
		// "per-table" - each table gets its own proto package (e.g. user.service.v1)
		// "by-service" - all tables under a service share one proto package (e.g. admin.service.v1)
		// "custom" - use protoPackage from each table's option
		ProtoPackageStrategy string

		OrmType string // ORM type, e.g., "gorm", "sqlx", "ent"

		ProjectName string
		ServiceName string

		Servers []string

		UseRepo bool

		GenerateProto    bool
		GenerateServer   bool
		GenerateService  bool
		GenerateORM      bool
		GenerateData     bool
		GenerateMain     bool
		GenerateConfig   bool
		GenerateMakefile bool
	}

	GeneratorOption func(*GeneratorOptions)
)

func WithOutputPath(path string) GeneratorOption {
	return func(i *GeneratorOptions) {
		i.OutputPath = path
	}
}

func WithSourceModuleName(name string) GeneratorOption {
	return func(i *GeneratorOptions) {
		i.SourceModuleName = name
	}
}

func WithModuleName(name string) GeneratorOption {
	return func(i *GeneratorOptions) {
		i.ModuleName = name
	}
}

func WithModuleVersion(ver string) GeneratorOption {
	return func(i *GeneratorOptions) {
		i.ModuleVersion = ver
	}
}

func WithIncludedTables(tables []string) GeneratorOption {
	return func(i *GeneratorOptions) {
		i.IncludedTables = tables
	}
}

func WithExcludedTables(tables []string) GeneratorOption {
	return func(i *GeneratorOptions) {
		i.ExcludedTables = tables
	}
}

func WithSource(dsn string) GeneratorOption {
	return func(i *GeneratorOptions) {
		i.Source = dsn
	}
}
