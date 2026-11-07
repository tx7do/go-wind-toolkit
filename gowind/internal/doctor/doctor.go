// Package doctor 实现 `gow doctor`：体检本机开发环境——CLI 依赖的构建工具、
// protoc 插件是否可用,以及常见装配问题(GOBIN 是否在 PATH)。只读探测,不改环境。
package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// CmdDoctor `gow doctor` 命令。
var CmdDoctor = &cobra.Command{
	Use:   "doctor",
	Short: "检查本机开发环境(构建工具、protoc 插件、GOBIN 装配)",
	Long: `体检本机开发环境,报告 gow 及生成管线依赖的外部工具可用性:

  - 必需: go, git —— 缺失时退出码非 0
  - 生成管线: protoc, wire, buf, node, pnpm 与各 protoc-gen-* 插件 —— 缺失只警告,
    按需安装即可(用到哪条管线装哪些)

探测全部只读,不会修改任何环境。`,
	RunE:         RunCheck,
	SilenceUsage: true,
}

// checkKind 决定缺失时的严重级别与提示方式。
type checkKind int

const (
	kindRequired checkKind = iota // 缺失即失败(gow 自身依赖)
	kindPipeline                  // 生成管线工具,缺失警告
	kindPlugin                    // protoc 插件,存在即装好
)

type check struct {
	name string
	kind checkKind
	// probe 为空则只做 LookPath(protoc 插件多为 stdin 读取器,喂 --version
	// 只会得到 flag 错误,没有可用的版本探针)。
	probe []string
	hint  string
	// minVersion 语义版本地板:探测到的版本低于它时降级为 WARN
	// (工具在但不达标,典型:生成的前端产物用 Vite 7,node 须 >= 20.19)。
	minVersion string
}

func checks() []check {
	return []check{
		{name: "go", kind: kindRequired, probe: []string{"version"}, hint: "https://go.dev/dl/"},
		{name: "git", kind: kindRequired, probe: []string{"--version"}, hint: "https://git-scm.com"},
		{name: "protoc", kind: kindPipeline, probe: []string{"--version"}, hint: "https://protobuf.dev/install/"},
		{name: "buf", kind: kindPipeline, probe: []string{"--version"}, hint: "https://buf.build/docs/installation"},
		{name: "wire", kind: kindPipeline, hint: "go install github.com/google/wire/cmd/wire@latest"},
		{name: "node", kind: kindPipeline, probe: []string{"--version"}, minVersion: "20.19.0", hint: "https://nodejs.org(前端生成产物联调需要;Vite 7 要求 >= 20.19)"},
		{name: "pnpm", kind: kindPipeline, probe: []string{"--version"}, hint: "npm i -g pnpm(可选,前端项目包管理)"},
		{name: "protoc-gen-go", kind: kindPlugin},
		{name: "protoc-gen-go-grpc", kind: kindPlugin},
		{name: "protoc-gen-go-http", kind: kindPlugin},
		{name: "protoc-gen-wind-errors", kind: kindPlugin},
		{name: "protoc-gen-go-redact", kind: kindPlugin},
		{name: "protoc-gen-fieldmap", kind: kindPlugin},
		{name: "protoc-gen-typescript-http", kind: kindPlugin},
		{name: "protoc-gen-dart-http", kind: kindPlugin},
	}
}

// RunCheck 执行全部体检项并打印报告。
func RunCheck(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "gow doctor — environment check")
	fmt.Fprintln(out)

	missingRequired, missingOptional := 0, 0
	for _, c := range checks() {
		path, err := exec.LookPath(c.name)
		if err != nil {
			switch c.kind {
			case kindRequired:
				missingRequired++
				fmt.Fprintf(out, "  \033[31mMISS\033[m  %-28s required — %s\n", c.name, c.hint)
			case kindPipeline:
				missingOptional++
				fmt.Fprintf(out, "  \033[33mMISS\033[m  %-28s %s\n", c.name, c.hint)
			case kindPlugin:
				missingOptional++
				fmt.Fprintf(out, "  \033[33mMISS\033[m  %-28s go install github.com/tx7do/go-wind-toolkit/%s@latest\n", c.name, c.name)
			}
			continue
		}
		version := probe(c.name, c.probe)
		if version == "" {
			version = shortPath(path)
		}
		if c.minVersion != "" && versionTooLow(version, c.minVersion) {
			missingOptional++
			fmt.Fprintf(out, "  \033[33mOLD\033[m   %-28s %s — 低于地板 %s,%s\n", c.name, version, c.minVersion, c.hint)
			continue
		}
		fmt.Fprintf(out, "  \033[32mOK\033[m    %-28s %s\n", c.name, version)
	}

	// GOBIN 装配检查:go install 装的插件目录必须能被 protoc 找到(在 PATH 里)。
	if warn := gobinMisassemble(); warn != "" {
		missingOptional++
		fmt.Fprintf(out, "\n  \033[33mWARN\033[m  %s\n", warn)
	}

	fmt.Fprintln(out)
	switch {
	case missingRequired > 0:
		return fmt.Errorf("%d required tool(s) missing", missingRequired)
	case missingOptional > 0:
		fmt.Fprintf(out, "\033[33m%d optional tool(s) missing — install as needed.\033[m\n", missingOptional)
	default:
		fmt.Fprintln(out, "\033[32mAll checks passed.\033[m")
	}
	return nil
}

// versionTooLow 尽力比较语义版本:解析失败(非 vX.Y.Z 形态)不告警,
// 宁可放过也不误报。
func versionTooLow(got, floor string) bool {
	parse := func(v string) (int, int, int, bool) {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		if i := strings.IndexByte(v, '-'); i >= 0 {
			v = v[:i]
		}
		parts := strings.Split(v, ".")
		var nums [3]int
		for i := 0; i < 3 && i < len(parts); i++ {
			n := 0
			for _, r := range parts[i] {
				if r < '0' || r > '9' {
					return 0, 0, 0, false
				}
				n = n*10 + int(r-'0')
			}
			nums[i] = n
		}
		return nums[0], nums[1], nums[2], true
	}
	gm, gn, gp, ok1 := parse(got)
	fm, fn, fp, ok2 := parse(floor)
	if !ok1 || !ok2 {
		return false
	}
	if gm != fm {
		return gm < fm
	}
	if gn != fn {
		return gn < fn
	}
	return gp < fp
}

// probe 尽力跑一次版本探针:失败(不存在该 flag/无输出/超时)就返回空串,不报错。
func probe(name string, args []string) string {
	if len(args) == 0 {
		return ""
	}
	cmd := exec.Command(name, args...)
	out, err := runWithTimeout(cmd, 3*time.Second)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(firstLine(out))
	if line == "" {
		return ""
	}
	return line
}

func runWithTimeout(cmd *exec.Cmd, d time.Duration) (string, error) {
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
		return buf.String(), nil
	case <-time.After(d):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("timeout")
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func shortPath(p string) string {
	p = filepath.ToSlash(p)
	if home, err := os.UserHomeDir(); err == nil {
		if hs := filepath.ToSlash(home) + "/"; strings.HasPrefix(p, hs) {
			return "~/" + p[len(hs):]
		}
	}
	return p
}

// gobinMisassemble 检查 `go install` 产物的落点是否在 PATH 上:
// protoc 找不到插件的最常见原因就是 GOBIN(或 GOPATH/bin)没进 PATH。
func gobinMisassemble() string {
	out, err := runWithTimeout(exec.Command("go", "env", "GOBIN", "GOPATH"), 3*time.Second)
	if err != nil {
		return ""
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return ""
	}
	gobin := fields[0]
	if gobin == "" && len(fields) > 1 {
		gobin = filepath.Join(fields[1], "bin")
	}
	if gobin == "" {
		return ""
	}
	if pathIn(gobin) {
		return ""
	}
	return fmt.Sprintf("go install 落点 %s 不在 PATH 上——protoc 将找不到 protoc-gen-* 插件", shortPath(gobin))
}

func pathIn(dir string) bool {
	dir = filepath.Clean(dir)
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(p) == dir {
			return true
		}
	}
	return false
}
