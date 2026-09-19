package detect

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultModuleSearchDepth 向下搜索 go.mod 的默认最大目录深度。
const DefaultModuleSearchDepth = 5

// ModuleCandidate 表示向下发现的一个 Go 模块候选。
type ModuleCandidate struct {
	Dir     string `json:"Dir"`     // 模块根绝对路径
	ModPath string `json:"ModPath"` // go.mod 声明的 module 路径
	RelPath string `json:"RelPath"` // 相对搜索起点的路径，用于展示
}

// HasGoMod 报告 dir 目录下是否直接存在 go.mod 文件。
func HasGoMod(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil && !st.IsDir()
}

// moduleSearchSkipDirs 向下搜索时跳过的目录名（体积大或与模块无关）。
var moduleSearchSkipDirs = map[string]bool{
	"vendor": true, "node_modules": true, "dist": true,
	"build": true, "bin": true, "testdata": true, "target": true,
}

func skipModuleSearchDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	return moduleSearchSkipDirs[name]
}

// FindModulesUnder 从 root 向下（含 root 自身）查找 Go 模块根，最多深入 maxDepth 层。
// 命中某目录的 go.mod 后记录该模块并不再深入其子树，避免把 monorepo 内的每个子包都列出。
// 纯文件系统读取，不执行 go 命令。
func FindModulesUnder(root string, maxDepth int) ([]ModuleCandidate, error) {
	if maxDepth <= 0 {
		maxDepth = DefaultModuleSearchDepth
	}
	var found []ModuleCandidate

	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if HasGoMod(dir) {
			if mp := readModulePath(filepath.Join(dir, "go.mod")); mp != "" {
				rel, _ := filepath.Rel(root, dir)
				found = append(found, ModuleCandidate{Dir: dir, ModPath: mp, RelPath: rel})
			}
			return // 已定位到模块根，不再深入其子树
		}
		if depth >= maxDepth {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() || skipModuleSearchDir(e.Name()) {
				continue
			}
			walk(filepath.Join(dir, e.Name()), depth+1)
		}
	}

	walk(root, 0)
	sort.Slice(found, func(i, j int) bool { return found[i].RelPath < found[j].RelPath })
	return found, nil
}

// readModulePath 从 go.mod 文本中解析 module 路径。
func readModulePath(goModPath string) string {
	f, err := os.Open(goModPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if rest, ok := strings.CutPrefix(line, "module"); ok {
			rest = strings.TrimSpace(rest)
			rest = strings.Trim(rest, "\"`")
			if i := strings.IndexAny(rest, " \t"); i >= 0 {
				rest = rest[:i]
			}
			return rest
		}
	}
	return ""
}
