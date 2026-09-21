package ent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

// EntCmd 封装 ent 命令调用的数据和行为。
type EntCmd struct {
	Args []string
	// TargetDir 一律是 ent 的 schema 目录本身(如 <service>/internal/data/ent/schema),
	// 与 RunNew 的 --target、RunGenerate 的位置参数同一语义。
	TargetDir string
	Timeout   time.Duration
	goCmd     *pkg.GoCmd
}

func NewEntCmd(targetDir string) *EntCmd {
	if targetDir == "" {
		targetDir = "internal/data/ent/schema"
	}
	timeout := 2 * time.Minute
	e := &EntCmd{
		TargetDir: filepath.Clean(targetDir),
		Timeout:   timeout,
	}
	e.goCmd = pkg.NewGoCmdWithTimeout(e.TargetDir, timeout)
	return e
}

// tryGoRunFirst 尝试在项目中通过 `go run entgo.io/ent/cmd/ent ...` 执行命令
// （从 schema 目录起向上查找可执行的模块根）。
// 成功时打印输出并返回 nil；失败时返回最后一次错误以便调用方决定后续处理。
func (e *EntCmd) tryGoRunFirst(ctx context.Context, args ...string) error {
	out, err := e.goCmd.RunUpwardUntilSucceeds(ctx, e.TargetDir, args...)
	if err == nil {
		// 打印 combined output（stdout+stderr）
		if len(out) > 0 {
			fmt.Print(string(out))
		}
		return nil
	}
	return err
}

// runGlobalEntIfAvailable 尝试在 PATH 中查找全局 `ent` 可执行程序并执行（遵循 e.Timeout）。
func (e *EntCmd) runGlobalEntIfAvailable(ctx context.Context, args ...string) error {
	entPath, err := exec.LookPath("ent")
	if err != nil {
		return fmt.Errorf("no local go-run ent succeeded and no global ent found: %w", err)
	}

	// 准备上下文（调用方 ctx 叠加 e.Timeout 超时）
	var runCtx context.Context
	var cancel context.CancelFunc
	if e.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, e.Timeout)
	} else {
		runCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	cmd := exec.CommandContext(runCtx, entPath, args...)

	// 将工作目录设置为 `e.TargetDir` 的绝对路径
	if e.TargetDir != "" {
		if abs, err := filepath.Abs(e.TargetDir); err == nil {
			cmd.Dir = abs
		} else {
			// 无法获取绝对路径时仍可使用原始值
			cmd.Dir = e.TargetDir
		}
	}

	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	//fmt.Println("running global ent:", entPath, args)
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("global ent failed: %w", err)
	}
	return nil
}

// schemaTargetArg 把 `ent new` 的落点钉成绝对路径。ent 的 --target 默认值是相对工作目录的
// "ent/schema",而 go run 分支会沿 RunUpwardUntilSucceeds 向上漂到模块根——不显式传 --target
// 时 schema 就落到 CWD 而非服务目录,命令照样退出 0。
func (e *EntCmd) schemaTargetArg() string {
	abs, err := filepath.Abs(e.TargetDir)
	if err != nil {
		abs = e.TargetDir
	}
	return "--target=" + abs
}

// verifySchemasCreated ent 退出码 0 不等于文件落在了预期的 schema 目录。按 ent 自己的命名
// 规则(<小写名>.go)复核一遍,把"报成功但 schema 没进生成"变成显式失败。
func (e *EntCmd) verifySchemasCreated(names []string) error {
	var missing []string
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(e.TargetDir, strings.ToLower(name)+".go")); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("ent new 退出正常但 %s 下缺少 schema 文件: %s", e.TargetDir, strings.Join(missing, ", "))
	}
	return nil
}

// RunNew 使用优先使用项目内的 go run，如果失败则尝试全局 ent 可执行程序来执行 `ent new`。
func (e *EntCmd) RunNew(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return errors.New("at least one schema name is required")
	}
	if err := os.MkdirAll(e.TargetDir, 0o755); err != nil {
		return fmt.Errorf("create schema dir failed: %w", err)
	}

	targetArg := e.schemaTargetArg()

	// 尝试 go run first
	goArgs := []string{"run", "entgo.io/ent/cmd/ent", "new", targetArg}
	goArgs = append(goArgs, names...)

	if err := e.tryGoRunFirst(ctx, goArgs...); err == nil {
		return e.verifySchemasCreated(names)
	}

	// 如果 go run 不可用，则尝试全局 ent
	entArgs := append([]string{"new", targetArg}, names...)
	if err := e.runGlobalEntIfAvailable(ctx, entArgs...); err != nil {
		return err
	}
	return e.verifySchemasCreated(names)
}

// RunGenerate 使用优先使用项目内的 go run，如果失败则尝试全局 ent 可执行程序来执行 `ent generate`。
// extraArgs 可传入像 `--feature` 这样的额外参数。
func (e *EntCmd) RunGenerate(ctx context.Context, extraArgs ...string) error {
	targetDir := e.TargetDir
	if targetDir == "" {
		targetDir = "."
	}

	goArgs := []string{"run", "entgo.io/ent/cmd/ent", "generate"}
	if len(extraArgs) > 0 {
		goArgs = append(goArgs, extraArgs...)
	}
	goArgs = append(goArgs, targetDir)

	// 尝试 go run first
	if err := e.tryGoRunFirst(ctx, goArgs...); err == nil {
		return nil
	}

	// 如果 go run 不可用，则尝试全局 ent
	entArgs := []string{"generate"}
	if len(extraArgs) > 0 {
		entArgs = append(entArgs, extraArgs...)
	}
	entArgs = append(entArgs, targetDir)
	return e.runGlobalEntIfAvailable(ctx, entArgs...)
}
