package frontendgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateErrorI18N(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write("user_errors_i18n.zh.json", `{"USER_NOT_FOUND":{"code":404,"message":"用户不存在"},"USER_NAME_CONFLICT":{"code":409,"message":"用户名已占用"}}`)
	write("role_errors_i18n.zh.json", `{"ROLE_NOT_FOUND":{"code":404,"message":"角色不存在"}}`)
	write("user_errors_i18n.en.json", `{"USER_NOT_FOUND":{"code":404,"message":"User not found"}}`)

	files := generateErrorI18N(Options{
		Framework:      FrameworkVueElement,
		ErrorI18NFiles: []string{
			filepath.Join(dir, "user_errors_i18n.zh.json"),
			filepath.Join(dir, "role_errors_i18n.zh.json"),
			filepath.Join(dir, "user_errors_i18n.en.json"),
		},
	})
	if len(files) != 2 {
		t.Fatalf("应产出 zh-CN/en-US 两份词表,得到 %d 份", len(files))
	}

	byPath := map[string]GeneratedFile{}
	for _, f := range files {
		byPath[f.Path] = f
	}
	zh, ok := byPath["locales/zh-CN/errors.json"]
	if !ok {
		t.Fatalf("应有 locales/zh-CN/errors.json,实际: %v", paths(files))
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(zh.Content), &m); err != nil {
		t.Fatalf("词表应为合法 JSON: %v", err)
	}
	if m["USER_NOT_FOUND"] != "用户不存在" || m["ROLE_NOT_FOUND"] != "角色不存在" || len(m) != 3 {
		t.Fatalf("zh 词表合并不对: %v", m)
	}
	if _, ok := byPath["locales/en-US/errors.json"]; !ok {
		t.Fatalf("en 词表应归一到 en-US: %v", strings.Join(paths(files), ","))
	}
}

func TestGenerateErrorI18N_SkipsBrokenFiles(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "broken_errors_i18n.zh.json")
	if err := os.WriteFile(p, []byte("不是 JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	if files := generateErrorI18N(Options{Framework: FrameworkVueElement, ErrorI18NFiles: []string{p}}); len(files) != 0 {
		t.Fatalf("坏词表应被静默跳过,得到 %v", paths(files))
	}
}

func TestGenerateErrorI18N_VbenPath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "user_errors_i18n.zh.json")
	if err := os.WriteFile(p, []byte(`{"USER_NOT_FOUND":{"code":404,"message":"用户不存在"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	files := generateErrorI18N(Options{
		Framework:      FrameworkVueVben,
		ErrorI18NFiles: []string{p},
	})
	if len(files) != 1 || files[0].Path != "locales/langs/zh-CN/errors.json" {
		t.Fatalf("vben 的词表应写 locales/langs/,得到 %+v", files)
	}
}

func paths(fs []GeneratedFile) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Path)
	}
	return out
}
