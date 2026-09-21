package run

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/build"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

// watchEnabled 对应 --watch 旗标:监听文件变更,重建并重启受影响的服务。
var watchEnabled bool

func init() {
	CmdRun.Flags().BoolVarP(&watchEnabled, "watch", "w", false, "watch sources and configs, rebuild and restart changed services on save (hot reload)")
}

// CmdRun run project command.
var CmdRun = &cobra.Command{
	Use:   "run",
	Short: "Run service project",
	Long: `Run service project. Example: gowind run admin

With --watch the started services are watched: saving a .go/.yaml/.proto
source rebuilds and restarts only the affected services (all of them for
changes in shared module-level code). A failed rebuild keeps the old
process running.`,
	RunE:          Run,
	SilenceUsage:  true,
}

// Run service.
func Run(cmd *cobra.Command, args []string) error {
	cmdArgs, _ := pkg.SplitArgs(cmd, args)

	inspector, err := pkg.NewModuleInspectorFromGo(cmd.Context(), "")
	if err != nil {
		return err
	}

	// 先在模块根目录运行 `go mod tidy`
	if err = pkg.GoModTidy(cmd.Context(), inspector.Root); err != nil {
		return err
	}

	var serviceName string

	if len(cmdArgs) > 0 {
		serviceName = strings.TrimSpace(cmdArgs[0])
		if serviceName == "" {
			return fmt.Errorf("service name is required")
		}

		valid, err := pkg.IsValidServiceName(inspector.Root, serviceName)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("service '%s' does not exist or is not valid (missing cmd/server or configs)", serviceName)
		}
	} else {
		// 未指定服务名称，检查当前目录是否为服务目录

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("os.Getwd: %w", err)
		}

		hasCmd, hasConfigs, err := pkg.HasCmdAndConfigs(wd)
		if err != nil {
			return err
		}

		if hasCmd && hasConfigs {
			// 当前目录即为服务目录
			if watchEnabled {
				return watchServices(inspector.Root, []watchTarget{{
					name: deriveWatchServiceName(wd),
					dir:  wd,
				}})
			}
			return runService(cmd.Context(), wd)
		}

		// 当前目录不是服务目录:枚举并一并运行模块内全部服务。
		names, err := pkg.ListServiceNames(inspector.Root)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return fmt.Errorf("no valid services found under %s", filepath.Join(inspector.Root, "app"))
		}
		if watchEnabled {
			targets := make([]watchTarget, 0, len(names))
			for _, name := range names {
				targets = append(targets, watchTarget{name: name, dir: filepath.Join(inspector.Root, "app", name, "service")})
			}
			return watchServices(inspector.Root, targets)
		}
		return runAllServices(cmd.Context(), inspector.Root, names)
	}

	if watchEnabled {
		return watchServices(inspector.Root, []watchTarget{{
			name: serviceName,
			dir:  filepath.Join(inspector.Root, "app", serviceName, "service"),
		}})
	}

	servicePath := path.Join(inspector.Root, "/app/", serviceName, "/service")

	return runService(cmd.Context(), servicePath)
}

// runService 运行单个服务:先编译服务二进制再直接执行。
// 直接执行二进制(而非 go run)使终止语义精确——SIGINT/SIGTERM 经信号
// 上下文由 exec 撤销的就是服务进程本身,不会留下 go run 包装进程已死、
// 服务进程仍存的孤儿(与 runAllServices 同一语义)。
func runService(ctx context.Context, serviceWorkPath string) error {
	// 信号上下文:中断/终止时由 exec 撤销子进程(编译阶段一并覆盖)。
	// 派生自 cmd.Context(),使上层取消同样能撤销。
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	name := deriveWatchServiceName(serviceWorkPath)
	binPath, err := build.ResolveOutputPath(name, serviceWorkPath, runtime.GOOS, runtime.GOARCH, false)
	if err != nil {
		return err
	}
	if err = build.BuildBinary(sigCtx, serviceWorkPath, binPath, runtime.GOOS, runtime.GOARCH, build.BuildOptions{}); err != nil {
		return err
	}

	proc := exec.CommandContext(sigCtx, binPath, "-c", filepath.Join(serviceWorkPath, "configs"))
	proc.Dir = serviceWorkPath
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// 两种收束路径都得走,不能只等信号:服务自行退出(正常结束、端口被占、
	// 配置错)若只 <-sigCtx.Done() 就永久挂起且退出码恒 0。
	return waitOrSignal(sigCtx, proc.Wait)
}

// waitOrSignal 在"子进程退出"与"信号到达"之间竞取先发生者。
// 信号驱动的终止不作为服务的失败;服务自行退出则原样回传其退出错误。
func waitOrSignal(sigCtx context.Context, wait func() error) error {
	waitCh := make(chan error, 1)
	go func() { waitCh <- wait() }()

	select {
	case waitErr := <-waitCh:
		if sigCtx.Err() != nil {
			// 由信号撤销子进程,退出码不属于服务的语义。
			return nil
		}
		return waitErr
	case <-sigCtx.Done():
		// exec 已发出 Kill,等它回收完再返回,不留僵尸。
		<-waitCh
		return nil
	}
}

// runAllServices 一并运行模块内全部服务:先编译全部服务二进制,再并发拉起。
// 各服务输出按行加名称前缀;SIGINT/SIGTERM 统一停止全部进程。
// 直接执行二进制(而非 go run)使终止语义精确——杀掉的就是服务进程本身,
// 不会留下 go run 包装进程已死、服务进程仍存的孤儿。
func runAllServices(ctx context.Context, root string, names []string) error {
	type serviceProc struct {
		name    string
		svcDir  string
		binPath string
	}

	// 信号上下文:中断/终止时由 exec 撤销全部子进程。置于编译阶段之前,
	// 使 Ctrl+C 同样能中断卡住的构建。
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 阶段一:编译全部服务(任一失败即中止,不启动任何进程)。
	procs := make([]serviceProc, 0, len(names))
	for _, name := range names {
		svcDir := filepath.Join(root, "app", name, "service")
		binPath, err := build.ResolveOutputPath(name, svcDir, runtime.GOOS, runtime.GOARCH, false)
		if err != nil {
			return fmt.Errorf("resolve output for service '%s' failed: %w", name, err)
		}
		if err = build.BuildBinary(sigCtx, svcDir, binPath, runtime.GOOS, runtime.GOARCH, build.BuildOptions{}); err != nil {
			return fmt.Errorf("build for service '%s' failed: %w", name, err)
		}
		procs = append(procs, serviceProc{name: name, svcDir: svcDir, binPath: binPath})
	}

	// 阶段二:并发拉起全部服务。
	// 终止语义:SIGINT/SIGTERM 撤销信号上下文,exec.CommandContext 随之杀掉各子进程;
	// 启动完成后另有显式 Kill 循环作冗余兜底,确保任何情形下子进程都被回收。
	stdoutMu := &sync.Mutex{}
	stderrMu := &sync.Mutex{}
	var wg sync.WaitGroup
	// failedMu 保护 failed:告警在各自的 Wait 协程里发生,退出码要在汇合后统一回传。
	var failedMu sync.Mutex
	var failed []string
	type startedProc struct {
		name string
		cmd  *exec.Cmd
	}
	startedProcs := make([]startedProc, 0, len(procs))
	for _, p := range procs {
		configPath := filepath.Join(p.svcDir, "configs")
		proc := exec.CommandContext(sigCtx, p.binPath, "-c", configPath)
		proc.Dir = p.svcDir
		proc.Stdout = newPrefixedWriter(os.Stdout, p.name, stdoutMu)
		proc.Stderr = newPrefixedWriter(os.Stderr, p.name, stderrMu)
		if err := proc.Start(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: failed to start service '%s': %s\033[m\n", p.name, err.Error())
			continue
		}
		startedProcs = append(startedProcs, startedProc{name: p.name, cmd: proc})
		wg.Add(1)
		go func(name string, proc *exec.Cmd) {
			defer wg.Done()
			err := proc.Wait()
			if sigCtx.Err() != nil {
				return // 整组停止,不逐个告警
			}
			_, _ = fmt.Fprintf(os.Stderr, "\033[33mWARNING: service '%s' exited: %v\033[m\n", name, err)
			if err != nil {
				failedMu.Lock()
				failed = append(failed, name)
				failedMu.Unlock()
			}
		}(p.name, proc)
	}

	if len(startedProcs) == 0 {
		return fmt.Errorf("no service could be started")
	}

	// 冗余兜底:信号上下文一旦取消,显式杀掉全部已启动子进程。
	go func() {
		<-sigCtx.Done()
		for _, sp := range startedProcs {
			if sp.cmd.Process != nil {
				_ = sp.cmd.Process.Kill()
			}
		}
	}()

	_, _ = fmt.Fprintf(os.Stdout, "\033[36mStarted %d service(s); Ctrl+C stops all of them.\033[m\n", len(startedProcs))

	wg.Wait()

	failedMu.Lock()
	defer failedMu.Unlock()
	if len(failed) > 0 {
		return fmt.Errorf("以下服务异常退出: %s", strings.Join(failed, ", "))
	}
	return nil
}

// prefixedWriter 给子进程输出逐行加服务名前缀。
// 同一底层流的多个实例共享互斥锁,保证行粒度不交叉。
type prefixedWriter struct {
	w      io.Writer
	prefix string
	mu     *sync.Mutex
	buf    []byte
}

func newPrefixedWriter(w io.Writer, name string, mu *sync.Mutex) *prefixedWriter {
	return &prefixedWriter{
		w:      w,
		prefix: "\033[36m[" + name + "]\033[0m ",
		mu:     mu,
	}
}

func (p *prefixedWriter) Write(b []byte) (int, error) {
	total := len(b)
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(b) > 0 {
		nl := bytes.IndexByte(b, '\n')
		if nl < 0 {
			p.buf = append(p.buf, b...)
			break
		}
		p.buf = append(p.buf, b[:nl+1]...)
		_, _ = p.w.Write([]byte(p.prefix))
		_, _ = p.w.Write(p.buf)
		p.buf = p.buf[:0]
		b = b[nl+1:]
	}
	return total, nil
}
