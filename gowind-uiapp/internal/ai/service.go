package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/svcname"
)

// chatClient 抽象对话能力:生产环境注入 *Client,测试注入 mock。
type chatClient interface {
	Chat(systemPrompt string, userMessage string) (string, error)
	ChatStream(systemPrompt string, userMessage string, onDelta func(string)) (string, error)
	TestConnection() (*StepResult, error)
}

// Service AI 服务层
type Service struct {
	// mu 保护 config/client:Wails 绑定调用各自在独立 goroutine 中执行,
	// "测试连接"与"生成/配置保存"并发时存在数据竞争。
	mu     sync.Mutex
	client chatClient
	config *Config
}

// NewService 创建 AI 服务,自动加载持久化配置(缺失时用默认值)。
func NewService() *Service {
	return &Service{
		config: LoadConfig(),
	}
}

// GetConfig 获取当前配置(副本)
func (s *Service) GetConfig() *Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := *s.config
	return &cfg
}

// SetConfig 设置配置、重建客户端并持久化到磁盘(持久化失败不影响本次生效)。
func (s *Service) SetConfig(config *Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
	s.client = NewClient(config)
	if err := SaveConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: AI 配置持久化失败: %v\n", err)
	}
}

// SetConfigTransient 设置配置并重建客户端,但不持久化。
// CLI 等由环境变量/旗标临时构造配置的场景使用,避免把临时值写进用户配置。
func (s *Service) SetConfigTransient(config *Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
	s.client = NewClient(config)
}

// GetClient 获取客户端（懒加载）
func (s *Service) GetClient() chatClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		s.client = NewClient(s.config)
	}
	return s.client
}

// TestConnection 测试 AI 连接
func (s *Service) TestConnection() (*StepResult, error) {
	return s.GetClient().TestConnection()
}

// maxDDLAttempts DDL 生成-校验循环的最大尝试次数:首次生成 + 至多两次修复重试。
const maxDDLAttempts = 3

// GenerateDDL 根据需求文档生成 DDL(内部带解析校验与修复重试)。
func (s *Service) GenerateDDL(requirements string) (*StepResult, error) {
	return s.GenerateDDLValidated(requirements, nil)
}

// GenerateDDLStream GenerateDDL 的流式版本:onDelta 逐块接收增量内容,
// 返回值与非流式版本一致(清理 markdown 围栏后的完整 DDL)。
func (s *Service) GenerateDDLStream(requirements string, onDelta func(string)) (*StepResult, error) {
	return s.GenerateDDLValidated(requirements, func(_ int, delta string) {
		if onDelta != nil {
			onDelta(delta)
		}
	})
}

// GenerateDDLValidated 生成 DDL 并做解析校验,失败则把解析错误带回给模型修复重试。
// onDelta 的第一个参数是本次尝试的序号(从 1 开始):重试时流式消费方必须据此
// 清空已累积的内容,否则两次尝试的文本会拼在一起。
// 传输层错误不重试,立即返回;只有校验失败才触发修复重试。
func (s *Service) GenerateDDLValidated(requirements string, onDelta func(attempt int, delta string)) (*StepResult, error) {
	userPrompt := GetDDLPrompt(requirements)
	var lastErr error

	for attempt := 1; attempt <= maxDDLAttempts; attempt++ {
		content, err := s.GetClient().ChatStream(ddlSystemPrompt, userPrompt, func(delta string) {
			if onDelta != nil {
				onDelta(attempt, delta)
			}
		})
		if err != nil {
			return &StepResult{Success: false, Error: err.Error()}, err
		}

		cleaned := cleanMarkdownFence(content)
		_, verr := ValidateDDL(cleaned)
		if verr == nil {
			return &StepResult{Success: true, Content: cleaned}, nil
		}
		lastErr = verr

		// 修复重试:附上被否决的 DDL 与解析错误,要求完整重出。
		userPrompt = GetDDLFixPrompt(requirements, cleaned, verr.Error())
	}

	err := fmt.Errorf("DDL 连续 %d 次未通过解析校验: %v", maxDDLAttempts, lastErr)
	return &StepResult{Success: false, Error: err.Error()}, err
}

// maxPartitionAttempts 划分生成-校验循环的最大尝试次数:首次生成 + 至多一次修复重试。
const maxPartitionAttempts = 2

// PartitionMicroservices 根据 DDL 建议微服务划分(内部带结构化校验与修复重试)。
func (s *Service) PartitionMicroservices(ddl string) ([]MicroservicePartition, error) {
	partitions, _, _, err := s.PartitionMicroservicesValidated(ddl)
	return partitions, err
}

// PartitionMicroservicesValidated PartitionMicroservices 的带校验版本:
// - 先校验 DDL 本身可解析,拿到真实表清单(一并返回,供前端编辑划分的未分配池);
// - LLM 返回的 JSON 解析失败、或清理后没有任何有效划分时,带问题反馈重试一次;
// - 输出做内容清洗:serviceName 非法/为空、tables 为空的划分剔除,
//   不在 DDL 中的幻觉表名剔除,跨服务重复表去重——全部记入 warnings 返回。
func (s *Service) PartitionMicroservicesValidated(ddl string) ([]MicroservicePartition, []string, []string, error) {
	ddlTables, err := ValidateDDL(ddl)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("DDL 无法解析，无法划分微服务: %w", err)
	}

	userPrompt := GetPartitionPrompt(ddl)
	var warnings []string

	for attempt := 1; attempt <= maxPartitionAttempts; attempt++ {
		content, err := s.GetClient().Chat(partitionSystemPrompt, userPrompt)
		if err != nil {
			return nil, nil, ddlTables, fmt.Errorf("AI 微服务划分失败: %w", err)
		}

		cleaned := strings.TrimSpace(cleanMarkdownFence(content))
		partitions, perr := parsePartitionsJSON(cleaned)
		problem := ""
		if perr != nil {
			problem = fmt.Sprintf("响应不是合法的划分 JSON 数组: %v", perr)
		} else {
			partitions, warnings = sanitizePartitions(partitions, ddlTables)
			if len(partitions) == 0 {
				problem = "响应中没有包含任何有效划分( serviceName 非法/为空、tables 为空、表名不在 DDL 中都会被判无效)"
			}
		}

		if problem == "" {
			return partitions, ddlTables, warnings, nil
		}
		if attempt == maxPartitionAttempts {
			return nil, ddlTables, warnings, fmt.Errorf("划分结果连续 %d 次无效(%s)，最后响应: %s", maxPartitionAttempts, problem, cleaned)
		}
		// 修复重试:把被否决的原始响应与问题一并带回。
		userPrompt = GetPartitionFixPrompt(ddl, cleaned, problem)
	}

	// 不可达:maxPartitionAttempts >= 1,循环内必返回。
	return nil, ddlTables, warnings, fmt.Errorf("微服务划分失败")
}

// parsePartitionsJSON 从清理后的 LLM 文本解析划分数组:
// 先整体 Unmarshal,失败再退化为截取首个 "[" 到最后一个 "]" 的启发式。
func parsePartitionsJSON(cleaned string) ([]MicroservicePartition, error) {
	var partitions []MicroservicePartition
	if err := json.Unmarshal([]byte(cleaned), &partitions); err != nil {
		start := strings.Index(cleaned, "[")
		end := strings.LastIndex(cleaned, "]")
		if start < 0 || end <= start {
			return nil, err
		}
		if err2 := json.Unmarshal([]byte(cleaned[start:end+1]), &partitions); err2 != nil {
			return nil, err2
		}
	}
	return partitions, nil
}

// sanitizePartitions 对划分结果做内容清洗,返回有效划分与告警清单。
// 表名以 DDL 原始拼写为准(LLM 输出的反引号/大小写会被归一化匹配后替换)。
func sanitizePartitions(partitions []MicroservicePartition, ddlTables []string) ([]MicroservicePartition, []string) {
	var valid []MicroservicePartition
	var warnings []string
	seen := map[string]string{} // 归一化表名 -> 首个认领它的服务

	for _, p := range partitions {
		if err := svcname.Validate(p.ServiceName); err != nil {
			warnings = append(warnings, fmt.Sprintf("划分服务名非法(%v)，已剔除该划分", err))
			continue
		}
		if len(p.Tables) == 0 {
			warnings = append(warnings, fmt.Sprintf("服务 %q 没有分到任何表，已剔除", p.ServiceName))
			continue
		}

		tables := make([]string, 0, len(p.Tables))
		for _, t := range p.Tables {
			canonical, ok := matchTableName(t, ddlTables)
			if !ok {
				warnings = append(warnings, fmt.Sprintf("表 %q 不在 DDL 中，已从服务 %q 剔除", t, p.ServiceName))
				continue
			}
			if owner, dup := seen[normalizeTableName(canonical)]; dup {
				warnings = append(warnings, fmt.Sprintf("表 %q 已归属服务 %q，从服务 %q 去重", canonical, owner, p.ServiceName))
				continue
			}
			seen[normalizeTableName(canonical)] = p.ServiceName
			tables = append(tables, canonical)
		}

		if len(tables) == 0 {
			warnings = append(warnings, fmt.Sprintf("服务 %q 的表全部无效，已剔除", p.ServiceName))
			continue
		}
		valid = append(valid, MicroservicePartition{
			ServiceName: p.ServiceName,
			Tables:      tables,
			Description: p.Description,
		})
	}
	return valid, warnings
}

// ReviewCode 审查代码
func (s *Service) ReviewCode(fileContents map[string]string) (*StepResult, error) {
	if len(fileContents) == 0 {
		return &StepResult{Success: false, Error: "没有可审查的代码文件"}, nil
	}

	content, err := s.GetClient().Chat(reviewSystemPrompt, GetReviewPrompt(fileContents))
	if err != nil {
		return &StepResult{Success: false, Error: err.Error()}, err
	}

	return &StepResult{Success: true, Content: content}, nil
}

// ReviewCodeStream ReviewCode 的流式版本:onDelta 逐块接收增量审查意见。
func (s *Service) ReviewCodeStream(fileContents map[string]string, onDelta func(string)) (*StepResult, error) {
	if len(fileContents) == 0 {
		return &StepResult{Success: false, Error: "没有可审查的代码文件"}, nil
	}

	content, err := s.GetClient().ChatStream(reviewSystemPrompt, GetReviewPrompt(fileContents), onDelta)
	if err != nil {
		return &StepResult{Success: false, Error: err.Error()}, err
	}
	return &StepResult{Success: true, Content: content}, nil
}

// FindOpenAPIFiles 在项目目录中查找 openapi.yaml 文件
func FindOpenAPIFiles(projectRoot string) ([]string, error) {
	var files []string
	searchPaths := []string{
		"cmd/server/assets/openapi.yaml",
		"cmd/server/assets/openapi.yml",
		"openapi.yaml",
		"openapi.yml",
	}

	// 在所有子目录中搜索
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		name := strings.ToLower(info.Name())
		for _, searchPath := range searchPaths {
			searchBase := filepath.Base(searchPath)
			if name == searchBase {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("搜索项目目录失败: %w", err)
	}

	return files, nil
}

// cleanMarkdownFence 清除 markdown 代码块标记
func cleanMarkdownFence(content string) string {
	content = strings.TrimSpace(content)
	// 移除开头的 ```sql 或 ```json 或 ```
	content = strings.TrimPrefix(content, "```sql")
	content = strings.TrimPrefix(content, "```SQL")
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```JSON")
	content = strings.TrimPrefix(content, "```")
	// 移除结尾的 ```
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}
