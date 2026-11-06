package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/svcname"
)

// 任务键:同一时刻 UI 对每种任务只跑一个,取消按任务粒度进行。
const (
	taskDDL       = "ddl"
	taskPartition = "partition"
	taskReview    = "review"
)

// chatClient 抽象对话能力:生产环境注入 *Client,测试注入 mock。
type chatClient interface {
	Chat(ctx context.Context, systemPrompt string, userMessage string) (string, error)
	ChatStream(ctx context.Context, systemPrompt string, userMessage string, onDelta func(string)) (string, error)
	TestConnection(ctx context.Context) (*StepResult, error)
}

// Service AI 服务层
type Service struct {
	// mu 保护 config/client/cancels:Wails 绑定调用各自在独立 goroutine 中执行,
	// "测试连接"与"生成/配置保存"并发时存在数据竞争。
	mu     sync.Mutex
	client chatClient
	config *Config
	// cancels 在途任务的可取消 ctx 注册表(taskCtx 注册,finishTask 清除)。
	cancels map[string]context.CancelFunc
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

// TestConnection 测试 AI 连接(不进取消注册表:轻量操作)
func (s *Service) TestConnection() (*StepResult, error) {
	return s.GetClient().TestConnection(context.Background())
}

// taskCtx 注册一个可取消的任务 ctx 并返回。同名旧任务若仍在途,先取消
// (僵尸兜底:正常路径 finishTask 会清,这里防的是异常退出漏清)。
func (s *Service) taskCtx(task string) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancels == nil {
		s.cancels = make(map[string]context.CancelFunc)
	}
	if old := s.cancels[task]; old != nil {
		old()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancels[task] = cancel
	return ctx
}

// finishTask 任务结束后清除注册项(须与 taskCtx 配对)。
func (s *Service) finishTask(task string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cancels, task)
}

// CancelTask 取消指定在途任务。返回是否确实取消了正在运行的任务。
func (s *Service) CancelTask(task string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cancel := s.cancels[task]
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// friendlyError 取消归一:ctx 已取消(含传输层包着的 context.Canceled)
// 统一报「已取消」,其余原样返回。
func friendlyError(ctx context.Context, err error) string {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) {
		return "已取消"
	}
	return err.Error()
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
// 传输层错误不重试,立即返回;只有校验失败才触发修复重试;可经 CancelTask("ddl") 取消。
func (s *Service) GenerateDDLValidated(requirements string, onDelta func(attempt int, delta string)) (*StepResult, error) {
	ctx := s.taskCtx(taskDDL)
	defer s.finishTask(taskDDL)

	userPrompt := GetDDLPrompt(requirements)
	var lastErr error

	for attempt := 1; attempt <= maxDDLAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return &StepResult{Success: false, Error: "已取消"}, err
		}
		content, err := s.GetClient().ChatStream(ctx, ddlSystemPrompt, userPrompt, func(delta string) {
			if onDelta != nil {
				onDelta(attempt, delta)
			}
		})
		if err != nil {
			msg := friendlyError(ctx, err)
			return &StepResult{Success: false, Error: msg}, err
		}

		cleaned := cleanMarkdownFence(content)
		// 诊断已随 verr.Error() 带行列进入修复提示,这里无需单独取用
		_, _, verr := ValidateDDLDetailed(cleaned)
		if verr == nil {
			return &StepResult{Success: true, Content: cleaned}, nil
		}
		lastErr = verr

		// 修复重试:附上被否决的 DDL 与带行列的解析诊断,要求完整重出。
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
// 可经 CancelTask("partition") 取消。
func (s *Service) PartitionMicroservicesValidated(ddl string) ([]MicroservicePartition, []string, []string, error) {
	ctx := s.taskCtx(taskPartition)
	defer s.finishTask(taskPartition)

	ddlTables, err := ValidateDDL(ddl)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("DDL 无法解析，无法划分微服务: %w", err)
	}

	userPrompt := GetPartitionPrompt(ddl)
	var warnings []string

	for attempt := 1; attempt <= maxPartitionAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, ddlTables, fmt.Errorf("已取消")
		}
		content, err := s.GetClient().Chat(ctx, partitionSystemPrompt, userPrompt)
		if err != nil {
			return nil, nil, ddlTables, fmt.Errorf("AI 微服务划分失败: %s", friendlyError(ctx, err))
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

	ctx := s.taskCtx(taskReview)
	defer s.finishTask(taskReview)

	content, err := s.GetClient().Chat(ctx, reviewSystemPrompt, GetReviewPrompt(fileContents))
	if err != nil {
		return &StepResult{Success: false, Error: friendlyError(ctx, err)}, err
	}

	return &StepResult{Success: true, Content: content}, nil
}

// maxStructuredReviewAttempts 结构化审查的最大尝试次数:首次 + 至多一次修复重试。
const maxStructuredReviewAttempts = 2

// ReviewCodeStructured 单批结构化审查:输出 JSON 契约(summary+issues),
// 解析失败或清洗后无有效内容时带原始响应重试一次。可经 CancelTask("review") 取消。
func (s *Service) ReviewCodeStructured(fileContents map[string]string) (*ReviewResult, error) {
	if len(fileContents) == 0 {
		return nil, fmt.Errorf("没有可审查的代码文件")
	}

	ctx := s.taskCtx(taskReview)
	defer s.finishTask(taskReview)

	userPrompt := GetStructuredReviewPrompt(fileContents)

	for attempt := 1; attempt <= maxStructuredReviewAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("已取消")
		}
		content, err := s.GetClient().Chat(ctx, structuredReviewSystemPrompt, userPrompt)
		if err != nil {
			return nil, fmt.Errorf("AI 审查失败: %s", friendlyError(ctx, err))
		}

		cleaned := strings.TrimSpace(cleanMarkdownFence(content))
		result, perr := parseReviewResultJSON(cleaned)
		problem := ""
		if perr != nil {
			problem = fmt.Sprintf("响应不是合法的审查 JSON 对象: %v", perr)
		} else {
			result.Issues = sanitizeReviewIssues(result.Issues, fileContents)
			if result.Issues == nil {
				result.Issues = []ReviewIssue{}
			}
			if result.Summary == "" {
				result.Summary = "本批未给出摘要"
			}
			return result, nil
		}

		if attempt == maxStructuredReviewAttempts {
			return nil, fmt.Errorf("审查结果连续 %d 次无效(%s)，最后响应: %s", maxStructuredReviewAttempts, problem, cleaned)
		}
		userPrompt = GetStructuredReviewFixPrompt(fileContents, cleaned, problem)
	}

	// 不可达:循环内必返回。
	return nil, fmt.Errorf("结构化审查失败")
}

// parseReviewResultJSON 从清理后的 LLM 文本解析审查结果:
// 先整体 Unmarshal,失败再退化为截取首个 "{" 到最后一个 "}" 的启发式。
func parseReviewResultJSON(cleaned string) (*ReviewResult, error) {
	var result ReviewResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		start := strings.Index(cleaned, "{")
		end := strings.LastIndex(cleaned, "}")
		if start < 0 || end <= start {
			return nil, err
		}
		if err2 := json.Unmarshal([]byte(cleaned[start:end+1]), &result); err2 != nil {
			return nil, err2
		}
	}
	return &result, nil
}

// reviewSeverities 合法严重度集合。
var reviewSeverities = map[string]bool{
	"critical": true,
	"major":    true,
	"minor":    true,
	"info":     true,
}

// sanitizeReviewIssues 清洗 issue 清单:severity 白名单(未知降 minor)、
// line 负数归 0、空 title 用 detail 前 40 字兜底。file 不在批次内也保留
// (LLM 偶尔写相对路径变体,信息仍有效)。
func sanitizeReviewIssues(issues []ReviewIssue, fileContents map[string]string) []ReviewIssue {
	var valid []ReviewIssue
	for _, iss := range issues {
		if !reviewSeverities[iss.Severity] {
			iss.Severity = "minor"
		}
		if iss.Line < 0 {
			iss.Line = 0
		}
		if strings.TrimSpace(iss.Title) == "" {
			detail := strings.TrimSpace(iss.Detail)
			if len(detail) > 40 {
				detail = detail[:40]
			}
			iss.Title = detail
		}
		if strings.TrimSpace(iss.Title) == "" {
			continue // title 与 detail 全空,无信息量,丢弃
		}
		valid = append(valid, iss)
	}
	return valid
}

// SynthesizeReview 汇总各批次结果生成总评(markdown,流式)。
// 可经 CancelTask("review") 取消。
func (s *Service) SynthesizeReview(batchSummaries []string, onDelta func(string)) (string, error) {
	if len(batchSummaries) == 0 {
		return "", fmt.Errorf("没有可汇总的批次结果")
	}

	ctx := s.taskCtx(taskReview)
	defer s.finishTask(taskReview)

	content, err := s.GetClient().ChatStream(ctx, reviewSystemPrompt, GetReviewSynthesisPrompt(batchSummaries), onDelta)
	if err != nil {
		return "", fmt.Errorf("AI 汇总失败: %s", friendlyError(ctx, err))
	}
	return content, nil
}
func (s *Service) ReviewCodeStream(fileContents map[string]string, onDelta func(string)) (*StepResult, error) {
	if len(fileContents) == 0 {
		return &StepResult{Success: false, Error: "没有可审查的代码文件"}, nil
	}

	ctx := s.taskCtx(taskReview)
	defer s.finishTask(taskReview)

	content, err := s.GetClient().ChatStream(ctx, reviewSystemPrompt, GetReviewPrompt(fileContents), onDelta)
	if err != nil {
		return &StepResult{Success: false, Error: friendlyError(ctx, err)}, err
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
