package schemasource

import (
	"database/sql"
	"net/url"

	atlasmysql "ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"

	"entgo.io/ent/dialect"
	"github.com/go-sql-driver/mysql"
)

func init() {
	Default.RegisterProvider(mysqlProvider, "mysql")
	Default.RegisterProvider(postgresProvider, "postgres", "postgresql")
	Default.RegisterProvider(textProvider, "text", "file")
	// Go 源码 schema 源:直接以用户已有的 ent schema / gorm model 目录为输入。
	Default.RegisterProvider(entSchemaProvider, "ent")
	Default.RegisterProvider(gormSchemaProvider, "gorm")
}

func mysqlProvider(dsn string) (*Driver, error) {
	db, err := sql.Open(dialect.MySQL, dsn)
	if err != nil {
		return nil, err
	}

	drv, err := atlasmysql.Open(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	// dsn example: root:pass@tcp(localhost:3308)/test?parseTime=True
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Closer:     db,
		Inspector:  drv,
		Dialect:    dialect.MySQL,
		SchemaName: cfg.DBName,
	}, nil
}

func postgresProvider(dsn string) (*Driver, error) {
	dsn = "postgres://" + dsn
	db, err := sql.Open(dialect.Postgres, dsn)
	if err != nil {
		return nil, err
	}

	drv, err := postgres.Open(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	// dsn example: postgresql://user:pass@localhost:5432/atlas?search_path=some_schema
	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	schemaName := "public"
	if s := parsed.Query().Get("search_path"); s != "" {
		schemaName = s
	}
	return &Driver{
		Closer:     db,
		Inspector:  drv,
		Dialect:    dialect.Postgres,
		SchemaName: schemaName,
	}, nil
}

func textProvider(dsn string) (*Driver, error) {
	return &Driver{
		SchemaName: dsn,
		Dialect:    "text",
	}, nil
}
