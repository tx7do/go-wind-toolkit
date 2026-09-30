//go:build windows

package devtools

import (
	"os/exec"
	"syscall"
)

// createNewConsole 是 Win32 的 CREATE_NEW_CONSOLE。Go 的 syscall 包只导出了
// CREATE_NEW_PROCESS_GROUP,不想为这一个常量把 golang.org/x/sys 提成直接依赖。
const createNewConsole = 0x00000010

// launchServiceInTerminal 在新的控制台窗口里直接跑 go run。
//
// 旧实现把标题和目录拼成 `cmd.exe /c start "<title>" /d "<path>" cmd /k go run …`
// 一整条命令行字符串,再交给 cmd.exe 自己解析:cmd.exe 对 `"` 没有任何转义手段,
// 值里一个引号就能闭合字符串并用 `&` 追加命令。这里根本不过 shell——argv 由 Go 负责
// 引号化,工作目录用 Cmd.Dir 传,路径因此不需要被任何解析器读懂;CREATE_NEW_CONSOLE
// 提供"另开一个窗口"这个交互。代价是窗口标题不再可设。
func launchServiceInTerminal(_ string, servicePath string) error {
	cmd := exec.Command("go", "run", "./cmd/server", "-c", "./configs")
	cmd.Dir = servicePath
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewConsole,
	}
	return cmd.Start()
}
