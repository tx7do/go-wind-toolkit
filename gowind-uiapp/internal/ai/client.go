package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 120 * time.Second
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
		httpClient: &http.Client{Timeout: defaultTimeout},
		config:     config,
	}
}

// UpdateConfig 更新配置
func (c *Client) UpdateConfig(config *Config) {
	c.config = config
}

// Chat 发送聊天请求，返回回复内容
func (c *Client) Chat(systemPrompt string, userMessage string) (string, error) {
	if c.config == nil {
		return "", fmt.Errorf("AI 配置未初始化")
	}

	if c.config.BaseURL == "" {
		return "", fmt.Errorf("API 地址不能为空")
	}

	if c.config.APIKey == "" && c.config.Provider != "ollama" {
		return "", fmt.Errorf("API 密钥不能为空")
	}

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	reqBody := chatRequest{
		Model:       c.config.Model,
		Messages:    messages,
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	url := strings.TrimRight(c.config.BaseURL, "/") + "/chat/completions"

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr apiError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			return "", fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return "", fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API 未返回有效响应")
	}

	return chatResp.Choices[0].Message.Content, nil
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
