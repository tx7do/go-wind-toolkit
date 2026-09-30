package schemasource

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/schema"
)

func TestGoSourceProviders(t *testing.T) {
	for _, tt := range []struct {
		dsn     string
		dialect string
		table   string
	}{
		{dsn: "ent://testdata/entschema", dialect: DialectEntSchema, table: "users"},
		{dsn: "gorm://testdata/gormmodels", dialect: DialectGormSchema, table: "blog_posts"},
	} {
		t.Run(tt.dsn, func(t *testing.T) {
			drv, err := Default.Open(tt.dsn)
			if err != nil {
				t.Fatalf("Open(%q): %v", tt.dsn, err)
			}
			defer drv.Close()
			if drv.Dialect != tt.dialect {
				t.Errorf("dialect = %q; want %q", drv.Dialect, tt.dialect)
			}

			s, err := drv.InspectSchema(context.Background(), drv.SchemaName, nil)
			if err != nil {
				t.Fatalf("InspectSchema: %v", err)
			}
			if len(s.Tables) == 0 {
				t.Fatal("inspected schema has no tables")
			}

			// include 过滤
			s2, err := drv.InspectSchema(context.Background(), drv.SchemaName,
				&schema.InspectOptions{Tables: []string{tt.table}})
			if err != nil {
				t.Fatalf("InspectSchema filtered: %v", err)
			}
			if len(s2.Tables) != 1 || s2.Tables[0].Name != tt.table {
				t.Errorf("filtered tables = %v; want [%s]", s2.Tables, tt.table)
			}

			if _, err := drv.InspectRealm(context.Background(), nil); err == nil {
				t.Error("InspectRealm should fail for go source schemas")
			}
		})
	}
}

func TestGoSourceProvider_Errors(t *testing.T) {
	if _, err := Default.Open("ent://testdata/does-not-exist"); err == nil {
		t.Error("expected error for missing ent schema dir")
	}
	if _, err := Default.Open("gorm://testdata/does-not-exist"); err == nil {
		t.Error("expected error for missing gorm model dir")
	}
}
