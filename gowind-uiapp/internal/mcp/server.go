// Package mcp 把 gowind-uiapp 的 AI 能力以 MCP(Model Context Protocol)
// stdio 服务模式暴露,供 ZCode/Claude/Cursor 等客户端作为工具调用。
//
// 实现纪律:
//   - 手写最小 JSON-RPC 2.0(仅 initialize/tools/* /ping),不引第三方 SDK;
//   - stdout 只走协议帧,一切日志进 stderr;
//   - 传输与逻辑分离:HandleMessage 是纯函数,可直接喂帧测试。
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/ai"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
)

const (
	// defaultProtocolVersion 客户端未携带版本时的兜底;实际响应回显客户端请求值。
	defaultProtocolVersion = "2025-06-18"
	serverName             = "gowind-uiapp"
	serverVersion          = "0.1.0"
)

// Server MCP 服务实例:工具注册表 + 单份后端生成计划状态。
type Server struct {
	aiService *ai.Service
	gen       *generator.Generator

	// mu 保护 plan:tools/call 串行处理,防御性持锁。
	mu   sync.Mutex
	plan *backendPlanState

	// ctx 允许测试注入可取消背景(生产用 context.Background)。
	ctx context.Context
}

// backendPlanState plan_backend/apply_backend 的单份计划(与 GUI 侧 App 同构)。
// root 是项目根目录(拷贝目标),modPath 是 Go 模块路径(生成 import 用)。
type backendPlanState struct {
	id       string
	dir      string
	services []string
	root     string
	modPath  string
	ormType  string
}

// NewServer 生产构造:AI 服务按持久化配置+GOWIND_AI_* 环境变量装配。
func NewServer() *Server {
	return &Server{
		aiService: newEnvAIService(),
		gen:       generator.NewGenerator(),
		ctx:       context.Background(),
	}
}

// NewServerWithService 测试构造:注入指定 AI 服务(mock 经 httptest 指入)。
func NewServerWithService(svc *ai.Service) *Server {
	return &Server{
		aiService: svc,
		gen:       generator.NewGenerator(),
		ctx:       context.Background(),
	}
}

// ---- JSON-RPC 最小结构 ----

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		// 结构全是本包可控的 map/slice,理论上不可达;兜底成合法帧
		return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"内部序列化失败"}}`)
	}
	return data
}

func okResult(id json.RawMessage, result any) []byte {
	return mustJSON(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func errResult(id json.RawMessage, code int, message string) []byte {
	return mustJSON(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}

// HandleMessage 处理一帧原始 JSON,返回响应帧与是否存在响应
// (notification 无 id,不回帧)。逻辑与传输分离,测试可直接喂帧。
func (s *Server) HandleMessage(raw []byte) ([]byte, bool) {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return errResult(json.RawMessage("null"), -32700, "解析错误: "+err.Error()), true
	}

	// notification(无 id):仅初始化通知有意义,其余一律静默
	if len(req.ID) == 0 || string(req.ID) == "null" {
		return nil, false
	}

	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &params)
		version := params.ProtocolVersion
		if version == "" {
			version = defaultProtocolVersion
		}
		return okResult(req.ID, map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": serverName, "version": serverVersion},
		}), true

	case "ping":
		return okResult(req.ID, map[string]any{}), true

	case "tools/list":
		return okResult(req.ID, map[string]any{"tools": toolDefinitions()}), true

	case "tools/call":
		return s.handleToolsCall(req.ID, req.Params)

	default:
		return errResult(req.ID, -32601, "未知方法: "+req.Method), true
	}
}

// ---- tools/call ----

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) handleToolsCall(id json.RawMessage, params json.RawMessage) ([]byte, bool) {
	var p toolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return errResult(id, -32602, "params 解析失败: "+err.Error()), true
	}

	text, isError := s.invokeTool(p.Name, p.Arguments)
	result := map[string]any{
		"content": []toolContent{{Type: "text", Text: text}},
		"isError": isError,
	}
	return okResult(id, result), true
}

// invokeTool 分发到具体工具 handler,panic 与错误统一转 isError 文本。
func (s *Server) invokeTool(name string, args json.RawMessage) (text string, isError bool) {
	defer func() {
		if r := recover(); r != nil {
			// 覆写命名返回值:panic 也走 isError 文本而非崩掉整个服务
			text = fmt.Sprintf("工具内部错误: %v", r)
			isError = true
		}
	}()
	handler, ok := toolHandlers()[name]
	if !ok {
		return "未知工具: " + name, true
	}
	result, err := handler(s, args)
	if err != nil {
		data, _ := json.Marshal(map[string]any{"error": err.Error()})
		return string(data), true
	}
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("结果序列化失败: %v", err), true
	}
	return string(data), false
}

// ---- stdio 主循环 ----

// Run 从 in 逐行读协议帧,响应写 out(每帧一行)。stdin 关闭即退出。
func (s *Server) Run(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		resp, has := s.HandleMessage([]byte(line))
		if !has {
			continue
		}
		if _, err := out.Write(append(resp, '\n')); err != nil {
			return err
		}
	}
	return scanner.Err()
}
