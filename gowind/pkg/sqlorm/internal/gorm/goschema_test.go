package gorm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImporterFromGoSchema 验证 gorm:// 源码源的 DAO 回转:
// 在临时 Go module 内生成 DAO、改写临时模型包 import、且绝不覆盖用户手写模型。
func TestImporterFromGoSchema(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "internal", "schemasource", "testdata", "gormmodels"))
	if err != nil {
		t.Fatal(err)
	}

	mod := t.TempDir()
	if err := os.WriteFile(filepath.Join(mod, "go.mod"),
		[]byte("module example.com/shop\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(mod, "internal", "models")
	daoPath := filepath.Join(mod, "internal", "dao")
	if err := os.MkdirAll(schemaPath, 0o755); err != nil {
		t.Fatal(err)
	}

	// 用户已有手写模型:文件必须保持不变,且同名类型不得被再次补齐(避免包内重复声明)。
	handwritten := "// hand written, do not touch\npackage models\n\ntype User struct {\n\tID   uint\n\tName string\n}\n"
	userModel := filepath.Join(schemaPath, "user.go")
	if err := os.WriteFile(userModel, []byte(handwritten), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ImporterFromGoSchema(context.Background(), "gorm://"+fixture,
		&schemaPath, &daoPath, []string{"users", "pets"}); err != nil {
		t.Fatalf("ImporterFromGoSchema: %v", err)
	}

	// 临时生成目录必须被清理
	if _, err := os.Stat(filepath.Join(daoPath, gensrcDirName)); !os.IsNotExist(err) {
		t.Error("temp gensrc dir was not removed")
	}

	// DAO 生成成功且 import 指向真实模型包
	daoFiles, err := os.ReadDir(daoPath)
	if err != nil {
		t.Fatal(err)
	}
	var generated int
	var modelsReferenced bool
	for _, f := range daoFiles {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") {
			continue
		}
		generated++
		content, err := os.ReadFile(filepath.Join(daoPath, f.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), gensrcDirName) {
			t.Errorf("dao file %s still references the temp package path", f.Name())
		}
		if strings.Contains(string(content), "example.com/shop/internal/models") {
			modelsReferenced = true
		}
	}
	if generated == 0 {
		t.Fatal("no DAO files generated")
	}
	if !modelsReferenced {
		t.Error("no DAO file references the real model package (import rewrite failed)")
	}

	// 手写模型未被覆盖;类型冲突的模型被跳过,缺失的模型被补齐(gen 产物命名为 <table>.gen.go)
	data, err := os.ReadFile(userModel)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != handwritten {
		t.Error("user model was overwritten")
	}
	if _, err := os.Stat(filepath.Join(schemaPath, "users.gen.go")); !os.IsNotExist(err) {
		t.Error("users.gen.go declares User which already exists in hand written model; should be skipped")
	}
	if _, err := os.Stat(filepath.Join(schemaPath, "pets.gen.go")); err != nil {
		t.Errorf("missing model should be filled in: %v", err)
	}
}

func TestImporterFromGoSchema_OutsideModule(t *testing.T) {
	dir := t.TempDir() // 无 go.mod
	schemaPath := filepath.Join(dir, "models")
	daoPath := filepath.Join(dir, "dao")
	err := ImporterFromGoSchema(context.Background(), "gorm://"+dir, &schemaPath, &daoPath, nil)
	if err == nil {
		t.Fatal("expected error when target is outside a Go module")
	}
}

func TestLocateGoModuleAndImportPath(t *testing.T) {
	mod := t.TempDir()
	if err := os.WriteFile(filepath.Join(mod, "go.mod"),
		[]byte("module github.com/acme/api\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(mod, "internal", "dao")
	root, path, err := locateGoModule(sub)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(root) != filepath.Base(mod) || path != "github.com/acme/api" {
		t.Fatalf("locateGoModule = %q, %q", root, path)
	}
	if got := importPathOf(root, path, sub); got != "github.com/acme/api/internal/dao" {
		t.Errorf("importPathOf = %q", got)
	}
	if got := importPathOf(root, path, t.TempDir()); got != "" {
		t.Errorf("importPathOf outside module = %q; want empty", got)
	}
}
