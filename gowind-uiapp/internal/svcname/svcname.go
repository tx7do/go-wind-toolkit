// Package svcname 校验前端传进来的服务名。
//
// 这些字符串来自 Wails 绑定,是用户可控输入,而它们既参与 filepath.Join(拼错就能
// 读写项目外的目录),又会落进终端命令串、远端配置中心的 key。所以"必须是单个普通
// 路径段"这条约束要在每个汇点成立,而不是只在某一处补引用。
package svcname

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Validate 接受一个普通的路径段,拒绝空串、"." 与 ".."、任何路径分隔符以及控制字符。
// 分隔符在 Unix 上是 '/', 在 Windows 上 '\' 同样能逃出目录,两者都拒。
func Validate(name string) error {
	if name == "" {
		return fmt.Errorf("服务名为空")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("非法服务名: %q", name)
	}
	if strings.ContainsRune(name, filepath.Separator) || strings.ContainsRune(name, '/') || strings.ContainsRune(name, '\\') {
		return fmt.Errorf("服务名不能包含路径分隔符: %q", name)
	}
	if strings.ContainsRune(name, 0) || strings.ContainsAny(name, "\n\r") {
		return fmt.Errorf("服务名含有控制字符")
	}
	return nil
}
