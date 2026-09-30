package run

import (
	"context"
	"errors"
	"testing"
	"time"
)

// callWithContext 在超时预算内调用 waitOrSignal,把"挂死"变成明确的测试失败。
func callWithContext(t *testing.T, sigCtx context.Context, wait func() error) error {
	t.Helper()

	got := make(chan error, 1)
	go func() { got <- waitOrSignal(sigCtx, wait) }()

	select {
	case err := <-got:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("waitOrSignal 挂起未返回")
		return nil
	}
}

// 服务自行退出(端口被占、配置错)必须带着退出错误返回,而不是等一个永不到来的信号。
func TestWaitOrSignal_ChildExitIsNotSwallowed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exitErr := errors.New("exit status 1")
	if err := callWithContext(t, ctx, func() error { return exitErr }); !errors.Is(err, exitErr) {
		t.Fatalf("waitOrSignal() = %v, want %v", err, exitErr)
	}

	if err := callWithContext(t, ctx, func() error { return nil }); err != nil {
		t.Fatalf("clean exit must return nil, got %v", err)
	}
}

// 信号驱动的终止不是服务的失败,但必须等子进程回收完再返回。
func TestWaitOrSignal_SignalWinsAndReapsChild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	release := make(chan struct{})
	reaped := make(chan struct{})
	wait := func() error {
		<-release
		close(reaped)
		return errors.New("signal: killed")
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
		close(release)
	}()

	if err := callWithContext(t, ctx, wait); err != nil {
		t.Fatalf("signal-driven termination must return nil, got %v", err)
	}

	select {
	case <-reaped:
	default:
		t.Fatal("返回前未等待子进程回收")
	}
}
