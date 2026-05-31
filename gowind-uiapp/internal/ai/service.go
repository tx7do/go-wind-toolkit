package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Service AI 服务层
type Service struct {
	client *Client
	config *Config
}

// NewService 创建 AI 服务
func NewService() *Service {
	return &Service{
		config: DefaultConfig(),
	}
}

// GetConfig 获取当前配置
func (s *Service) GetConfig() *Config {
	return s.config
}

// SetConfig 设置配置并重建客户端
func (s *Service) SetConfig(config *Config) {
	s.config = config
	s.client = NewClient(config)
}

// GetClient 获取客户端（懒加载）
func (s *Service) GetClient() *Client {
	if s.client == nil {
		s.client = NewClient(s.config)
	}
	return s.client
}

// TestConnection 测试 AI 连接
func (s *Service) TestConnection() (*StepResult, error) {
	return s.GetClient().TestConnection()
}

// GenerateDDL 根据需求文档生成 DDL
func (s *Service) GenerateDDL(requirements string) (*StepResult, error) {
	content, err := s.GetClient().Chat(ddlSystemPrompt, GetDDLPrompt(requirements))
	if err != nil {
		return &StepResult{Success: false, Error: err.Error()}, err
	}

	// 清理可能的 markdown 代码块包裹
	cleaned := cleanMarkdownFence(content)

	return &StepResult{Success: true, Content: cleaned}, nil
}

// PartitionMicroservices 根据 DDL 建议微服务划分
func (s *Service) PartitionMicroservices(ddl string) ([]MicroservicePartition, error) {
	content, err := s.GetClient().Chat(partitionSystemPrompt, GetPartitionPrompt(ddl))
	if err != nil {
		return nil, fmt.Errorf("AI 微服务划分失败: %w", err)
	}

	// 清理并解析 JSON
	cleaned := cleanMarkdownFence(content)
	cleaned = strings.TrimSpace(cleaned)

	var partitions []MicroservicePartition
	if err := json.Unmarshal([]byte(cleaned), &partitions); err != nil {
		// 尝试从文本中提取 JSON 数组
		start := strings.Index(cleaned, "[")
		end := strings.LastIndex(cleaned, "]")
		if start >= 0 && end > start {
			jsonStr := cleaned[start : end+1]
			if err2 := json.Unmarshal([]byte(jsonStr), &partitions); err2 != nil {
				return nil, fmt.Errorf("解析微服务划分结果失败: %w\n原始响应: %s", err, cleaned)
			}
		} else {
			return nil, fmt.Errorf("解析微服务划分结果失败: %w\n原始响应: %s", err, cleaned)
		}
	}

	return partitions, nil
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
