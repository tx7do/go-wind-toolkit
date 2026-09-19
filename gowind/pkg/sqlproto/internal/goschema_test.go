package internal

import (
	"context"
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

// openGoSource 通过进程级 Mux 打开 Go 源码 schema 源,验证 provider→方言→转换器 全链路。
func openGoSource(t *testing.T, dsn string) SchemaConverter {
	t.Helper()
	drv, err := schemasource.Default.Open(dsn)
	if err != nil {
		t.Fatalf("open %q: %v", dsn, err)
	}
	t.Cleanup(func() { _ = drv.Close() })
	c, err := NewConvert(WithSchemaPath(dsn), WithDriver(drv))
	if err != nil {
		t.Fatalf("NewConvert(%q): %v", dsn, err)
	}
	return c
}

func tableDataByName(t *testing.T, tables []*TableData, name string) *TableData {
	t.Helper()
	for _, tb := range tables {
		if tb.Name == name {
			return tb
		}
	}
	t.Fatalf("table %q not found", name)
	return nil
}

func TestGoSchemaConvertEnt(t *testing.T) {
	tables, err := openGoSource(t, "ent://../../../internal/schemasource/testdata/entschema").
		SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("SchemaTables(ent): %v", err)
	}

	names := map[string]bool{}
	for _, tb := range tables {
		names[tb.Name] = true
	}
	for _, want := range []string{"users", "pets", "groups", "member_cards"} {
		if !names[want] {
			t.Errorf("expected table %q in %v", want, names)
		}
	}
	// 中间表应按 join-table 规则被排除
	for _, skip := range []string{"user_groups", "user_subordinates"} {
		if names[skip] {
			t.Errorf("join table %q should be excluded from proto tables", skip)
		}
	}

	users := tableDataByName(t, tables, "users")
	var id, role *FieldData
	for i := range users.Fields {
		switch users.Fields[i].Name {
		case "id":
			id = &users.Fields[i]
		case "role":
			role = &users.Fields[i]
		}
	}
	if id == nil || id.Type != "int64" || !id.IsPrimaryKey {
		t.Errorf("users.id field = %+v", id)
	}
	if role == nil || role.Type != "string" {
		t.Errorf("users.role field = %+v; want proto type string", role)
	}
}

func TestGoSchemaConvertGorm(t *testing.T) {
	tables, err := openGoSource(t, "gorm://../../../internal/schemasource/testdata/gormmodels").
		SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("SchemaTables(gorm): %v", err)
	}

	names := map[string]bool{}
	for _, tb := range tables {
		names[tb.Name] = true
	}
	for _, want := range []string{"users", "pets", "blog_posts", "comments", "profiles", "groups"} {
		if !names[want] {
			t.Errorf("expected table %q in %v", want, names)
		}
	}
	if names["user_groups"] {
		t.Error("join table user_groups should be excluded")
	}

	posts := tableDataByName(t, tables, "blog_posts")
	var uuid, status *FieldData
	for i := range posts.Fields {
		switch posts.Fields[i].Name {
		case "uuid":
			uuid = &posts.Fields[i]
		case "status":
			status = &posts.Fields[i]
		}
	}
	// char(36) → string, enum 风格 varchar → string;不应出现空类型
	if uuid == nil || uuid.Type == "" {
		t.Errorf("blog_posts.uuid field = %+v", uuid)
	}
	if status == nil || status.Type != "string" {
		t.Errorf("blog_posts.status field = %+v", status)
	}
}

func TestGoSchemaConvertIncludeFilter(t *testing.T) {
	drv, err := schemasource.Default.Open("ent://../../../internal/schemasource/testdata/entschema")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = drv.Close() }()
	tables, err := mustConvert(t, drv, []string{"users"}).SchemaTables(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 || tables[0].Name != "users" {
		t.Errorf("included filter: got %v", tableDataByName(t, tables, "users").Name)
	}
}

func mustConvert(t *testing.T, drv *schemasource.Driver, include []string) SchemaConverter {
	t.Helper()
	c, err := NewConvert(WithDriver(drv), WithIncludedTables(include))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
