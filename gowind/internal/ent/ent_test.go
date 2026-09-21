package ent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewEntCmdDefaults(t *testing.T) {
	e := NewEntCmd("")
	if e.TargetDir != filepath.Clean("internal/data/ent/schema") {
		t.Fatalf("default TargetDir = %q", e.TargetDir)
	}
	if e.Timeout != 2*time.Minute {
		t.Fatalf("default Timeout = %v", e.Timeout)
	}

	// 显式目录按 filepath.Clean 归一。
	e = NewEntCmd("./app/admin/service/./internal/data/ent/schema/../schema")
	if e.TargetDir != filepath.Clean("app/admin/service/internal/data/ent/schema") {
		t.Fatalf("explicit TargetDir = %q", e.TargetDir)
	}
}

func TestRunNewRequiresNames(t *testing.T) {
	e := NewEntCmd(filepath.Join(t.TempDir(), "schema"))
	if err := e.RunNew(context.Background(), nil); err == nil {
		t.Fatal("RunNew without schema names must fail before touching the filesystem")
	}
}

func TestNewEntCmdGoCmdStartsAtSchemaDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "./internal/data/ent/./schema")
	e := NewEntCmd(dir)

	// go run 分支从 schema 目录起向上找模块根;若传空串会退回 os.Getwd(),
	// 落点随调用者的 CWD 漂移。
	if e.goCmd.Dir != e.TargetDir {
		t.Fatalf("goCmd.Dir = %q, want TargetDir %q", e.goCmd.Dir, e.TargetDir)
	}
}

func TestSchemaTargetArgIsAbsolute(t *testing.T) {
	e := NewEntCmd(filepath.Join(t.TempDir(), "internal", "data", "ent", "schema"))

	arg := e.schemaTargetArg()
	want := "--target=" + e.TargetDir
	if arg != want {
		t.Fatalf("schemaTargetArg() = %q, want %q", arg, want)
	}
	if !filepath.IsAbs(strings.TrimPrefix(arg, "--target=")) {
		t.Fatalf("schemaTargetArg() must carry an absolute path, got %q", arg)
	}
}

func TestVerifySchemasCreated(t *testing.T) {
	e := NewEntCmd(filepath.Join(t.TempDir(), "internal", "data", "ent", "schema"))
	if err := os.MkdirAll(e.TargetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// ent 的落盘规则:<schema 目录>/<小写名>.go
	if err := os.WriteFile(filepath.Join(e.TargetDir, "user.go"), []byte("package schema\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := e.verifySchemasCreated([]string{"User"}); err != nil {
		t.Fatalf("existing schema reported missing: %v", err)
	}

	// 只产出到别处(CWD)时,必须显式失败而不是报成功。
	err := e.verifySchemasCreated([]string{"User", "Role"})
	if err == nil {
		t.Fatal("missing Role.go must be reported")
	}
	if !strings.Contains(err.Error(), e.TargetDir) || !strings.Contains(err.Error(), "Role") {
		t.Fatalf("error must name the dir and the missing schema, got: %v", err)
	}
}
