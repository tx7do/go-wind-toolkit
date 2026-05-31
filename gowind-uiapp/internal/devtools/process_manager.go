package devtools

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunServiceInTerminal 在系统终端中启动服务（不追踪状态，不监控进程）
func RunServiceInTerminal(projectRoot, serviceName string) *CommandResult {
	servicePath := filepath.Join(projectRoot, "app", serviceName, "service")
	appPath := filepath.Join(servicePath, "cmd", "server")

	if _, err := os.Stat(appPath); err != nil {
		return &CommandResult{Success: false, Error: fmt.Sprintf("服务目录不存在: %s", appPath)}
	}

	if err := launchServiceInTerminal(serviceName, servicePath); err != nil {
		return &CommandResult{Success: false, Error: fmt.Sprintf("启动服务失败: %v", err)}
	}

	return &CommandResult{
		Success: true,
		Output:  fmt.Sprintf("服务 %s 已在终端中启动", serviceName),
	}
}
