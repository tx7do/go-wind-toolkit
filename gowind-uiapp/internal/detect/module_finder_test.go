package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindModulesUnder(t *testing.T) {
	root := t.TempDir()

	// 两个顶层模块 + 一个嵌套子模块 + 一个应被跳过的 vendor 模块。
	writeGoMod(t, filepath.Join(root, "modA"), "example.com/modA")
	writeGoMod(t, filepath.Join(root, "nested", "modB"), "example.com/modB")
	writeGoMod(t, filepath.Join(root, "nested", "modB", "inner"), "example.com/inner") // 不应出现
	writeGoMod(t, filepath.Join(root, "vendor", "dep"), "example.com/dep")             // 不应出现
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindModulesUnder(root, DefaultModuleSearchDepth)
	if err != nil {
		t.Fatal(err)
	}

	paths := map[string]bool{}
	for _, c := range got {
		paths[filepath.ToSlash(filepath.Clean(c.RelPath))] = true
	}

	for _, want := range []string{"modA", filepath.ToSlash(filepath.Join("nested", "modB"))} {
		if !paths[want] {
			t.Errorf("missing module %q in %+v", want, got)
		}
	}
	for _, bad := range []string{"nested/modB/inner", "vendor/dep", "."} {
		if paths[bad] {
			t.Errorf("unexpected module %q in %+v", bad, got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want 2: %+v", len(got), got)
	}
}

func TestFindModulesUnder_IncludesRootWhenItIsAModule(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/rootmod")
	got, err := FindModulesUnder(root, DefaultModuleSearchDepth)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ModPath != "example.com/rootmod" {
		t.Fatalf("got %+v, want single rootmod", got)
	}
}

func TestReadModulePath(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"module example.com/a\n\ngo 1.21\n":     "example.com/a",
		"module   example.com/b   // comment\n": "example.com/b",
		"module \"example.com/c\"\ngo 1.21\n":   "example.com/c",
		"// leading\nmodule example.com/d\n":    "example.com/d",
		"package main\n":                        "",
	}
	for content, want := range cases {
		p := filepath.Join(dir, "go.mod")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := readModulePath(p); got != want {
			t.Errorf("readModulePath(%q) = %q, want %q", content, got, want)
		}
	}
}
