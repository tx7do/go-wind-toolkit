package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/redirect"
)

const (
	defaultTimeout = 120 * time.Second
	// streamTimeout 流式请求整体上限:长生成(DDL/代码审查)可能远超普通请求。
	streamTimeout = 5 * time.Minute
)

// chatMessage OpenAI Chat API 消息
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest OpenAI Chat Completion 请求
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// chatResponse OpenAI Chat Completion 响应
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// apiError API 错误响应
type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Client OpenAI 兼容 HTTP 客户端
type Client struct {
	httpClient *http.Client
	config     *Config
}

// NewClient 创建 AI 客户端
func NewClient(config *Config) *Client {
	return &Client{
		// 密钥放在 api-key / x-api-key / x-goog-api-key 这类自定义头里,Go 只在
		// 跨域重定向时剥掉 Authorization 与 Cookie,这些自定义头会照发;307/308
		// 还会把提示词正文一并带走。所以重定向必须限制在同源。
		httpClient: &http.Client{Timeout: defaultTimeout, CheckRedirect: redirect.SameOrigin},
		config:     config,
	}
}

// UpdateConfig 更新配置
func (c *Client) UpdateConfig(config *Config) {
	c.config = config
}

// Chat 发送聊天请求，返回回复内容
func (c *Client) Chat(systemPrompt string, userMessage string) (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}

	body, err := c.postChat(context.Background(), systemPrompt, userMessage)
	if err != nil {
		return "", err
	}

	content, err := c.extractContent(body)
	if err != nil {
		return "", err
	}
	if content == "" {
		return "", fmt.Errorf("API 未返回有效响应")
	}
	return content, nil
}

// chatStreamResponse OpenAI 流式 Chat Completion 的单个 SSE 数据帧
type chatStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatStream 流式发送聊天请求,按 OpenAI 兼容的 SSE 协议接收增量回复。
// 每收到一个增量块就调用一次 onDelta(可为 nil);返回完整内容。
// 流式模式整体超时为 streamTimeout(长生成比普通请求更耗时)。
func (c *Client) ChatStream(systemPrompt string, userMessage string, onDelta func(string)) (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), streamTimeout)
	defer cancel()

	resp, err := c.postChatStream(ctx, systemPrompt, userMessage)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", c.apiError(resp.StatusCode, body)
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue // 空行或 SSE 注释
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}

		delta, err := c.streamDelta([]byte(payload))
		if err != nil {
			return "", err
		}
		if delta == "" {
			continue
		}
		full.WriteString(delta)
		if onDelta != nil {
			onDelta(delta)
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), fmt.Errorf("读取流式响应失败: %w", err)
	}

	content := full.String()
	if content == "" {
		return "", fmt.Errorf("API 未返回有效响应")
	}
	return content, nil
}

// validate 校验发起请求前的必要配置。
func (c *Client) validate() error {
	if c.config == nil {
		return fmt.Errorf("AI 配置未初始化")
	}
	if c.config.BaseURL == "" {
		return fmt.Errorf("API 地址不能为空")
	}
	if c.config.APIKey == "" && c.config.Provider != "ollama" {
		return fmt.Errorf("API 密钥不能为空")
	}
	if c.config.Provider == "azure" && c.config.AzureAPIVersion == "" {
		return fmt.Errorf("Azure 部署需要配置 api-version")
	}
	return nil
}

// postChat 发送非流式请求并返回响应体。
func (c *Client) postChat(ctx context.Context, systemPrompt, userMessage string) ([]byte, error) {
	body, err := c.marshalRequest(systemPrompt, userMessage, false)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.requestTarget(false), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.apiError(resp.StatusCode, data)
	}
	return data, nil
}

// postChatStream 发送流式请求,返回未读取的 SSE 响应,由调用方解析。
func (c *Client) postChatStream(ctx context.Context, systemPrompt, userMessage string) (*http.Response, error) {
	body, err := c.marshalRequest(systemPrompt, userMessage, true)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.requestTarget(true), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	return resp, nil
}

// ==================== 供应商差异收口 ====================
//
// OpenAI 兼容家系(openai/deepseek/ollama/custom/Azure)共用 chatRequest/
// chatResponse/chatStreamResponse 线格式;Anthropic 与 Gemini 各自的原生
// 线格式在下方定义。端点、认证头、请求体、响应提取统一由 requestTarget、
// applyAuth、marshalRequest、extractContent、streamDelta 按供应商分派。

// requestTarget 返回本次请求的完整 URL;stream 区分流式/非流式端点。
func (c *Client) requestTarget(stream bool) string {
	base := strings.TrimRight(c.config.BaseURL, "/")
	switch c.config.Provider {
	case "azure":
		return fmt.Sprintf("%s/deployments/%s/chat/completions?api-version=%s",
			base, url.PathEscape(c.config.Model), url.QueryEscape(c.config.AzureAPIVersion))
	case "anthropic":
		return base + "/v1/messages"
	case "gemini":
		action := "generateContent"
		if stream {
			action = "streamGenerateContent?alt=sse"
		}
		return fmt.Sprintf("%s/v1beta/models/%s:%s", base, url.PathEscape(c.config.Model), action)
	default:
		return base + "/chat/completions"
	}
}

// applyAuth 设置供应商各自的认证头:OpenAI 兼容家系为 Bearer,Azure 为
// api-key,Anthropic 为 x-api-key + anthropic-version,Gemini 为
// x-goog-api-key。
func (c *Client) applyAuth(req *http.Request) {
	if c.config.APIKey == "" {
		return
	}
	switch c.config.Provider {
	case "azure":
		req.Header.Set("api-key", c.config.APIKey)
	case "anthropic":
		req.Header.Set("x-api-key", c.config.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case "gemini":
		req.Header.Set("x-goog-api-key", c.config.APIKey)
	default:
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
}

// setHeaders 设置通用请求头并叠加供应商认证头。
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	c.applyAuth(req)
}

// ---- Anthropic Messages 原生线格式 ----

// anthropicContentBlock Anthropic 消息内容块
type anthropicContentBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// anthropicMessage Anthropic 消息
type anthropicMessage struct {
	Role    string                  `json:"role"` // "user"
	Content []anthropicContentBlock `json:"content"`
}

// anthropicRequest Anthropic Messages 请求
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Stream      bool               `json:"stream,omitempty"`
}

// anthropicResponse Anthropic 非流式响应
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// anthropicStreamFrame Anthropic SSE 事件帧
type anthropicStreamFrame struct {
	Type  string `json:"type"` // content_block_delta / 其他事件
	Delta struct {
		Type string `json:"type"` // text_delta
		Text string `json:"text"`
	} `json:"delta"`
}

// ---- Gemini generateContent 原生线格式 ----

// geminiPart Gemini 内容分片
type geminiPart struct {
	Text string `json:"text"`
}

// geminiContent Gemini 对话轮
type geminiContent struct {
	Role  string       `json:"role"` // "user"
	Parts []geminiPart `json:"parts"`
}

// geminiInstruction Gemini 系统指令
type geminiInstruction struct {
	Parts []geminiPart `json:"parts"`
}

// geminiGenerationConfig Gemini 生成参数
type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// geminiRequest Gemini generateContent 请求
type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction *geminiInstruction     `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
}

// geminiResponse Gemini 响应(流式帧与非流式同形)
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// marshalRequest 按供应商构造请求体。
func (c *Client) marshalRequest(systemPrompt, userMessage string, stream bool) ([]byte, error) {
	var req any
	switch c.config.Provider {
	case "anthropic":
		temperature := c.config.Temperature
		if temperature > 1 {
			temperature = 1 // Anthropic 温度上限 1.0
		}
		req = anthropicRequest{
			Model:       c.config.Model,
			MaxTokens:   c.config.MaxTokens,
			Temperature: temperature,
			System:      systemPrompt,
			Messages: []anthropicMessage{{
				Role:    "user",
				Content: []anthropicContentBlock{{Type: "text", Text: userMessage}},
			}},
			Stream: stream,
		}
	case "gemini":
		g := geminiRequest{
			Contents: []geminiContent{{
				Role:  "user",
				Parts: []geminiPart{{Text: userMessage}},
			}},
			SystemInstruction: &geminiInstruction{
				Parts: []geminiPart{{Text: systemPrompt}},
			},
		}
		g.GenerationConfig.Temperature = c.config.Temperature
		g.GenerationConfig.MaxOutputTokens = c.config.MaxTokens
		req = g
	default: // OpenAI 兼容家系
		req = chatRequest{
			Model:       c.config.Model,
			Messages:    []chatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userMessage}},
			Temperature: c.config.Temperature,
			MaxTokens:   c.config.MaxTokens,
			Stream:      stream,
		}
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	return data, nil
}

// extractContent 从非流式响应体提取文本。
func (c *Client) extractContent(body []byte) (string, error) {
	switch c.config.Provider {
	case "anthropic":
		var resp anthropicResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return "", fmt.Errorf("解析响应失败: %w", err)
		}
		var sb strings.Builder
		for _, blk := range resp.Content {
			if blk.Type == "text" {
				sb.WriteString(blk.Text)
			}
		}
		return sb.String(), nil
	case "gemini":
		var resp geminiResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return "", fmt.Errorf("解析响应失败: %w", err)
		}
		if len(resp.Candidates) == 0 {
			return "", nil
		}
		var sb strings.Builder
		for _, part := range resp.Candidates[0].Content.Parts {
			sb.WriteString(part.Text)
		}
		return sb.String(), nil
	default:
		var chatResp chatResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			return "", fmt.Errorf("解析响应失败: %w", err)
		}
		if len(chatResp.Choices) == 0 {
			return "", nil
		}
		return chatResp.Choices[0].Message.Content, nil
	}
}

// streamDelta 从单个 SSE data 帧提取增量文本。
// 帧无法解析时报错(部分网关在出错时也走 data: 帧返回 JSON 错误对象);
// 合法但非文本增量的帧(心跳/用量/收尾事件)返回空串。
func (c *Client) streamDelta(payload []byte) (string, error) {
	switch c.config.Provider {
	case "anthropic":
		var frame anthropicStreamFrame
		if err := json.Unmarshal(payload, &frame); err != nil {
			return "", fmt.Errorf("解析流式响应帧失败: %w", err)
		}
		if frame.Type == "content_block_delta" && frame.Delta.Type == "text_delta" {
			return frame.Delta.Text, nil
		}
		return "", nil
	case "gemini":
		var frame geminiResponse
		if err := json.Unmarshal(payload, &frame); err != nil {
			return "", fmt.Errorf("解析流式响应帧失败: %w", err)
		}
		if len(frame.Candidates) == 0 {
			return "", nil
		}
		var sb strings.Builder
		for _, part := range frame.Candidates[0].Content.Parts {
			sb.WriteString(part.Text)
		}
		return sb.String(), nil
	default:
		var frame chatStreamResponse
		if err := json.Unmarshal(payload, &frame); err != nil {
			return "", fmt.Errorf("解析流式响应帧失败: %w", err)
		}
		if len(frame.Choices) == 0 {
			return "", nil
		}
		return frame.Choices[0].Delta.Content, nil
	}
}

// apiError 将非 200 响应转换为错误。
func (c *Client) apiError(status int, body []byte) error {
	var apiErr apiError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
		return fmt.Errorf("API 错误 (%d): %s", status, apiErr.Error.Message)
	}
	return fmt.Errorf("API 错误 (%d): %s", status, string(body))
}

// TestConnection 测试 AI 连接
func (c *Client) TestConnection() (*StepResult, error) {
	content, err := c.Chat(
		"You are a helpful assistant. Reply with exactly: CONNECTION_OK",
		"Hello, please respond with CONNECTION_OK to confirm the connection is working.",
	)
	if err != nil {
		return &StepResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &StepResult{
		Success: true,
		Content: content,
	}, nil
}
