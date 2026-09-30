//go:build !windows

package devtools

import (
	"fmt"
	"os/exec"
	"strings"
)

// serviceShellCommand 拼出在终端里启动服务的 shell 命令。
//
// 路径必须整体经 shellQuote:这条命令要交给一个真正的 shell 解析(Terminal.app 的
// do script、bash -c),裸插值让路径里的单个引号就能闭合字符串、接上任意命令。
func serviceShellCommand(servicePath string) string {
	return "cd " + shellQuote(servicePath) + " && go run ./cmd/server -c ./configs"
}

// shellQuote 把 s 引用成 POSIX shell 的单引号字面量。单引号内一切皆字面量,唯一
// 要处理的是 `'` 本身:闭合、拼一个转义的单引号、再重新开引号。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// launchServiceInTerminal 在系统终端中运行 go run
func launchServiceInTerminal(serviceName, servicePath string) error {
	title := fmt.Sprintf("GoWind - %s", serviceName)
	goCmd := serviceShellCommand(servicePath)

	// macOS: Terminal.app
	if _, err := exec.LookPath("osascript"); err == nil {
		// 命令走 argv,不拼进 AppleScript 源码:AppleScript 的字符串字面量里 `"`
		// 和 `\` 都是活的语法,想靠转义表把它关住就是把攻击面留给实现者。
		script := `on run argv
	tell application "Terminal"
		activate
		do script (item 1 of argv)
	end tell
end run`
		return exec.Command("osascript", "-e", script, goCmd).Start()
	}

	// Linux: 依次尝试终端模拟器。goCmd 与 title 都是独立的 argv,再过一层 shell
	// 时只有 bash -c 里的 bash 会解析它,而路径已经在 serviceShellCommand 里引用过。
	type td struct {
		name string
		args []string
	}
	terminals := []td{
		{name: "gnome-terminal", args: []string{"--title", title, "--", "bash", "-c", goCmd + "; exec bash"}},
		{name: "konsole", args: []string{"--new-tab", "-p", "tabtitle=" + title, "-e", "bash", "-c", goCmd + "; exec bash"}},
		{name: "alacritty", args: []string{"-t", title, "-e", "bash", "-c", goCmd + "; exec bash"}},
		{name: "xterm", args: []string{"-T", title, "-e", "bash", "-c", goCmd + "; exec bash"}},
	}

	for _, t := range terminals {
		if _, err := exec.LookPath(t.name); err == nil {
			return exec.Command(t.name, t.args...).Start()
		}
	}

	return fmt.Errorf("未找到可用的终端模拟器，请安装 gnome-terminal、konsole、alacritty 或 xterm")
}
