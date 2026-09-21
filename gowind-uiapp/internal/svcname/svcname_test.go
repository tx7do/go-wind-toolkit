package svcname

import "testing"

// TestValidate 钉住"服务名只能是一个普通路径段"。这些字符串从 Wails 绑定原样进来,
// 分隔符一旦放过,app/<name>/... 就能指到项目外去读、去写、去启动。
func TestValidate(t *testing.T) {
	for _, name := range []string{"core", "user-service", "a.b", "带中文的服务名", "a b"} {
		if err := Validate(name); err != nil {
			t.Errorf("Validate(%q) = %v, 应当放行", name, err)
		}
	}

	for _, name := range []string{"", ".", "..", "../evil", "a/../b", "/abs", `a\b`, "a/b", "x\x00y", "a\nb"} {
		if err := Validate(name); err == nil {
			t.Errorf("Validate(%q) 应当拒绝", name)
		}
	}
}

// TestValidate_RejectsWindowsAlternatePath 反斜杠在 Windows 上是分隔符,在 Unix 上
// 只是普通字符;两个平台都得拒,否则同一份工程文件换个系统就换了语义。
func TestValidate_RejectsWindowsAlternatePath(t *testing.T) {
	for _, name := range []string{`..\evil`, `C:\Windows`, `a/../../b`} {
		if err := Validate(name); err == nil {
			t.Errorf("Validate(%q) 应当拒绝", name)
		}
	}
}
