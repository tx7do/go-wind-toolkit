package ai

// Config AI 服务配置
type Config struct {
	Provider    string  `json:"provider"`    // "openai", "azure", "ollama", "deepseek", "custom"
	BaseURL     string  `json:"baseUrl"`     // API 基础地址
	APIKey      string  `json:"apiKey"`      // API 密钥
	Model       string  `json:"model"`       // 模型名称
	Temperature float64 `json:"temperature"` // 温度参数 (0.0-2.0)
	MaxTokens   int     `json:"maxTokens"`   // 最大 token 数
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Provider:    "openai",
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4o",
		Temperature: 0.7,
		MaxTokens:   4096,
	}
}

// MicroservicePartition 微服务划分建议
type MicroservicePartition struct {
	ServiceName string   `json:"serviceName"`
	Tables      []string `json:"tables"`
	Description string   `json:"description"`
}

// StepResult AI 步骤结果
type StepResult struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// AIProviderPreset AI 服务商预设
type AIProviderPreset struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	BaseURL string `json:"baseUrl"`
}

// GetProviderPresets 返回支持的 AI 服务商预设列表
func GetProviderPresets() []AIProviderPreset {
	return []AIProviderPreset{
		{Name: "OpenAI", Value: "openai", BaseURL: "https://api.openai.com/v1"},
		{Name: "DeepSeek", Value: "deepseek", BaseURL: "https://api.deepseek.com/v1"},
		{Name: "Ollama (Local)", Value: "ollama", BaseURL: "http://localhost:11434/v1"},
		{Name: "Azure OpenAI", Value: "azure", BaseURL: "https://YOUR_RESOURCE.openai.azure.com/openai"},
		{Name: "Custom", Value: "custom", BaseURL: ""},
	}
}

// PartitionResult 微服务划分结果
type PartitionResult struct {
	Success    bool                    `json:"success"`
	Partitions []MicroservicePartition `json:"partitions"`
	Error      string                  `json:"error,omitempty"`
}

// NewPartitionResult 创建成功的微服务划分结果
func NewPartitionResult(partitions []MicroservicePartition) *PartitionResult {
	return &PartitionResult{Success: true, Partitions: partitions}
}

// NewPartitionResultWithError 创建失败的微服务划分结果
func NewPartitionResultWithError(err string) *PartitionResult {
	return &PartitionResult{Success: false, Error: err}
}

// OpenAPIResult OpenAPI 文件查找结果
type OpenAPIResult struct {
	Success bool     `json:"success"`
	Files   []string `json:"files"`
	Message string   `json:"message,omitempty"`
	Error   string   `json:"error,omitempty"`
}
