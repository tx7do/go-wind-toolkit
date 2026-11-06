package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

// blockingChat ChatStream 挂在 ctx.Done() 上的桩:模拟长时间在途的流式生成。
type blockingChat struct{}

func (b *blockingChat) Chat(context.Context, string, string) (string, error) {
	return "", context.Canceled
}

func (b *blockingChat) ChatStream(ctx context.Context, _ string, _ string, onDelta func(string)) (string, error) {
	onDelta("部分内容")
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(30 * time.Second):
		// 永不该走到:取消失败会让测试在超时保护下失败
		return testDDL, nil
	}
}

func (b *blockingChat) TestConnection(context.Context) (*StepResult, error) {
	return &StepResult{Success: true}, nil
}

func TestCancelTask_DDL(t *testing.T) {
	svc := &Service{config: DefaultConfig(), client: &blockingChat{}}

	done := make(chan *StepResult, 1)
	go func() {
		result, _ := svc.GenerateDDLValidated("用户管理", nil)
		done <- result
	}()

	// 等任务真正进入注册表(流式已推出第一个增量)再取消
	time.Sleep(50 * time.Millisecond)
	if !svc.CancelTask(taskDDL) {
		t.Fatal("在途任务应可取消")
	}

	select {
	case result := <-done:
		if result.Success {
			t.Fatal("取消后 result.Success 应为 false")
		}
		if !strings.Contains(result.Error, "已取消") {
			t.Fatalf("错误应归一为「已取消」,得到: %s", result.Error)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("取消后生成应在 5 秒内返回")
	}

	// 任务结束后注册表应已清空,再取消返回 false
	if svc.CancelTask(taskDDL) {
		t.Fatal("结束后再取消应返回 false")
	}
}

func TestCancelTask_NoTaskRunning(t *testing.T) {
	svc := &Service{config: DefaultConfig(), client: &mockChat{}}
	if svc.CancelTask(taskDDL) || svc.CancelTask(taskPartition) || svc.CancelTask(taskReview) {
		t.Fatal("无在途任务时取消应返回 false")
	}
}

func TestTaskCtx_ReplacesStaleTask(t *testing.T) {
	svc := &Service{config: DefaultConfig(), client: &blockingChat{}}

	// 模拟异常退出漏清:连续两次注册,旧的应被兜底取消
	ctx1 := svc.taskCtx(taskDDL)
	ctx2 := svc.taskCtx(taskDDL)
	select {
	case <-ctx1.Done():
	default:
		t.Fatal("旧任务的 ctx 应被新注册兜底取消")
	}
	if ctx2.Err() != nil {
		t.Fatal("新任务的 ctx 不应被取消")
	}
	svc.finishTask(taskDDL)
}
