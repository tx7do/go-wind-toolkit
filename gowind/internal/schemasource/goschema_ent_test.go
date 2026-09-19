package schemasource

import (
	"strings"
	"testing"

	"ariga.io/atlas/sql/schema"
)

func parseEntFixture(t *testing.T) *ParsedSchema {
	t.Helper()
	ps, err := ParseEntSchemaDir("testdata/entschema")
	if err != nil {
		t.Fatalf("ParseEntSchemaDir: %v", err)
	}
	return ps
}

func tableByName(t *testing.T, ps *ParsedSchema, name string) *schema.Table {
	t.Helper()
	for _, tb := range ps.Tables {
		if tb.Name == name {
			return tb
		}
	}
	t.Fatalf("table %q not found in %v", name, tableNames(ps))
	return nil
}

func tableNames(ps *ParsedSchema) []string {
	var out []string
	for _, tb := range ps.Tables {
		out = append(out, tb.Name)
	}
	return out
}

func columnOfTable(t *testing.T, tb *schema.Table, name string) *schema.Column {
	t.Helper()
	c := columnOf(tb, name)
	if c == nil {
		t.Fatalf("table %q has no column %q", tb.Name, name)
	}
	return c
}

func assertRaw(t *testing.T, c *schema.Column, want string) {
	t.Helper()
	if c.Type.Raw != want {
		t.Errorf("column %q raw type = %q; want %q", c.Name, c.Type.Raw, want)
	}
}

func TestParseEntSchemaDir_Tables(t *testing.T) {
	ps := parseEntFixture(t)

	want := map[string]bool{
		"users": true, "pets": true, "groups": true, "member_cards": true,
		"user_groups": true, "user_subordinates": true,
	}
	if len(ps.Tables) != len(want) {
		t.Fatalf("tables = %v; want %v", tableNames(ps), want)
	}
	for _, tb := range ps.Tables {
		if !want[tb.Name] {
			t.Errorf("unexpected table %q", tb.Name)
		}
	}
}

func TestParseEntSchemaDir_Fields(t *testing.T) {
	ps := parseEntFixture(t)
	users := tableByName(t, ps, "users")

	id := columnOfTable(t, users, "id")
	assertRaw(t, id, "bigint")
	if !ps.AutoIncrement[incKey("users", "id")] {
		t.Error("users.id should be marked auto-increment")
	}
	if users.PrimaryKey == nil || len(users.PrimaryKey.Parts) != 1 || users.PrimaryKey.Parts[0].C != id {
		t.Error("users primary key should be (id)")
	}

	name := columnOfTable(t, users, "name")
	assertRaw(t, name, "varchar(64)")
	if got := commentOf(name); got != "用户昵称" {
		t.Errorf("users.name comment = %q; want 用户昵称", got)
	}

	assertRaw(t, columnOfTable(t, users, "age"), "bigint")
	if !columnOfTable(t, users, "age").Type.Null {
		t.Error("users.age (Optional) should be nullable")
	}
	if !columnOfTable(t, users, "email").Type.Null {
		t.Error("users.email (Optional+Nillable) should be nullable")
	}
	assertRaw(t, columnOfTable(t, users, "role"), "enum('admin','user')")

	// StorageKey 重命名
	if columnOf(users, "old_col") == nil {
		t.Error(`users should expose storage-key column "old_col"`)
	}
	if columnOf(users, "legacy_column") != nil {
		t.Error(`users should not keep the field name for a StorageKey'd column`)
	}

	assertRaw(t, columnOfTable(t, tableByName(t, ps, "pets"), "born_at"), "datetime")
}

func TestParseEntSchemaDir_Relations(t *testing.T) {
	ps := parseEntFixture(t)

	// O2M: pets.owner_id → users
	fk := findFK(t, tableByName(t, ps, "pets"), "owner_id")
	if fk.RefTable.Name != "users" {
		t.Errorf("pets.owner_id FK refs %q; want users", fk.RefTable.Name)
	}

	// O2O: member_cards.owner_id → users (entsql.Annotation 表名覆盖)
	fk = findFK(t, tableByName(t, ps, "member_cards"), "owner_id")
	if fk.RefTable.Name != "users" {
		t.Errorf("member_cards.owner_id FK refs %q; want users", fk.RefTable.Name)
	}

	// M2M: user_groups 中间表
	jt := tableByName(t, ps, "user_groups")
	if !IsJoinTable(jt) {
		t.Error("user_groups should satisfy IsJoinTable")
	}
	columnOfTable(t, jt, "user_id")
	columnOfTable(t, jt, "group_id")

	// 自引用 M2M: user_subordinates 第二列以边标签命名
	jt = tableByName(t, ps, "user_subordinates")
	if !IsJoinTable(jt) {
		t.Error("user_subordinates should satisfy IsJoinTable")
	}
	columnOfTable(t, jt, "user_id")
	columnOfTable(t, jt, "subordinate_id")
}

func TestParseEntSchemaDir_Warnings(t *testing.T) {
	ps := parseEntFixture(t)
	var found bool
	for _, w := range ps.Warnings {
		if strings.Contains(w, "mixin") && strings.Contains(w, "Group") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a mixin warning for Group, got %v", ps.Warnings)
	}
}

func findFK(t *testing.T, tb *schema.Table, column string) *schema.ForeignKey {
	t.Helper()
	for _, fk := range tb.ForeignKeys {
		if len(fk.Columns) == 1 && fk.Columns[0].Name == column {
			return fk
		}
	}
	t.Fatalf("table %q has no FK on column %q", tb.Name, column)
	return nil
}

func commentOf(c *schema.Column) string {
	for _, a := range c.Attrs {
		if cm, ok := a.(*schema.Comment); ok {
			return cm.Text
		}
	}
	return ""
}
