package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Module struct {
	Path    string
	Version string
}

// ModuleInspector 保存项目根目录和模块名。
type ModuleInspector struct {
	Root    string // 含有 `go.mod` 的目录
	ModPath string // go module path

	GoVersion    string   // Go 版本要求
	Main         bool     // 是否为主模块
	Indirect     bool     // 是否为间接依赖
	Replace      *Module  // 替换信息
	Version      string   // 模块版本
	Dependencies []Module // 依赖列表
}

// NewModuleInspectorFromGo 使用 `go list -m -json` 在 startDir（或当前目录）执行，
// 返回模块路径和项目根目录。若在 startDir 执行失败，会向上遍历父目录重试。
func NewModuleInspectorFromGo(rootDir string) (*ModuleInspector, error) {
	if rootDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		rootDir = wd
	}

	g := NewGoCmd("")
	out, err := g.RunUpwardUntilValid(rootDir, isRealMainModule, "list", "-m", "-json")
	if err != nil {
		return nil, err
	}

	var info struct {
		Path      string  `json:"Path"`
		Dir       string  `json:"Dir"`
		GoVersion string  `json:"GoVersion"`
		GoMod     string  `json:"GoMod"`
		Main      bool    `json:"Main"`
		Indirect  bool    `json:"Indirect"`
		Version   string  `json:"Version"`
		Replace   *Module `json:"Replace"`
	}
	if err = json.Unmarshal(out, &info); err != nil {
		return nil, err
	}

	root := info.Dir
	if root == "" {
		// 兜底：使用 go env GOMOD 推断
		g2 := NewGoCmd("")
		if goModOut, e := g2.Output("env", "GOMOD"); e == nil {
			goMod := strings.TrimSpace(string(goModOut))
			if goMod != "" {
				root = filepath.Dir(goMod)
			}
		}
		if root == "" {
			root = rootDir
		}
	}

	return &ModuleInspector{
		Root:      root,
		ModPath:   info.Path,
		GoVersion: info.GoVersion,
		Main:      info.Main,
		Indirect:  info.Indirect,
		Replace:   info.Replace,
		Version:   info.Version,
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
