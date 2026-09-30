package pkg

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModuleInspector 保存项目根目录和模块名。
type ModuleInspector struct {
	Root    string // 含有 `go.mod` 的目录
	ModPath string // go module path
}

// NewModuleInspectorFromGo 使用 `go list -m -json` 在 startDir（或当前目录）执行，
// 返回模块路径和项目根目录。若在 startDir 执行失败，会向上遍历父目录重试。
func NewModuleInspectorFromGo(ctx context.Context, startDir string) (*ModuleInspector, error) {
	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		startDir = wd
	}

	g := NewGoCmdWithTimeout("", 5*time.Minute)
	out, err := g.RunUpwardUntilValid(ctx, startDir, isRealMainModule, "list", "-m", "-json")
	if err != nil {
		return nil, err
	}

	var info struct {
		Path string `json:"Path"`
		Dir  string `json:"Dir"`
	}
	if err = json.Unmarshal(out, &info); err != nil {
		return nil, err
	}

	root := info.Dir
	if root == "" {
		// 兜底：使用 go env GOMOD 推断
		g2 := NewGoCmdWithTimeout("", 30*time.Second)
		if goModOut, e := g2.Output(ctx, "env", "GOMOD"); e == nil {
			goMod := strings.TrimSpace(string(goModOut))
			if goMod != "" {
				root = filepath.Dir(goMod)
			}
		}
		if root == "" {
			root = startDir
		}
	}

	return &ModuleInspector{
		Root:    root,
		ModPath: info.Path,
	}, nil
}

// isRealMainModule 判定 `go list -m -json` 的输出是否来自真实模块目录。
// Go 1.27 起在缺少 go.mod 的目录里该命令仍会 exit 0，返回伪主模块
// （Path="command-line-arguments"，无 GoMod/Dir）。以 GoMod 路径非空且 Path 合法为准。
func isRealMainModule(out []byte) bool {
	var probe struct {
		Path  string `json:"Path"`
		GoMod string `json:"GoMod"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		return false
	}
	return probe.GoMod != "" && probe.Path != "" && probe.Path != "command-line-arguments"
}
