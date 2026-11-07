package schemasource

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"ariga.io/atlas/sql/schema"

	// 直接用驱动建库,provider 再从文件内省——完整回路。
	_ "modernc.org/sqlite"
)

func TestSQLiteProvider_IntrospectsSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		score REAL DEFAULT 0,
		active BOOLEAN NOT NULL DEFAULT 1,
		avatar BLOB
	)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	drv, err := Default.Open("sqlite://" + dbPath)
	if err != nil {
		t.Fatalf("打开 sqlite 数据源失败: %v", err)
	}
	defer drv.Close()

	s, err := drv.InspectSchema(context.Background(), drv.SchemaName, nil)
	if err != nil {
		t.Fatalf("内省失败: %v", err)
	}
	if len(s.Tables) != 1 || s.Tables[0].Name != "users" {
		t.Fatalf("应内省出 users 表,得到 %+v", s.Tables)
	}

	kinds := map[string]string{}
	for _, c := range s.Tables[0].Columns {
		switch c.Type.Type.(type) {
		case *schema.IntegerType:
			kinds[c.Name] = "integer"
		case *schema.StringType:
			kinds[c.Name] = "string"
		case *schema.FloatType:
			kinds[c.Name] = "float"
		case *schema.BoolType:
			kinds[c.Name] = "bool"
		case *schema.BinaryType:
			kinds[c.Name] = "binary"
		}
	}
	for name, want := range map[string]string{
		"id": "integer", "name": "string", "score": "float", "active": "bool", "avatar": "binary",
	} {
		if kinds[name] != want {
			t.Errorf("%s 应为 %s,得到 %s(%v)", name, want, kinds[name], kinds[name])
		}
	}

	// 不存在的文件必须报错,不能静默建空库
	if _, err := Default.Open("sqlite://" + filepath.Join(t.TempDir(), "missing.db")); err == nil {
		t.Fatal("不存在的 sqlite 文件应报错")
	}
}
