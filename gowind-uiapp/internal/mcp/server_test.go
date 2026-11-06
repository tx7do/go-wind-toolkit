package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/ai"
)

// ---- 协议帧 ----

func TestHandleMessage_Initialize(t *testing.T) {
	s := NewServer()
	resp, has := s.HandleMessage([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`))
	if !has {
		t.Fatal("initialize 应有响应")
	}
	if !strings.Contains(string(resp), `"protocolVersion":"2025-06-18"`) {
		t.Fatalf("应回显客户端协议版本: %s", resp)
	}
	if !strings.Contains(string(resp), `"tools":{}`) || !strings.Contains(string(resp), serverName) {
		t.Fatalf("应声明 tools 能力与服务器名: %s", resp)
	}
}

func TestHandleMessage_InitializeEmptyVersion(t *testing.T) {
	s := NewServer()
	resp, _ := s.HandleMessage([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	if !strings.Contains(string(resp), defaultProtocolVersion) {
		t.Fatalf("缺版本应回退默认: %s", resp)
	}
}

func TestHandleMessage_NotificationSilent(t *testing.T) {
	s := NewServer()
	for _, raw := range []string{
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"whatever/notification","params":{}}`,
	} {
		if resp, has := s.HandleMessage([]byte(raw)); has {
			t.Fatalf("notification 不应有响应: %s -> %s", raw, resp)
		}
	}
}

func TestHandleMessage_ToolsList(t *testing.T) {
	s := NewServer()
	resp, _ := s.HandleMessage([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	defs := toolDefinitions()
	if len(defs) != 8 {
		t.Fatalf("应有 8 个工具,得到 %d", len(defs))
	}
	for _, want := range []string{"generate_ddl", "validate_ddl", "partition_microservices", "plan_backend", "apply_backend", "review_files"} {
		if !strings.Contains(string(resp), `"`+want+`"`) {
			t.Fatalf("tools/list 应包含 %s: %.200s", want, resp)
		}
	}
	if !strings.Contains(string(resp), "inputSchema") {
		t.Fatal("工具应带 inputSchema")
	}
}

func TestHandleMessage_UnknownMethod(t *testing.T) {
	s := NewServer()
	resp, _ := s.HandleMessage([]byte(`{"jsonrpc":"2.0","id":3,"method":"no/such"}`))
	if !strings.Contains(string(resp), "-32601") {
		t.Fatalf("未知方法应回 -32601: %s", resp)
	}
}

func TestHandleMessage_MalformedFrame(t *testing.T) {
	s := NewServer()
	resp, _ := s.HandleMessage([]byte(`not-json`))
	if !strings.Contains(string(resp), "-32700") {
		t.Fatalf("坏帧应回 -32700: %s", resp)
	}
}

// ---- LLM mock:httptest 起 OpenAI 兼容端点,按请求顺序回预设体 ----

// mockLLM 顺序响应的服务器。chatResponses 用于非流式,streamResponses 用于流式
// (SSE)。流式判定读请求体里的 "stream":true(OpenAI 兼容端点同路径)。
func mockLLM(t *testing.T, chatResponses []string, streamResponses []string) *httptest.Server {
	t.Helper()
	i, j := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 32*1024)
		n, _ := r.Body.Read(buf)
		isStream := strings.Contains(string(buf[:n]), `"stream":true`)

		w.Header().Set("Content-Type", "application/json")
		if isStream {
			body := "{}"
			if j < len(streamResponses) {
				body = streamResponses[j]
				j++
			}
			frame := `{"choices":[{"delta":{"content":` + quoteJSON(body) + `}}]}`
			fmt.Fprintf(w, "data: %s\n\n", frame)
			fmt.Fprint(w, "data: [DONE]\n")
			return
		}
		body := "{}"
		if i < len(chatResponses) {
			body = chatResponses[i]
			i++
		}
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%s}}]}`, quoteJSON(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func quoteJSON(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

func newTestServer(t *testing.T, chatResponses []string, streamResponses []string) *Server {
	t.Helper()
	srv := mockLLM(t, chatResponses, streamResponses)
	svc := ai.NewService()
	svc.SetConfigTransient(&ai.Config{
		Provider: "custom",
		BaseURL:  srv.URL,
		APIKey:   "test-key",
		Model:    "test-model",
	})
	return NewServerWithService(svc)
}

// testDDLLocal 最小合法 DDL(mcp 包内自足,不跨包引测试常量)。
const testDDLLocal = "CREATE TABLE `user` (\n  `id` BIGINT NOT NULL AUTO_INCREMENT,\n  `name` VARCHAR(50) NOT NULL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户';"

func callTool(s *Server, t *testing.T, name string, args string) (string, bool) {
	t.Helper()
	frame := fmt.Sprintf(`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":%s,"arguments":%s}}`, quoteJSON(name), args)
	resp, has := s.HandleMessage([]byte(frame))
	if !has {
		t.Fatalf("tools/call 应有响应")
	}
	var parsed struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		t.Fatalf("响应解析失败: %v\n%s", err, resp)
	}
	if len(parsed.Result.Content) == 0 {
		t.Fatalf("响应缺 content: %s", resp)
	}
	return parsed.Result.Content[0].Text, parsed.Result.IsError
}

// ---- 工具 ----

func TestTool_ValidateDDL(t *testing.T) {
	s := NewServerWithService(ai.NewService()) // 校验不调 LLM,服务不触网

	text, isErr := callTool(s, t, "validate_ddl", `{"sql":"CREATE TABLE `+"`user`"+` (id BIGINT PRIMARY KEY);"}`)
	if isErr {
		t.Fatalf("合法 DDL 不应报错: %s", text)
	}
	if !strings.Contains(text, `"success":true`) || !strings.Contains(text, "user") {
		t.Fatalf("结果不符: %s", text)
	}

	// 坏 DDL:isError=false(诊断是数据不是错误),success=false + diagnostics 带行号
	text, isErr = callTool(s, t, "validate_ddl", `{"sql":"CREATE TABLE broken (id BIGINT"}`)
	if isErr {
		t.Fatalf("校验诊断不应走 isError 通道: %s", text)
	}
	var parsed struct {
		Success     bool `json:"success"`
		Diagnostics []struct {
			Line int `json:"line"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("结果解析失败: %v\n%s", err, text)
	}
	if parsed.Success || len(parsed.Diagnostics) == 0 || parsed.Diagnostics[0].Line < 1 {
		t.Fatalf("应为 success=false 且诊断带行号: %s", text)
	}
}

func TestTool_GenerateDDL(t *testing.T) {
	s := newTestServer(t, nil, []string{testDDLLocal}) // GenerateDDL 内部走 ChatStream

	text, isErr := callTool(s, t, "generate_ddl", `{"requirements":"用户管理"}`)
	if isErr {
		t.Fatalf("生成不应报错: %s", text)
	}
	if !strings.Contains(text, `"success":true`) {
		t.Fatalf("结果不符: %.120s", text)
	}

	// 空需求走 isError
	text, isErr = callTool(s, t, "generate_ddl", `{"requirements":"  "}`)
	if !isErr || !strings.Contains(text, "requirements 不能为空") {
		t.Fatalf("空需求应报错: %s, %v", text, isErr)
	}
}

func TestTool_PartitionMicroservices(t *testing.T) {
	partitionJSON := `{"summary":"ok","issues":[]}` // 不会被用到
	_ = partitionJSON
	partitionResp := `[{"serviceName":"user","tables":["user"],"description":"用户"}]`
	s := newTestServer(t, nil, []string{partitionResp}) // 划分已走流式端点

	text, isErr := callTool(s, t, "partition_microservices", `{"ddl":`+quoteJSON(testDDLLocal)+`}`)
	if isErr {
		t.Fatalf("划分不应报错: %s", text)
	}
	if !strings.Contains(text, "availableTables") || !strings.Contains(text, "user") {
		t.Fatalf("结果应含表清单: %.160s", text)
	}
}

func TestTool_PlanAndApplyBackend(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 预置一个将被覆盖的文件
	pre := filepath.Join(root, "app", "user", "service", "Makefile")
	if err := os.MkdirAll(filepath.Dir(pre), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pre, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServerWithService(ai.NewService()) // plan 不调 LLM
	rootJSON := quoteJSON(root)
	partitions := `[{"serviceName":"user","tables":["user"],"description":"用户"}]`

	args := fmt.Sprintf(`{"projectRoot":%s,"ddl":%s,"ormType":"ent","partitions":%s}`,
		rootJSON, quoteJSON("CREATE TABLE `user` (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(50) NOT NULL);"), partitions)

	text, isErr := callTool(s, t, "plan_backend", args)
	if isErr {
		t.Fatalf("plan 失败: %s", text)
	}
	var planRes struct {
		PlanID         string `json:"planId"`
		NewCount       int    `json:"newCount"`
		OverwriteCount int    `json:"overwriteCount"`
	}
	if err := json.Unmarshal([]byte(text), &planRes); err != nil {
		t.Fatalf("plan 结果解析失败: %v\n%s", err, text)
	}
	if planRes.PlanID == "" || planRes.OverwriteCount < 1 {
		t.Fatalf("planId 与覆盖标注不符: %+v", planRes)
	}
	// 预览不写项目
	if data, _ := os.ReadFile(pre); string(data) != "old" {
		t.Fatal("预览阶段不应写入项目")
	}

	// 过期 planId 拒绝
	text, isErr = callTool(s, t, "apply_backend", `{"planId":"bogus"}`)
	if !isErr || !strings.Contains(text, "不存在或已过期") {
		t.Fatalf("过期 planId 应拒绝: %s", text)
	}

	// 正确 apply:拷贝完成,但 buf 不在(无 api 目录)→ 注明"文件已写入"的失败
	text, isErr = callTool(s, t, "apply_backend", fmt.Sprintf(`{"planId":%s}`, quoteJSON(planRes.PlanID)))
	if !isErr || !strings.Contains(text, "文件已写入") {
		t.Fatalf("apply 应报带\"文件已写入\"前缀的错误: %s", text)
	}
	if data, rerr := os.ReadFile(pre); rerr != nil || string(data) == "old" {
		t.Fatalf("apply 后 Makefile 应被覆盖: read=%v content=%.60q apply返回=%s", rerr, data, text)
	}

	// 计划已清理:再次 apply 同 planId 拒绝
	text, isErr = callTool(s, t, "apply_backend", fmt.Sprintf(`{"planId":%s}`, quoteJSON(planRes.PlanID)))
	if !isErr {
		t.Fatalf("应用后计划应清理: %s", text)
	}
}

func TestTool_UnknownTool(t *testing.T) {
	s := NewServer()
	text, isErr := callTool(s, t, "no_such_tool", `{}`)
	if !isErr || !strings.Contains(text, "未知工具") {
		t.Fatalf("未知工具应报错: %s", text)
	}
}
