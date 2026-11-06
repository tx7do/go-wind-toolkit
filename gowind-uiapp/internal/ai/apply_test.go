package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newPlanTree 构造一份预览树: a/b.txt、c.txt 两个文件。
func newPlanTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a/b.txt", "plan-b")
	write("c.txt", "plan-c")
	return dir
}

func TestWalkPlanFiles(t *testing.T) {
	planDir := newPlanTree(t)
	projectRoot := t.TempDir()
	// 项目里已有 c.txt → 覆盖;a/b.txt 不存在 → 新增
	if err := os.WriteFile(filepath.Join(projectRoot, "c.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := WalkPlanFiles(planDir, projectRoot)
	if err != nil {
		t.Fatalf("遍历失败: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("应有 2 个文件,得到 %d: %+v", len(files), files)
	}
	if files[0].Path != "a/b.txt" || !files[0].IsNew {
		t.Fatalf("a/b.txt 应排前且为新增: %+v", files[0])
	}
	if files[1].Path != "c.txt" || files[1].IsNew {
		t.Fatalf("c.txt 应为覆盖: %+v", files[1])
	}
}

func TestCopyTree(t *testing.T) {
	planDir := newPlanTree(t)
	projectRoot := t.TempDir()
	// 预置同名旧文件,验证覆盖;预置多余文件,验证不被清除(拷贝不是同步)
	if err := os.WriteFile(filepath.Join(projectRoot, "c.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyTree(projectRoot, planDir); err != nil {
		t.Fatalf("拷贝失败: %v", err)
	}
	for rel, want := range map[string]string{
		"a/b.txt":  "plan-b",
		"c.txt":    "plan-c",
		"keep.txt": "keep",
	} {
		data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s 应存在: %v", rel, err)
		}
		if string(data) != want {
			t.Fatalf("%s 内容不符: %q", rel, data)
		}
	}
}

func TestReadPlanFile(t *testing.T) {
	planDir := newPlanTree(t)

	content, err := ReadPlanFile(planDir, "a/b.txt")
	if err != nil || content != "plan-b" {
		t.Fatalf("读取失败: %v, %q", err, content)
	}

	for _, bad := range []string{
		"../escape.txt",          // 越出计划目录
		filepath.Join("a", ".."), // 目录
		"",                       // 空
		"C:/Windows/system.ini",  // 绝对路径
	} {
		if _, err := ReadPlanFile(planDir, bad); err == nil {
			t.Fatalf("非法路径 %q 不应通过", bad)
		}
	}
}

func TestNewPlanID(t *testing.T) {
	id1, id2 := NewPlanID(), NewPlanID()
	if id1 == "" || id2 == "" {
		t.Fatal("计划 ID 不应为空")
	}
	if id1 == id2 {
		t.Fatal("两次生成的计划 ID 不应相同")
	}
	if strings.ContainsAny(id1, "/\\ ") {
		t.Fatalf("计划 ID 不应含路径/空白字符: %q", id1)
	}
}
