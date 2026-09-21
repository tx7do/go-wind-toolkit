package gorm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// importerFixtureDDL 走 rawsql 驱动:gorm 直接把 DDL 当库表读,用例因此不需要任何
// 数据库实例。原用例连的是硬编码口令的 Postgres,且对结果零断言,连不上也恒绿。
var importerFixtureDDL = `CREATE TABLE users (
  id bigint NOT NULL AUTO_INCREMENT,
  name varchar(100) NOT NULL,
  email varchar(255) NULL,
  PRIMARY KEY (id)
);`

func TestImporter_FromDDLText_WritesSchemaAndDao(t *testing.T) {
	root := t.TempDir()
	schemaPath := filepath.Join(root, "schema") + string(os.PathSeparator)
	daoPath := filepath.Join(root, "dao") + string(os.PathSeparator)
	drv := "postgres" // SQL 文本分支不参与连接,仅满足参数契约

	if err := Importer(context.Background(), &drv, &importerFixtureDDL, &schemaPath, &daoPath, nil, nil); err != nil {
		t.Fatalf("Importer() error = %v", err)
	}

	model := readFile(t, filepath.Join(root, "schema", "users.gen.go"))
	for _, want := range []string{
		"package schema",
		`const TableNameUser = "users"`,
		"type User struct",
		"column:id",
		"column:name",
		"column:email",
		// FieldNullable: true 的契约所在:可空列必须是指针类型。
		"*string",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("schema/users.gen.go 缺少 %q\n%s", want, model)
		}
	}

	for _, f := range []string{"dao/gen.go", "dao/users.gen.go"} {
		q := readFile(t, filepath.Join(root, f))
		if !strings.Contains(q, "package dao") {
			t.Errorf("%s 不是 dao 包:\n%s", f, q)
		}
	}
	if !strings.Contains(readFile(t, filepath.Join(root, "dao", "users.gen.go")), "schema.User") {
		t.Error("dao 未引用生成的模型 schema.User")
	}
}

// 连接串分支必须把连不上库显式报错,而不是生成一套空表。
func TestImporter_UnreachableDatabaseFailsLoudly(t *testing.T) {
	root := t.TempDir()
	schemaPath := filepath.Join(root, "schema") + string(os.PathSeparator)
	daoPath := filepath.Join(root, "dao") + string(os.PathSeparator)
	drv := "postgres"
	dsn := "postgres://postgres@127.0.0.1:1/example?sslmode=disable"

	if err := Importer(context.Background(), &drv, &dsn, &schemaPath, &daoPath, nil, nil); err == nil {
		t.Fatal("连不上数据库时 Importer 必须返回错误")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
