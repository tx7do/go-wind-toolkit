package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

// streamingPartitionChat 区分流式/非流式端点的桩:
// Chat 永远返回坏响应;ChatStream 按 streamResponses 顺序回,切成增量。
type streamingPartitionChat struct {
	streamResponses []string
	i               int
}

func (m *streamingPartitionChat) Chat(context.Context, string, string) (string, error) {
	return "CHAT端点不应被调用", nil
}

func (m *streamingPartitionChat) ChatStream(_ context.Context, _ string, _ string, onDelta func(string)) (string, error) {
	if m.i >= len(m.streamResponses) {
		return "", context.Canceled
	}
	resp := m.streamResponses[m.i]
	m.i++
	// 切成两半模拟流式增量
	half := len(resp) / 2
	if half == 0 {
		half = len(resp)
	}
	if onDelta != nil {
		onDelta(resp[:half])
		if half < len(resp) {
			onDelta(resp[half:])
		}
	}
	return resp, nil
}

func (m *streamingPartitionChat) TestConnection(context.Context) (*StepResult, error) {
	return &StepResult{Success: true}, nil
}

func TestPartitionMicroservicesValidated_Stream(t *testing.T) {
	svc := &Service{config: DefaultConfig(), client: &streamingPartitionChat{streamResponses: []string{partitionGood}}}

	var attempts []int
	var streamed strings.Builder
	partitions, ddlTables, _, err := svc.PartitionMicroservicesValidatedStream(testDDL, func(attempt int, delta string) {
		attempts = append(attempts, attempt)
		streamed.WriteString(delta)
	})
	if err != nil {
		t.Fatalf("流式划分失败: %v", err)
	}
	if len(partitions) != 2 {
		t.Fatalf("应解析出 2 个划分,得到 %d", len(partitions))
	}
	if strings.Join(ddlTables, ",") != "user,user_role,order" {
		t.Fatalf("表清单透传不符: %v", ddlTables)
	}
	if streamed.String() != partitionGood {
		t.Fatalf("流式增量拼接应还原完整响应")
	}
	if len(attempts) == 0 {
		t.Fatal("应有增量回调")
	}
	for _, a := range attempts {
		if a != 1 {
			t.Fatalf("单次成功的 attempt 应全为 1,得到 %v", attempts)
		}
	}
}

func TestPartitionMicroservicesValidated_StreamRetry(t *testing.T) {
	svc := &Service{config: DefaultConfig(), client: &streamingPartitionChat{
		streamResponses: []string{"不是 JSON。", partitionGood},
	}}

	var attempts []int
	partitions, _, warnings, err := svc.PartitionMicroservicesValidatedStream(testDDL, func(attempt int, _ string) {
		attempts = append(attempts, attempt)
	})
	if err != nil {
		t.Fatalf("重试后应成功: %v", err)
	}
	if len(partitions) != 2 {
		t.Fatalf("重试后应返回两个划分,得到 %d", len(partitions))
	}
	// attempt 序列非递减且含 1、2
	seen := map[int]bool{}
	for i, a := range attempts {
		if i > 0 && a < attempts[i-1] {
			t.Fatalf("attempt 应非递减: %v", attempts)
		}
		seen[a] = true
	}
	if !seen[1] || !seen[2] {
		t.Fatalf("attempt 1 与 2 都应出现: %v", attempts)
	}
	if warnings != nil {
		t.Fatalf("干净划分不应有 warnings: %v", warnings)
	}
}

// 确认流式化后取消链仍生效:流式挂起时取消,应立即返回「已取消」。
func TestPartitionMicroservicesValidated_StreamCancel(t *testing.T) {
	svc := &Service{config: DefaultConfig(), client: &blockingChat{}}

	done := make(chan error, 1)
	go func() {
		_, _, _, err := svc.PartitionMicroservicesValidatedStream(testDDL, nil)
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	if !svc.CancelTask(taskPartition) {
		t.Fatal("在途划分应可取消")
	}
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "已取消") {
			t.Fatalf("应返回已取消,得到: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("取消后应在 5 秒内返回")
	}
}
