package frontend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindOpenAPIFiles(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("openapi: 3.0.3"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("cmd/server/assets/openapi.yaml")
	write("openapi.yml")
	write("sub/OpenAPI.YAML") // 大小写不敏感
	write("other.txt")        // 非目标文件
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join("node_modules", "openapi.yaml")) // 应被跳过

	files, err := findOpenAPIFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("应命中 3 个(忽略 node_modules 与大小写),得到 %v", files)
	}

	if _, err := findOpenAPIFiles(filepath.Join(root, "missing-dir")); err != nil {
		t.Fatalf("不存在的根目录不应报错(WalkDir 容错): %v", err)
	}
}

func TestExpandErrorI18N(t *testing.T) {
	dir := t.TempDir()
	write := func(name string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	a := write("user_errors_i18n.zh.json")
	b := write("role_errors_i18n.en.json")
	other := write("unrelated.json")

	files := expandErrorI18N([]string{dir, a, other, filepath.Join(dir, "missing.json")})
	if len(files) != 4 { // 目录展开 2 个 + 显式文件 a 与 other 原样保留(重复项由合并阶段先到先得)
		t.Fatalf("目录应展开词表、显式文件原样保留: %v", files)
	}
	has := func(want string) bool {
		for _, f := range files {
			if f == want {
				return true
			}
		}
		return false
	}
	if !has(a) || !has(b) || !has(other) {
		t.Fatalf("展开结果缺项: %v", files)
	}
}
