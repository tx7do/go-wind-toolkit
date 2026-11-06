package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newScanProject 构造一个最小可审查项目树:
// app/user 下一对 service Go 文件(含被排除的测试与生成物)、api/protos 一个 proto、
// 外加若干应被排除的目录与文件。
func newScanProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("app/user/service/user.go", "package service\n")
	write("app/user/service/user_test.go", "package service\n")
	write("app/user/service/user.pb.go", "package service\n")
	write("app/user/service/user_grpc.pb.go", "package service\n")
	write("api/protos/user.proto", "syntax = \"proto3\";\n")
	write("api/protos/generated/gen.proto", "syntax = \"proto3\";\n")
	write("other/readme.txt", "hello\n")
	write(".gowind/ai/ddl.sql", testDDL)
	write("vendor/x/x.go", "package x\n")

	return root
}

func TestScanProjectCode(t *testing.T) {
	root := newScanProject(t)

	res, err := ScanProjectCode(root)
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	if !res.Success {
		t.Fatal("res.Success 应为 true")
	}

	var got []string
	for _, f := range res.Files {
		got = append(got, f.Path)
	}
	want := []string{"api/protos/generated/gen.proto", "api/protos/user.proto", "app/user/service/user.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("扫描结果不符:\n got  %v\n want %v", got, want)
	}
}

func TestScanProjectCode_FileSizeCap(t *testing.T) {
	root := t.TempDir()
	big := strings.Repeat("a", maxReviewFileBytes+1)
	path := filepath.Join(root, "app", "user", "big.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ScanProjectCode(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 0 {
		t.Fatalf("超限文件不应入选,得到 %v", res.Files)
	}
	if len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "已跳过") {
		t.Fatalf("应给出跳过告警,得到 %v", res.Warnings)
	}
}

func TestReadProjectFiles(t *testing.T) {
	root := newScanProject(t)

	contents, warnings, err := ReadProjectFiles(root, []string{
		"app/user/service/user.go",
		"api/protos/user.proto",
		// 不可审查的路径:各自记 warning 后跳过,不阻断其余文件。
		"../outside.go",              // 越出项目根
		".gowind/ai/ddl.sql",         // 不在可审查范围
		"app/user/service/absent.go", // 不存在
	})
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if len(warnings) != 3 {
		t.Fatalf("应有 3 条告警,得到 %v", warnings)
	}
	if len(contents) != 2 {
		t.Fatalf("应读到 2 个文件,得到 %d: %v", len(contents), contents)
	}
	if _, ok := contents["app/user/service/user.go"]; !ok {
		t.Fatalf("缺 user.go: %v", contents)
	}
}

func TestReadProjectFiles_Empty(t *testing.T) {
	root := newScanProject(t)
	if _, _, err := ReadProjectFiles(root, nil); err == nil {
		t.Fatal("空选择应报错")
	}
}

func TestReadProjectFiles_AllUnreadable(t *testing.T) {
	root := newScanProject(t)
	if _, _, err := ReadProjectFiles(root, []string{"other/readme.txt"}); err == nil {
		t.Fatal("全部不可读时应报错")
	}
}

func TestResolveProjectFile_RejectsUnsafe(t *testing.T) {
	root := t.TempDir()
	for _, bad := range []string{
		"..\\..\\secret",         // 越出项目根
		"C:/Windows/system.ini",  // 绝对路径
		"/etc/passwd",            // 绝对路径(POSIX 形式)
		"app/user/service/x.env", // 非可审查后缀
		"",                       // 空
	} {
		if _, err := resolveProjectFile(root, bad); err == nil {
			t.Fatalf("路径 %q 不应通过校验", bad)
		}
	}
}
