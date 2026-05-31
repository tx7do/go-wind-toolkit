//go:build windows

package devtools

import (
	"fmt"
	"os/exec"
	"syscall"
)

// launchServiceInTerminal 在新的 cmd 窗口中运行 go run
// 使用 SysProcAttr.CmdLine 直接控制命令行字符串，避免 Go 参数转义问题
func launchServiceInTerminal(serviceName, servicePath string) error {
	title := fmt.Sprintf("GoWind - %s", serviceName)
	cmdLine := fmt.Sprintf(`/c start "%s" /d "%s" cmd /k go run ./cmd/server -c ./configs`, title, servicePath)

	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: cmdLine,
	}
	return cmd.Run()
}
