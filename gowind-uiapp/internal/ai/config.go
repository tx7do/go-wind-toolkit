package ai

// Config AI 服务配置
type Config struct {
	Provider        string  `json:"provider"`                  // "openai", "azure", "anthropic", "gemini", "ollama", "deepseek", "custom"
	BaseURL         string  `json:"baseUrl"`                   // API 基础地址
	APIKey          string  `json:"apiKey"`                    // API 密钥
	AzureAPIVersion string  `json:"azureApiVersion,omitempty"` // Azure OpenAI 部署的 api-version(仅 provider=azure 生效)
	Model           string  `json:"model"`                     // 模型名称
	Temperature     float64 `json:"temperature"`               // 温度参数 (0.0-2.0)
	MaxTokens       int     `json:"maxTokens"`                 // 最大 token 数
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
		{Name: "Anthropic", Value: "anthropic", BaseURL: "https://api.anthropic.com"},
		{Name: "Gemini", Value: "gemini", BaseURL: "https://generativelanguage.googleapis.com"},
		{Name: "Ollama (Local)", Value: "ollama", BaseURL: "http://localhost:11434/v1"},
		{Name: "Azure OpenAI", Value: "azure", BaseURL: "https://YOUR_RESOURCE.openai.azure.com/openai"},
		{Name: "Custom", Value: "custom", BaseURL: ""},
	}
}

// PartitionResult 微服务划分结果
type PartitionResult struct {
	Success    bool                    `json:"success"`
	Partitions []MicroservicePartition `json:"partitions"`
	// AvailableTables DDL 解析出的全部表名(按出现顺序):前端编辑划分时的
	// "未分配池"数据源,用于发现遗漏或手滑删掉的表。
	AvailableTables []string `json:"availableTables,omitempty"`
	// Warnings 内容清洗告警:幻觉表名剔除、跨服务去重、非法服务名剔除等。
	Warnings []string `json:"warnings,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// DDLValidateResult DDL 校验结果
type DDLValidateResult struct {
	Success bool     `json:"success"`
	Tables  []string `json:"tables,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// BackendPlanFile 后端生成计划中的单个文件
type BackendPlanFile struct {
	Path  string `json:"path"` // 相对项目根,正斜杠
	Size  int64  `json:"size"`
	IsNew bool   `json:"isNew"` // true=新增,false=覆盖项目内已有文件
}

// BackendPlanResult 后端生成预览(Plan)结果:完整生成树落在服务端临时目录,
// 用户确认后由 AIApplyBackendCode 原样拷贝进项目并执行后处理链。
type BackendPlanResult struct {
	Success        bool              `json:"success"`
	PlanID         string            `json:"planId,omitempty"`
	Files          []BackendPlanFile `json:"files,omitempty"`
	NewCount       int               `json:"newCount"`
	OverwriteCount int               `json:"overwriteCount"`
	// PostProcess 确认写入后将执行的后处理步骤说明(预览阶段不执行)。
	PostProcess []string `json:"postProcess,omitempty"`
	Error       string   `json:"error,omitempty"`
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
