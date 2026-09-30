package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
)

// fixture 直接复用 gowind 的 schemasource 测试数据：uiapp/go.mod 已有
// replace github.com/tx7do/go-wind-toolkit/gowind => ../gowind，同仓相对布局是既有前提。
func entFixture(t *testing.T) string {
	return fixture(t, filepath.Join("..", "gowind", "internal", "schemasource", "testdata", "entschema"))
}

func gormFixture(t *testing.T) string {
	return fixture(t, filepath.Join("..", "gowind", "internal", "schemasource", "testdata", "gormmodels"))
}

func fixture(t *testing.T, dir string) string {
	t.Helper()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Skipf("fixture 路径不可用: %v", err)
	}
	return abs
}

func TestPlanGoSchemaTables_Ent(t *testing.T) {
	names, err := planGoSchemaTables(context.Background(), "ent://"+entFixture(t), "ent")
	if err != nil {
		t.Fatalf("ent:// 预览失败: %v", err)
	}
	t.Logf("ent tables = %v", names)
	if len(names) != 4 {
		t.Fatalf("期望 4 张表，实际 %d: %v", len(names), names)
	}
}

func TestPlanGoSchemaTables_Gorm(t *testing.T) {
	names, err := planGoSchemaTables(context.Background(), "gorm://"+gormFixture(t), "gorm")
	if err != nil {
		t.Fatalf("gorm:// 预览失败: %v", err)
	}
	t.Logf("gorm tables = %v", names)
	if len(names) == 0 {
		t.Fatal("gorm:// 预览没有解析到表")
	}
}

func TestImportGoSchemaTables_RejectsMismatch(t *testing.T) {
	a := &App{generator: generator.NewGenerator()}

	cases := []struct {
		name    string
		source  string
		ormType string
	}{
		{"scheme 与 ORM 不匹配", "ent://" + entFixture(t), "gorm"},
		{"反向不匹配", "gorm://" + gormFixture(t), "ent"},
		{"不支持的 scheme", "mysql://root@tcp(127.0.0.1)/db", "ent"},
		{"目录不存在", "ent:///no/such/schema/dir", "ent"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 走到成功分支会触发 EventsEmit(nil ctx)，测试里必须不出现返回值以外的 panic。
			if msg := a.ImportGoSchemaTables(c.source, c.ormType); msg == "" {
				t.Fatalf("应当拒绝，但返回了成功")
			}
			if opts := a.generator.GetOptions(); len(opts) != 0 {
				t.Fatalf("拒绝时不应写入表选项，实际 %d 个", len(opts))
			}
		})
	}
}
