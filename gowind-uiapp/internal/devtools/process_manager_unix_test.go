//go:build !windows

package devtools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestShellQuote_RoundTripsThroughSh 让真正的 /bin/sh 来判:引用后的字符串必须原样
// 变成一个参数。凡是只在 Go 侧比对转义表的测试,都证明不了 shell 那头的结论。
func TestShellQuote_RoundTripsThroughSh(t *testing.T) {
	for _, payload := range []string{
		"/tmp/plain", "/tmp/with space", "/tmp/a'b", `/tmp/a"b`, `/tmp/a\b`,
		"/tmp/a$(id)b", "/tmp/a`id`b", "/tmp/a;b|c&d", "/tmp/a\nb", "", "--",
	} {
		out, err := exec.Command("/bin/sh", "-c", "printf %s "+shellQuote(payload)).CombinedOutput()
		if err != nil {
			t.Fatalf("sh 执行失败 (payload %q): %v\n%s", payload, err, out)
		}
		if got := string(out); got != payload {
			t.Errorf("shell 收到的值 = %q, 期望 %q", got, payload)
		}
	}
}

// TestServiceShellCommand_DoesNotExecutePathData 成对验证:同一条含 `'` 的目录路径,
// 旧实现(裸插值进单引号)会让 sh 把后半截当命令执行,新实现只把它当数据。
//
// 两条都让 sh 真跑,但目录都不存在,所以命令除了被注入的 touch 之外不会有任何副作用。
func TestServiceShellCommand_DoesNotExecutePathData(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "pwned")
	hostile := filepath.Join(dir, "x'; touch "+marker+"; echo '")

	t.Run("旧写法确实可注入", func(t *testing.T) {
		legacy := "cd '" + hostile + "' && go run ./cmd/server -c ./configs"
		_ = exec.Command("/bin/sh", "-c", legacy).Run()
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("对照组失效:裸插值没有把命令跑起来,这个测试已经测不到东西了 (%v)", err)
		}
		if err := os.Remove(marker); err != nil {
			t.Fatalf("清理 marker 失败: %v", err)
		}
	})

	t.Run("现写法只当数据", func(t *testing.T) {
		cmd := serviceShellCommand(hostile)
		if err := exec.Command("/bin/sh", "-c", cmd).Run(); err == nil {
			t.Fatal("目录并不存在,sh 不该报告成功")
		}
		if _, err := os.Stat(marker); err == nil {
			t.Fatal("被注入的 touch 执行了:路径里的引号逃出了引用")
		}
	})
}

// TestOsascriptReceivesCommandViaArgv 钉住 macOS 分支依赖的机制:命令经 argv 交给
// osascript 时必须逐字节到达,不必再逃一层 AppleScript 字符串字面量。
// 用 `return` 而不是 `do script`,免得真的打开终端窗口。
func TestOsascriptReceivesCommandViaArgv(t *testing.T) {
	if _, err := exec.LookPath("osascript"); err != nil {
		t.Skip("本机没有 osascript(非 macOS),macOS 分支不参与")
	}

	want := `cd '/tmp/x' y"z\w' && go run ./cmd/server`
	out, err := exec.Command("osascript",
		"-e", "on run argv",
		"-e", "return item 1 of argv",
		"-e", "end run",
		want,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("osascript 失败: %v\n%s", err, out)
	}
	if got := strings.TrimRight(string(out), "\n"); got != want {
		t.Errorf("osascript 收到的命令 = %q, 期望 %q", got, want)
	}
}
