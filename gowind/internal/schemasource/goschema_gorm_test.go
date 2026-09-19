package schemasource

import (
	"strings"
	"testing"
)

func parseGormFixture(t *testing.T) *ParsedSchema {
	t.Helper()
	ps, err := ParseGormSchemaDir("testdata/gormmodels")
	if err != nil {
		t.Fatalf("ParseGormSchemaDir: %v", err)
	}
	return ps
}

func TestParseGormSchemaDir_Tables(t *testing.T) {
	ps := parseGormFixture(t)

	want := map[string]bool{
		"users": true, "profiles": true, "pets": true, "groups": true,
		"blog_posts": true, "comments": true, "user_groups": true,
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

func TestParseGormSchemaDir_NamingAndTags(t *testing.T) {
	ps := parseGormFixture(t)
	users := tableByName(t, ps, "users")

	// gorm.Model 注入 id/created_at/updated_at/deleted_at
	id := columnOfTable(t, users, "id")
	assertRaw(t, id, "bigint unsigned")
	if users.PrimaryKey == nil || len(users.PrimaryKey.Parts) != 1 || users.PrimaryKey.Parts[0].C != id {
		t.Error("users primary key should be (id)")
	}
	if !ps.AutoIncrement[incKey("users", "id")] {
		t.Error("users.id should be marked auto-increment")
	}
	columnOfTable(t, users, "created_at")
	columnOfTable(t, users, "updated_at")
	if !columnOfTable(t, users, "deleted_at").Type.Null {
		t.Error("users.deleted_at should be nullable (soft delete)")
	}

	columnOfTable(t, users, "email_addr")           // column tag 重命名
	assertRaw(t, columnOfTable(t, users, "name"), "varchar(64)") // size tag
	if got := commentOf(columnOfTable(t, users, "name")); got != "用户名" {
		t.Errorf("users.name comment = %q", got)
	}
	if columnOf(users, "secret") != nil {
		t.Error(`users.secret (gorm:"-") should be skipped`)
	}
	assertRaw(t, columnOfTable(t, users, "tags"), "json") // []string → json
	assertRaw(t, columnOfTable(t, users, "meta"), "json") // map → json
	if !columnOfTable(t, users, "active").Type.Null {
		t.Error("users.active (*bool) should be nullable")
	}

	posts := tableByName(t, ps, "blog_posts") // TableName() 覆盖
	assertRaw(t, columnOfTable(t, posts, "uuid"), "char(36)")
	assertRaw(t, columnOfTable(t, posts, "body"), "text") // type tag
	assertRaw(t, columnOfTable(t, posts, "status"), "varchar(255)")
	assertRaw(t, columnOfTable(t, posts, "published"), "datetime")
	if !columnOfTable(t, posts, "published").Type.Null {
		t.Error("blog_posts.published (*time.Time) should be nullable")
	}
	// Base 匿名嵌入展开
	columnOfTable(t, posts, "created_at")
	if got := commentOf(columnOfTable(t, posts, "title")); got != "标题" {
		t.Errorf("blog_posts.title comment = %q", got)
	}
}

func TestParseGormSchemaDir_Defaults(t *testing.T) {
	ps := parseGormFixture(t)
	posts := tableByName(t, ps, "blog_posts")

	if got := defaultExpr(columnOfTable(t, posts, "status").Default); got != "'draft'" {
		t.Errorf("blog_posts.status default = %q; want 'draft'", got)
	}
	if got := defaultExpr(columnOfTable(t, posts, "views").Default); got != "0" {
		t.Errorf("blog_posts.views default = %q; want 0", got)
	}
}

func TestParseGormSchemaDir_Associations(t *testing.T) {
	ps := parseGormFixture(t)

	// belongs-to: pets.owner_id → users (tag foreignKey:OwnerID)
	fk := findFK(t, tableByName(t, ps, "pets"), "owner_id")
	if fk.RefTable.Name != "users" {
		t.Errorf("pets.owner_id FK refs %q; want users", fk.RefTable.Name)
	}

	// has-many: comments.post_id → blog_posts
	fk = findFK(t, tableByName(t, ps, "comments"), "post_id")
	if fk.RefTable.Name != "blog_posts" {
		t.Errorf("comments.post_id FK refs %q; want blog_posts", fk.RefTable.Name)
	}

	// has-one: profiles.user_id → users
	fk = findFK(t, tableByName(t, ps, "profiles"), "user_id")
	if fk.RefTable.Name != "users" {
		t.Errorf("profiles.user_id FK refs %q; want users", fk.RefTable.Name)
	}

	// belongs-to 显式外键列: blog_posts.author_id → users
	fk = findFK(t, tableByName(t, ps, "blog_posts"), "author_id")
	if fk.RefTable.Name != "users" {
		t.Errorf("blog_posts.author_id FK refs %q; want users", fk.RefTable.Name)
	}

	// many2many: user_groups 中间表(双向声明只生成一次)
	jt := tableByName(t, ps, "user_groups")
	if !IsJoinTable(jt) {
		t.Error("user_groups should satisfy IsJoinTable")
	}
	columnOfTable(t, jt, "group_id")
	columnOfTable(t, jt, "user_id")
}

func TestParseGormSchemaDir_BaseNotModel(t *testing.T) {
	ps := parseGormFixture(t)
	for _, tb := range ps.Tables {
		if tb.Name == "bases" {
			t.Error("Base 是被嵌入的基类型,不应生成模型表")
		}
	}
}

func TestParseGormSchemaDir_Warnings(t *testing.T) {
	ps := parseGormFixture(t)
	var slice, mapped bool
	for _, w := range ps.Warnings {
		switch {
		case strings.Contains(w, "Tags") && strings.Contains(w, "slice"):
			slice = true
		case strings.Contains(w, "Meta") && strings.Contains(w, "map"):
			mapped = true
		}
	}
	if !slice || !mapped {
		t.Errorf("expected slice+map warnings, got %v", ps.Warnings)
	}
}
