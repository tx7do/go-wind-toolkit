//go:build !windows

package devtools

import (
	"fmt"
	"os/exec"
)

// launchServiceInTerminal 在系统终端中运行 go run
func launchServiceInTerminal(serviceName, servicePath string) error {
	title := fmt.Sprintf("GoWind - %s", serviceName)
	goCmd := fmt.Sprintf("cd '%s' && go run ./cmd/server -c ./configs", servicePath)

	// macOS: Terminal.app
	if _, err := exec.LookPath("osascript"); err == nil {
		script := fmt.Sprintf(`tell application "Terminal"
			activate
			do script "%s"
		end tell`, goCmd)
		return exec.Command("osascript", "-e", script).Start()
	}

	// Linux: 依次尝试终端模拟器
	type td struct {
		name string
		args []string
	}
	terminals := []td{
		{name: "gnome-terminal", args: []string{"--title", title, "--", "bash", "-c", fmt.Sprintf("%s; exec bash", goCmd)}},
		{name: "konsole", args: []string{"--new-tab", "-p", "tabtitle=" + title, "-e", "bash", "-c", fmt.Sprintf("%s; exec bash", goCmd)}},
		{name: "alacritty", args: []string{"-t", title, "-e", "bash", "-c", fmt.Sprintf("%s; exec bash", goCmd)}},
		{name: "xterm", args: []string{"-T", title, "-e", "bash", "-c", fmt.Sprintf("%s; exec bash", goCmd)}},
	}

	for _, t := range terminals {
		if _, err := exec.LookPath(t.name); err == nil {
			return exec.Command(t.name, t.args...).Start()
		}
	}

	return fmt.Errorf("未找到可用的终端模拟器，请安装 gnome-terminal、konsole、alacritty 或 xterm")
}
