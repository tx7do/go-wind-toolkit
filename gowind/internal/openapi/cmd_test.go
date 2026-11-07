package openapi

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestBufTemplateManagedAndStrategy 生成的临时 buf.gen.yaml 必须带 managed
// 兜底(很多项目的 proto 无 go_package)与 strategy: all(buf v2 按文件并发
// 调用插件,缺失时只留第一个空文档)——两个踩坑点钉死。
func TestBufTemplateManagedAndStrategy(t *testing.T) {
	tmpDir := t.TempDir()
	tmpOut := filepath.Join(tmpDir, "out")

	tmplBody := buildBufTemplate(tmpOut)

	if !strings.Contains(tmplBody, "managed:") || !strings.Contains(tmplBody, "go_package_prefix") {
		t.Fatalf("模板应包含 managed go_package 兜底:\n%s", tmplBody)
	}
	if !strings.Contains(tmplBody, "strategy: all") {
		t.Fatalf("插件必须 strategy: all(否则 buf 按文件并发只留第一个空文档):\n%s", tmplBody)
	}
	if !strings.Contains(tmplBody, filepath.ToSlash(tmpOut)) {
		t.Fatalf("插件 out 应指向临时输出目录:\n%s", tmplBody)
	}
}

// TestResolveTarget 输出目标解析:缺省 <api>/openapi.yaml;
// --out 为已存在目录时写入其中,否则视为文件路径。
func TestResolveTarget(t *testing.T) {
	apiDir := t.TempDir()

	if got := resolveTarget(apiDir, ""); got != filepath.Join(apiDir, "openapi.yaml") {
		t.Fatalf("缺省目标不对: %s", got)
	}

	outDir := t.TempDir()
	if got := resolveTarget(apiDir, outDir); got != filepath.Join(outDir, "openapi.yaml") {
		t.Fatalf("目录目标应追加文件名: %s", got)
	}

	outFile := filepath.Join(t.TempDir(), "custom.yaml")
	if got := resolveTarget(apiDir, outFile); got != outFile {
		t.Fatalf("文件目标应原样使用: %s", got)
	}
}
