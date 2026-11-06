package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockChat 预设响应序列的对话客户端桩:Chat/ChatStream 共用同一响应队列,
// prompts 记录每次调用的完整 userMessage,供重试提示断言使用。ctx 被忽略。
type mockChat struct {
	responses []string
	i         int
	prompts   []string
	chunkN    int // >0 时把每个响应切成 chunkN 份模拟流式增量
}

func (m *mockChat) pop(_ context.Context, userMessage string) (string, error) {
	m.prompts = append(m.prompts, userMessage)
	if m.i >= len(m.responses) {
		return "", fmt.Errorf("mock 响应已耗尽")
	}
	resp := m.responses[m.i]
	m.i++
	return resp, nil
}

func (m *mockChat) Chat(_ context.Context, systemPrompt string, userMessage string) (string, error) {
	return m.pop(context.Background(), userMessage)
}

func (m *mockChat) ChatStream(_ context.Context, systemPrompt string, userMessage string, onDelta func(string)) (string, error) {
	resp, err := m.pop(context.Background(), userMessage)
	if err != nil {
		return "", err
	}
	if onDelta != nil {
		if m.chunkN <= 1 {
			onDelta(resp)
		} else {
			size := (len(resp) + m.chunkN - 1) / m.chunkN
			for start := 0; start < len(resp); start += size {
				end := start + size
				if end > len(resp) {
					end = len(resp)
				}
				onDelta(resp[start:end])
			}
		}
	}
	return resp, nil
}

func (m *mockChat) TestConnection(_ context.Context) (*StepResult, error) {
	return &StepResult{Success: true}, nil
}

func newTestService(responses ...string) (*Service, *mockChat) {
	mock := &mockChat{responses: responses, chunkN: 2}
	return &Service{config: DefaultConfig(), client: mock}, mock
}

// ddlFail 不能通过解析校验的 LLM 输出(纯解释文字,没有 CREATE TABLE)。
const ddlFail = "抱歉,这里没有可解析的建表语句。"

func TestGenerateDDLValidated_RetrySucceeds(t *testing.T) {
	svc, mock := newTestService(ddlFail, testDDL)

	var deltas []string
	attempts := []int{}
	result, err := svc.GenerateDDLValidated("用户管理", func(attempt int, delta string) {
		attempts = append(attempts, attempt)
		deltas = append(deltas, delta)
	})
	if err != nil {
		t.Fatalf("第二次应成功: %v", err)
	}
	if !result.Success || result.Content != testDDL {
		t.Fatalf("最终内容应为第二次响应,得到: %.80s", result.Content)
	}
	if mock.i != 2 {
		t.Fatalf("应调用两次 ChatStream,实际 %d 次", mock.i)
	}
	// 增量的 attempt 必须非递减且两个尝试都出现过——前端据此清空重试前的累积。
	seen := map[int]bool{}
	for i, a := range attempts {
		if i > 0 && a < attempts[i-1] {
			t.Fatalf("attempt 序列应非递减,在 %d 处回退: %v", i, attempts)
		}
		seen[a] = true
	}
	if !seen[1] || !seen[2] {
		t.Fatalf("attempt 1 与 2 都应出现,得到: %v", attempts)
	}
	// 第二次调用的提示词必须带回解析错误与被否决的产物。
	if !strings.Contains(mock.prompts[1], "failed to parse") || !strings.Contains(mock.prompts[1], ddlFail) {
		t.Fatalf("修复重试提示应包含被否决的 DDL,得到: %.120s", mock.prompts[1])
	}
}

func TestGenerateDDLValidated_AllAttemptsFail(t *testing.T) {
	svc, _ := newTestService(ddlFail, ddlFail, ddlFail)

	result, err := svc.GenerateDDLValidated("用户管理", nil)
	if err == nil {
		t.Fatal("全部尝试失败时应返回错误")
	}
	if result.Success {
		t.Fatal("result.Success 应为 false")
	}
	if !strings.Contains(result.Error, "3 次") {
		t.Fatalf("错误应说明尝试次数,得到: %s", result.Error)
	}
}

func TestGenerateDDLValidated_TransportErrorNoRetry(t *testing.T) {
	// 响应队列只有一条且会耗尽:若传输错误被重试,第二次调用会得到"mock 响应已耗尽"
	// 这个不同的错误。断言错误就是第一次的那个,证明未重试。
	svc, _ := newTestService()
	svc.client = &exhaustedMock{}

	result, err := svc.GenerateDDLValidated("用户管理", nil)
	if err == nil || !strings.Contains(err.Error(), "连接被拒") {
		t.Fatalf("应直接透传传输层错误,得到: %v", err)
	}
	if result.Success {
		t.Fatal("result.Success 应为 false")
	}
}

// exhaustedMock 永远返回同一传输层错误的桩。
type exhaustedMock struct{}

func (e *exhaustedMock) Chat(context.Context, string, string) (string, error) {
	return "", fmt.Errorf("连接被拒")
}
func (e *exhaustedMock) ChatStream(context.Context, string, string, func(string)) (string, error) {
	return "", fmt.Errorf("连接被拒")
}
func (e *exhaustedMock) TestConnection(context.Context) (*StepResult, error) {
	return nil, fmt.Errorf("连接被拒")
}

func TestGenerateDDLStream_WrapsAttempt(t *testing.T) {
	svc, _ := newTestService(testDDL)

	var streamed strings.Builder
	result, err := svc.GenerateDDLStream("用户管理", func(delta string) {
		streamed.WriteString(delta)
	})
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if !result.Success || result.Content != testDDL {
		t.Fatalf("返回内容应与流式累积一致")
	}
	if streamed.String() != testDDL {
		t.Fatalf("流式增量拼接应还原完整内容")
	}
}

// partitionBad 无效划分:一个服务混入幻觉表,一个服务名为空,一个服务重复认领。
const partitionBad = `[
  {"serviceName": "user", "tables": ["user", "user_role", "ghost_table"], "description": "用户"},
  {"serviceName": "", "tables": ["order"], "description": "无名"},
  {"serviceName": "order", "tables": ["order", "` + "`Order`" + `"], "description": "订单"}
]`

// partitionGood 干净划分,作为第二次响应。
const partitionGood = `[
  {"serviceName": "user", "tables": ["user", "user_role"], "description": "用户"},
  {"serviceName": "order", "tables": ["order"], "description": "订单"}
]`

func TestPartitionMicroservicesValidated_Sanitize(t *testing.T) {
	svc, _ := newTestService(partitionBad)

	partitions, ddlTables, warnings, err := svc.PartitionMicroservicesValidated(testDDL)
	if err != nil {
		t.Fatalf("清洗后仍有有效划分,不应报错: %v", err)
	}
	if len(partitions) != 2 {
		t.Fatalf("应剩 2 个有效划分,得到 %d: %+v", len(partitions), partitions)
	}
	if got := strings.Join(partitions[0].Tables, ","); got != "user,user_role" {
		t.Fatalf("幻觉表应被剔除,得到 %q", got)
	}
	// 反引号+大写的 `Order` 应归一化匹配并以 DDL 原拼写回填。
	if partitions[1].Tables[0] != "order" {
		t.Fatalf("表名应回填 DDL 原拼写,得到 %q", partitions[1].Tables[0])
	}
	// DDL 全表清单应透传,供前端未分配池使用。
	if strings.Join(ddlTables, ",") != "user,user_role,order" {
		t.Fatalf("AvailableTables 应为 DDL 原始表清单,得到 %v", ddlTables)
	}
	joined := strings.Join(warnings, ";")
	for _, want := range []string{"ghost_table", "服务名非法", "已归属服务"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("warnings 应包含 %q,得到: %s", want, joined)
		}
	}
}

func TestPartitionMicroservicesValidated_ParseRetry(t *testing.T) {
	svc, mock := newTestService("这不是 JSON。", partitionGood)

	partitions, _, _, err := svc.PartitionMicroservicesValidated(testDDL)
	if err != nil {
		t.Fatalf("重试后应成功: %v", err)
	}
	if len(partitions) != 2 || mock.i != 2 {
		t.Fatalf("应重试一次并返回两个划分,得到 %d 个划分、%d 次调用", len(partitions), mock.i)
	}
	if !strings.Contains(mock.prompts[1], "has a problem") {
		t.Fatalf("第二次提示应包含问题反馈,得到: %.120s", mock.prompts[1])
	}
}

func TestPartitionMicroservicesValidated_TwiceInvalid(t *testing.T) {
	svc, _ := newTestService("仍然不是 JSON。", "还是不是 JSON。")

	if _, _, _, err := svc.PartitionMicroservicesValidated(testDDL); err == nil {
		t.Fatal("两次都无效时应返回错误")
	}
}

func TestPartitionMicroservicesValidated_InvalidDDL(t *testing.T) {
	svc, _ := newTestService(partitionGood)

	if _, _, _, err := svc.PartitionMicroservicesValidated("SELECT 1;"); err == nil {
		t.Fatal("DDL 本身不可解析时应直接报错")
	}
}

func TestPartitionMicroservices_CompatWrapper(t *testing.T) {
	svc, _ := newTestService(partitionBad)

	partitions, err := svc.PartitionMicroservices(testDDL)
	if err != nil {
		t.Fatalf("旧签名包装不应报错: %v", err)
	}
	if len(partitions) != 2 {
		t.Fatalf("应返回清洗后的划分,得到 %d 个", len(partitions))
	}
}
