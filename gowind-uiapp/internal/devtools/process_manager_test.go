package devtools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunServiceInTerminal_RejectsPathEscape 目标目录真的存在时也不许启动:
// tmp/app/../evil/service/cmd/server 是合法存在的目录,旧实现只看 os.Stat 是否成功。
func TestRunServiceInTerminal_RejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	serverDir := filepath.Join(root, "evil", "service", "cmd", "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	result := RunServiceInTerminal(root, "../evil")
	if result.Success {
		t.Fatal("用 ../ 逃出 app/ 的服务名不能被判成可启动")
	}
	if !strings.Contains(result.Error, "路径分隔符") {
		t.Errorf("应当因服务名校验而失败,实际错误: %q", result.Error)
	}
}

// TestRunServiceInTerminal_MissingService 正常名字 + 目录不存在时按原路径报错,
// 不触达终端启动。
func TestRunServiceInTerminal_MissingService(t *testing.T) {
	result := RunServiceInTerminal(t.TempDir(), "core")
	if result.Success {
		t.Fatal("目录不存在却报告成功")
	}
	if !strings.Contains(result.Error, "服务目录不存在") {
		t.Errorf("错误应当指向缺失的目录,实际: %q", result.Error)
	}
}
