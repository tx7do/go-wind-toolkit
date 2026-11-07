package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCopyLocalTemplate 验证本地模板目录的拷贝与模块名替换:
// 模板 module 名被替换为目标名,.git 被忽略。
func TestCopyLocalTemplate(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "go.mod"), []byte("module github.com/example/my-template\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "app", "main.go"), []byte("package main\n\nimport _ \"github.com/example/my-template/internal\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "newproj")
	if err := CopyLocalTemplate(src, dst, "newproj", []string{".git"}); err != nil {
		t.Fatalf("CopyLocalTemplate: %v", err)
	}

	main, err := os.ReadFile(filepath.Join(dst, "app", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(main), "my-template") {
		t.Fatalf("module 名应被替换: %s", main)
	}
	if !strings.Contains(string(main), `"newproj/internal"`) {
		t.Fatalf("替换结果不对: %s", main)
	}
	if _, err := os.Stat(filepath.Join(dst, ".git")); !os.IsNotExist(err) {
		t.Fatalf(".git 应被忽略,实际: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "go.mod")); err != nil {
		t.Fatalf("go.mod 应被拷贝: %v", err)
	}
}

// TestCopyLocalTemplate_Errors 缺 go.mod、路径不存在、非目录三类入参错误。
func TestCopyLocalTemplate_Errors(t *testing.T) {
	noMod := t.TempDir()
	if err := CopyLocalTemplate(noMod, filepath.Join(t.TempDir(), "x"), "x", nil); err == nil {
		t.Fatal("缺 go.mod 的模板应报错")
	}

	file := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyLocalTemplate(file, filepath.Join(t.TempDir(), "x"), "x", nil); err == nil {
		t.Fatal("非目录模板应报错")
	}

	if err := CopyLocalTemplate(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "x"), "x", nil); err == nil {
		t.Fatal("不存在的模板应报错")
	}
}
