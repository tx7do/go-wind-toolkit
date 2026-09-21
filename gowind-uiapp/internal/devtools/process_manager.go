package devtools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/svcname"
)

// RunServiceInTerminal 在系统终端中启动服务（不追踪状态，不监控进程）
func RunServiceInTerminal(projectRoot, serviceName string) *CommandResult {
	// serviceName 从 Wails 绑定原样进来。它既参与拼路径(可以写成 ../../x 逃到项目
	// 外去跑别处的 cmd/server),又要落进终端的 shell 命令串,所以先关在"单个路径段"
	// 里;命令串的引用是第二道防线,见 process_manager_unix.go。
	if err := svcname.Validate(serviceName); err != nil {
		return &CommandResult{Success: false, Error: err.Error()}
	}

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
