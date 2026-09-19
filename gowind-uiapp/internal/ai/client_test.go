package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatStreamParsesSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		frames := []string{
			`data: {"choices":[{"delta":{"content":"CREATE "}}]}`,
			``,
			`: keep-alive comment`,
			`data: {"choices":[{"delta":{"content":"TABLE users"}}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
			`data: {"choices":[{"delta":{"content":"MUST NOT APPEAR"}}]}`,
		}
		for _, f := range frames {
			_, _ = w.Write([]byte(f + "\n\n"))
			flusher.Flush()
		}
	}))
	defer srv.Close()

	var deltas []string
	client := NewClient(&Config{
		Provider: "openai",
		BaseURL:  srv.URL,
		APIKey:   "test-key",
		Model:    "test-model",
	})
	content, err := client.ChatStream("sys", "user", func(d string) { deltas = append(deltas, d) })
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if content != "CREATE TABLE users" {
		t.Fatalf("content = %q, want %q", content, "CREATE TABLE users")
	}
	if strings.Join(deltas, "") != content {
		t.Fatalf("deltas %v do not assemble to content %q", deltas, content)
	}
}

func TestChatStreamReportsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key","type":"auth"}}`))
	}))
	defer srv.Close()

	client := NewClient(&Config{Provider: "openai", BaseURL: srv.URL, APIKey: "bad", Model: "m"})
	_, err := client.ChatStream("sys", "user", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("expected API error, got %v", err)
	}
}

func TestChatNonStreamStillWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hi"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	client := NewClient(&Config{Provider: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m"})
	content, err := client.Chat("sys", "user")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "hi" {
		t.Fatalf("content = %q", content)
	}
}

func TestConfigPersistRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// os.UserConfigDir 各平台取值不同:Windows=%AppData%,Linux=$XDG_CONFIG_HOME
	// (未设时回落 $HOME/.config),macOS=$HOME/Library/Application Support 且忽略 XDG。
	// 三者指向同一临时目录,才能在所有平台把配置隔离到测试沙箱。
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	saved := &Config{Provider: "deepseek", BaseURL: "https://api.deepseek.com/v1", APIKey: "sk-x", AzureAPIVersion: "2024-02-01", Model: "deepseek-chat", Temperature: 0.3, MaxTokens: 2048}
	if err := SaveConfig(saved); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded := LoadConfig()
	if loaded.Provider != "deepseek" || loaded.Model != "deepseek-chat" || loaded.APIKey != "sk-x" {
		t.Fatalf("loaded config mismatch: %+v", loaded)
	}
	if loaded.AzureAPIVersion != "2024-02-01" {
		t.Fatalf("azure api version round trip failed: %+v", loaded)
	}
	if loaded.Temperature != 0.3 || loaded.MaxTokens != 2048 {
		t.Fatalf("numeric fields mismatch: %+v", loaded)
	}
}

func TestLoadConfigFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	// 见 TestConfigPersistRoundTrip:三个平台的 UserConfigDir 取值都要隔离,
	// macOS 走 $HOME 而非 XDG_CONFIG_HOME。
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	cfg := LoadConfig()
	def := DefaultConfig()
	if cfg.Provider != def.Provider || cfg.Model != def.Model {
		t.Fatalf("missing file should fall back to defaults, got %+v", cfg)
	}
}

// ---- Azure / Anthropic / Gemini 供应商适配 ----

func TestAzureChatUsesDeploymentPathAndApiKey(t *testing.T) {
	var gotPath, gotVersion, gotKey, gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotVersion = r.URL.Query().Get("api-version")
		gotKey = r.Header.Get("api-key")
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"azure-ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	client := NewClient(&Config{
		Provider:        "azure",
		BaseURL:         srv.URL,
		APIKey:          "test-key",
		AzureAPIVersion: "2024-02-01",
		Model:           "dep-x",
	})
	content, err := client.Chat("sys", "user")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "azure-ok" {
		t.Fatalf("content = %q", content)
	}
	if gotPath != "/deployments/dep-x/chat/completions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotVersion != "2024-02-01" {
		t.Fatalf("api-version = %q", gotVersion)
	}
	if gotKey != "test-key" || gotAuth != "" {
		t.Fatalf("auth headers: api-key=%q Authorization=%q", gotKey, gotAuth)
	}
	msgs, _ := gotBody["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("messages len = %d, want 2 (system+user)", len(msgs))
	}
}

func TestAzureRequiresApiVersion(t *testing.T) {
	client := NewClient(&Config{Provider: "azure", BaseURL: "https://example.invalid", APIKey: "k", Model: "d"})
	if _, err := client.Chat("sys", "user"); err == nil || !strings.Contains(err.Error(), "api-version") {
		t.Fatalf("expected api-version error, got %v", err)
	}
}

func TestAnthropicRequestShapeAndNonStream(t *testing.T) {
	var gotPath, gotKey, gotVersion string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"text","text":"anthropic-ok"}]}`))
	}))
	defer srv.Close()

	client := NewClient(&Config{
		Provider:    "anthropic",
		BaseURL:     srv.URL,
		APIKey:      "sk-ant",
		Model:       "claude-3-5-sonnet",
		Temperature: 1.5, // 超出 Anthropic 上限,应被钳制到 1.0
		MaxTokens:   4096,
	})
	content, err := client.Chat("sys-prompt", "user-msg")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "anthropic-ok" {
		t.Fatalf("content = %q", content)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotKey != "sk-ant" || gotVersion != "2023-06-01" {
		t.Fatalf("headers: x-api-key=%q anthropic-version=%q", gotKey, gotVersion)
	}
	if gotBody["system"] != "sys-prompt" {
		t.Fatalf("system = %v", gotBody["system"])
	}
	msgList, _ := gotBody["messages"].([]any)
	if len(msgList) != 1 {
		t.Fatalf("messages len = %d", len(msgList))
	}
	msg, _ := msgList[0].(map[string]any)
	blocks, _ := msg["content"].([]any)
	if len(blocks) != 1 {
		t.Fatalf("content blocks = %d", len(blocks))
	}
	blk, _ := blocks[0].(map[string]any)
	if blk["type"] != "text" || blk["text"] != "user-msg" {
		t.Fatalf("block = %v", blk)
	}
	if gotBody["max_tokens"] != float64(4096) {
		t.Fatalf("max_tokens = %v", gotBody["max_tokens"])
	}
	if gotBody["temperature"] != float64(1) {
		t.Fatalf("temperature = %v, want clamped 1", gotBody["temperature"])
	}
	if _, has := gotBody["stream"]; has {
		t.Fatal("non-stream request must not set stream")
	}
}

func TestAnthropicStreamAssemblesTextDeltas(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		lines := []string{
			"event: message_start",
			`data: {"type":"message_start","message":{}}`,
			"",
			"event: content_block_delta",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"CREATE "}}`,
			"",
			"event: content_block_delta",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"TABLE users"}}`,
			"",
			"event: message_delta",
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
			"",
			"event: message_stop",
			`data: {"type":"message_stop"}`,
			"",
		}
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
			flusher.Flush()
		}
	}))
	defer srv.Close()

	client := NewClient(&Config{Provider: "anthropic", BaseURL: srv.URL, APIKey: "k", Model: "claude-x"})
	content, err := client.ChatStream("sys", "user", nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q", gotPath)
	}
	if content != "CREATE TABLE users" {
		t.Fatalf("content = %q", content)
	}
}

func TestGeminiRequestShapeAndNonStream(t *testing.T) {
	var gotPath, gotKey string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-goog-api-key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"gemini-ok"}],"role":"model"}}]}`))
	}))
	defer srv.Close()

	client := NewClient(&Config{
		Provider:    "gemini",
		BaseURL:     srv.URL,
		APIKey:      "goog-key",
		Model:       "gemini-1.5-pro",
		Temperature: 0.7,
		MaxTokens:   4096,
	})
	content, err := client.Chat("sys-prompt", "user-msg")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "gemini-ok" {
		t.Fatalf("content = %q", content)
	}
	if gotPath != "/v1beta/models/gemini-1.5-pro:generateContent" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotKey != "goog-key" {
		t.Fatalf("x-goog-api-key = %q", gotKey)
	}
	contentList, _ := gotBody["contents"].([]any)
	if len(contentList) != 1 {
		t.Fatalf("contents len = %d", len(contentList))
	}
	c, _ := contentList[0].(map[string]any)
	parts, _ := c["parts"].([]any)
	if len(parts) != 1 {
		t.Fatalf("parts len = %d", len(parts))
	}
	p, _ := parts[0].(map[string]any)
	if p["text"] != "user-msg" {
		t.Fatalf("part text = %v", p["text"])
	}
	instr, _ := gotBody["systemInstruction"].(map[string]any)
	if instr == nil {
		t.Fatal("systemInstruction missing")
	}
	iparts, _ := instr["parts"].([]any)
	if len(iparts) != 1 {
		t.Fatalf("instruction parts len = %d", len(iparts))
	}
	ip, _ := iparts[0].(map[string]any)
	if ip["text"] != "sys-prompt" {
		t.Fatalf("instruction text = %v", ip["text"])
	}
	gc, _ := gotBody["generationConfig"].(map[string]any)
	if gc["temperature"] != float64(0.7) {
		t.Fatalf("temperature = %v", gc["temperature"])
	}
	if gc["maxOutputTokens"] != float64(4096) {
		t.Fatalf("maxOutputTokens = %v", gc["maxOutputTokens"])
	}
}

func TestGeminiStreamAssemblesTextParts(t *testing.T) {
	var gotPath, gotAlt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAlt = r.URL.Query().Get("alt")
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		frames := []string{
			`data: {"candidates":[{"content":{"parts":[{"text":"CREATE "}]}}]}`,
			``,
			`data: {"candidates":[{"content":{"parts":[{"text":"TABLE users"}]}}]}`,
			``,
			`data: {"usageMetadata":{"promptTokenCount":1}}`, // 无 candidates 的收尾帧
			``,
		}
		for _, f := range frames {
			_, _ = w.Write([]byte(f + "\n"))
			flusher.Flush()
		}
	}))
	defer srv.Close()

	client := NewClient(&Config{Provider: "gemini", BaseURL: srv.URL, APIKey: "k", Model: "gemini-x"})
	content, err := client.ChatStream("sys", "user", nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if gotPath != "/v1beta/models/gemini-x:streamGenerateContent" || gotAlt != "sse" {
		t.Fatalf("path = %q alt = %q", gotPath, gotAlt)
	}
	if content != "CREATE TABLE users" {
		t.Fatalf("content = %q", content)
	}
}
