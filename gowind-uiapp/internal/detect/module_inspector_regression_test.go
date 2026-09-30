package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsRealMainModule(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want bool
	}{
		{"real module", `{"Path":"example.com/app","Main":true,"Dir":"/x","GoMod":"/x/go.mod","GoVersion":"1.27.1"}`, true},
		{"fake no-gomod (go1.27)", `{"Path":"command-line-arguments","Main":true,"GoVersion":"1.27.1"}`, false},
		{"empty goMod", `{"Path":"example.com/app","Main":true,"GoMod":""}`, false},
		{"empty path", `{"Path":"","GoMod":"/x/go.mod"}`, false},
		{"not json", `go: go.mod file not found`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isRealMainModule([]byte(c.out)); got != c.want {
				t.Fatalf("isRealMainModule(%q) = %v, want %v", c.out, got, c.want)
			}
		})
	}
}

// 在模块的任意子目录探测，应向上定位到真正的模块根，而不是返回 command-line-arguments。
func TestNewModuleInspectorFromGo_LocatesModuleRootFromSubdir(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "go.mod")); err != nil {
		t.Skip("requires go toolchain/module context")
	}

	modDir := t.TempDir()
	writeGoMod(t, modDir, "example.com/detecttest")
	subDir := filepath.Join(modDir, "app", "identity", "service")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	inspector, err := NewModuleInspectorFromGo(subDir)
	if err != nil {
		t.Fatalf("detect from subdir failed: %v", err)
	}
	if inspector.ModPath == "command-line-arguments" {
		t.Fatalf("got fake module name command-line-arguments")
	}
	if inspector.ModPath != "example.com/detecttest" {
		t.Fatalf("ModPath = %q, want example.com/detecttest", inspector.ModPath)
	}
	// macOS 上 t.TempDir 的 /var 是 /private/var 的符号链接，go 返回已解析路径，比较前统一解析。
	gotRoot, e := filepath.EvalSymlinks(inspector.Root)
	if e != nil {
		gotRoot = filepath.Clean(inspector.Root)
	}
	wantRoot, e := filepath.EvalSymlinks(modDir)
	if e != nil {
		wantRoot = filepath.Clean(modDir)
	}
	if gotRoot != wantRoot {
		t.Fatalf("Root = %q, want %q", inspector.Root, modDir)
	}
}

func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "module " + module + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
