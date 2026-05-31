package gorm

import (
	"context"
	"testing"
)

func TestImporter(t *testing.T) {
	ctx := context.Background()
	drv := "postgres"
	//dsn := "mysql://root:*Abcd123456@tcp(localhost:3306)/example"
	dsn := "postgres://postgres:*Abcd123456@localhost:5432/example?sslmode=disable"
	schemaPath := "./models/"
	daoPath := "./daos/"
	var includeTables []string
	var excludeTables []string
	_ = Importer(ctx, &drv, &dsn, &schemaPath, &daoPath, includeTables, excludeTables)
}
