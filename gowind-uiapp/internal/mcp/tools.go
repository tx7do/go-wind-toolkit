package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/ai"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/detect"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/devtools"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/svcname"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/frontendgen"
)

// toolHandler 工具处理器:入参原始 JSON,返回可序列化结果或错误。
type toolHandler func(s *Server, args json.RawMessage) (any, error)

// toolDef 工具定义(name/描述/JSON Schema)。
type toolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// schema 构造 object 型 JSON Schema。
func schema(properties map[string]any, required ...string) json.RawMessage {
	if required == nil {
		required = []string{}
	}
	data, err := json.Marshal(map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	})
	if err != nil {
		return json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
	}
	return data
}

// toolDefinitions tools/list 的全部工具定义。
func toolDefinitions() []toolDef {
	return []toolDef{
		{
			Name:        "generate_ddl",
			Description: "根据需求文档生成 MySQL DDL(生成后自动解析校验,失败带行列诊断自动修复重试)",
			InputSchema: schema(map[string]any{
				"requirements": map[string]any{"type": "string", "description": "软件需求描述(中文/英文皆可)"},
			}, "requirements"),
		},
		{
			Name:        "validate_ddl",
			Description: "用 tidb parser 校验 MySQL DDL 语法,返回表名清单与带行列的错误诊断",
			InputSchema: schema(map[string]any{
				"sql": map[string]any{"type": "string", "description": "DDL 文本(可多语句)"},
			}, "sql"),
		},
		{
			Name:        "partition_microservices",
			Description: "按 DDL 建议微服务划分(JSON 输出);幻觉表名剔除、跨服务去重,附 warnings",
			InputSchema: schema(map[string]any{
				"ddl": map[string]any{"type": "string", "description": "CREATE TABLE 语句文本"},
			}, "ddl"),
		},
		{
			Name:        "find_openapi",
			Description: "在项目中查找 OpenAPI 文档文件",
			InputSchema: schema(map[string]any{
				"projectRoot": map[string]any{"type": "string", "description": "项目根目录(Go 模块)"},
			}, "projectRoot"),
		},
		{
			Name:        "scan_reviewable_files",
			Description: "扫描项目内可审查代码(app/ 下 Go 源码与 api/protos 下 proto,排除测试与生成物)",
			InputSchema: schema(map[string]any{
				"projectRoot": map[string]any{"type": "string", "description": "项目根目录(Go 模块)"},
			}, "projectRoot"),
		},
		{
			Name:        "review_files",
			Description: "AI 分批审查选中的项目文件:逐批结构化 issue + 汇总总评(多次 LLM 调用)",
			InputSchema: schema(map[string]any{
				"projectRoot": map[string]any{"type": "string", "description": "项目根目录"},
				"paths":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "相对项目根的文件路径(来自 scan_reviewable_files)"},
			}, "projectRoot", "paths"),
		},
		{
			Name:        "plan_backend",
			Description: "预演后端 gRPC 代码生成:完整生成树落临时目录(不碰项目),返回新增/覆盖文件清单;用 apply_backend 确认写入",
			InputSchema: schema(map[string]any{
				"projectRoot": map[string]any{"type": "string", "description": "项目根目录(Go 模块)"},
				"ddl":         map[string]any{"type": "string", "description": "CREATE TABLE 语句文本"},
				"ormType":     map[string]any{"type": "string", "enum": []string{"ent", "gorm"}, "description": "ORM 类型,默认 ent"},
				"partitions":  map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "微服务划分 [{serviceName, tables[], description}]"},
				"withTests":   map[string]any{"type": "boolean", "description": "为 CRUD service 追加单测骨架(fake 仓储),默认 false"},
			}, "projectRoot", "ddl", "partitions"),
		},
		{
			Name:        "apply_backend",
			Description: "把 plan_backend 的预览树写入项目并执行后处理链(go mod tidy/buf generate/ent generate/wire;需这些工具在 PATH)。写入后失败不回滚",
			InputSchema: schema(map[string]any{
				"planId": map[string]any{"type": "string", "description": "plan_backend 返回的 planId"},
			}, "planId"),
		},
		{
			Name:        "generate_frontend",
			Description: "从 OpenAPI 规范生成前端 CRUD 页面(vue-element/vue-vben/react);openapi 缺省时用 find_openapi 自动查找,dryRun=true 只返回文件清单不写盘",
			InputSchema: schema(map[string]any{
				"projectRoot": map[string]any{"type": "string", "description": "项目根目录(Go 模块)"},
				"framework":   map[string]any{"type": "string", "enum": []string{"vue-element", "vue-vben", "react"}, "description": "目标前端框架"},
				"openapi":     map[string]any{"type": "string", "description": "OpenAPI 3.0 文件路径或 http(s) URL(缺省时自动查找项目内 openapi.yaml)"},
				"out":         map[string]any{"type": "string", "description": "输出目录(前端项目 src 目录;相对路径时基于 projectRoot 解析)"},
				"serviceName": map[string]any{"type": "string", "description": "服务名(generated 导入路径前缀),默认 admin"},
				"tags":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "要生成的服务 tag 名(缺省=全部)"},
				"types":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "文件类型 composable/page/drawer/router/locale(缺省=全部)"},
				"dryRun":      map[string]any{"type": "boolean", "description": "只返回文件清单不写盘,默认 false"},
				"errorI18n":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "wind 错误词表文件或目录(*_errors_i18n.<lang>.json),合并产出 locales/<lang>/errors.json"},
			}, "projectRoot", "framework", "out"),
		},
	}
}

// toolHandlers 工具名 → 处理器(方法表达式,签名统一为 func(*Server, json.RawMessage))。
func toolHandlers() map[string]toolHandler {
	return map[string]toolHandler{
		"generate_ddl":            (*Server).handleGenerateDDL,
		"validate_ddl":            (*Server).handleValidateDDL,
		"partition_microservices": (*Server).handlePartition,
		"find_openapi":            (*Server).handleFindOpenAPI,
		"scan_reviewable_files":   (*Server).handleScanFiles,
		"review_files":            (*Server).handleReviewFiles,
		"plan_backend":            (*Server).handlePlanBackend,
		"apply_backend":           (*Server).handleApplyBackend,
		"generate_frontend":       (*Server).handleGenerateFrontend,
	}
}

// ---- 入参结构 ----

type generateDDLArgs struct {
	Requirements string `json:"requirements"`
}

type validateDDLArgs struct {
	SQL string `json:"sql"`
}

type partitionArgs struct {
	DDL string `json:"ddl"`
}

type projectRootArgs struct {
	ProjectRoot string `json:"projectRoot"`
}

type reviewFilesArgs struct {
	ProjectRoot string   `json:"projectRoot"`
	Paths       []string `json:"paths"`
}

type planBackendArgs struct {
	ProjectRoot string          `json:"projectRoot"`
	DDL         string          `json:"ddl"`
	ORMType     string          `json:"ormType"`
	Partitions  json.RawMessage `json:"partitions"`
	WithTests   bool            `json:"withTests"`
}

type applyBackendArgs struct {
	PlanID string `json:"planId"`
}

type generateFrontendArgs struct {
	ProjectRoot string   `json:"projectRoot"`
	Framework   string   `json:"framework"`
	OpenAPI     string   `json:"openapi"`
	Out         string   `json:"out"`
	ServiceName string   `json:"serviceName"`
	Tags        []string `json:"tags"`
	Types       []string `json:"types"`
	ErrorI18N   []string `json:"errorI18n"`
	DryRun      bool     `json:"dryRun"`
}

// ---- AI 服务装配 ----

// newEnvAIService 持久化配置为底 + GOWIND_AI_* 环境变量覆盖(与 CLI 语义一致),
// SetConfigTransient 不落盘。
func newEnvAIService() *ai.Service {
	cfg := ai.LoadConfig()
	env := func(key string) string { return os.Getenv(key) }
	if v := env("GOWIND_AI_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := env("GOWIND_AI_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := env("GOWIND_AI_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := env("GOWIND_AI_AZURE_API_VERSION"); v != "" {
		cfg.AzureAPIVersion = v
	}
	if v := env("GOWIND_AI_MODEL"); v != "" {
		cfg.Model = v
	}
	svc := ai.NewService()
	svc.SetConfigTransient(cfg)
	return svc
}

// resolveRoot 解析工具入参中的项目根:必须可检测为 Go 模块,
// 返回 (Root, ModPath)。失败返回错误——MCP 工具的 root 是显式入参,
// 不像 GUI 有"已打开项目"兜底,宽松回退容易写错目录。
func resolveRoot(projectRoot string) (string, string, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return "", "", fmt.Errorf("projectRoot 不能为空")
	}
	info, err := detect.NewProjectDetector().Detect(projectRoot)
	if err != nil || info.ModPath == "" {
		return "", "", fmt.Errorf("projectRoot 不是有效的 Go 模块: %v", err)
	}
	return info.Root, info.ModPath, nil
}

// ---- 工具处理器 ----

func (s *Server) handleGenerateDDL(args json.RawMessage) (any, error) {
	var a generateDDLArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Requirements) == "" {
		return nil, fmt.Errorf("requirements 不能为空")
	}
	result, err := s.aiService.GenerateDDL(a.Requirements)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": result.Success, "ddl": result.Content, "error": result.Error}, nil
}

func (s *Server) handleValidateDDL(args json.RawMessage) (any, error) {
	var a validateDDLArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	tables, diags, err := ai.ValidateDDLDetailed(a.SQL)
	if err != nil {
		return map[string]any{"success": false, "tables": tables, "diagnostics": diags, "error": err.Error()}, nil
	}
	return map[string]any{"success": true, "tables": tables}, nil
}

func (s *Server) handlePartition(args json.RawMessage) (any, error) {
	var a partitionArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	partitions, tables, warnings, err := s.aiService.PartitionMicroservicesValidated(a.DDL)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"partitions":      partitions,
		"availableTables": tables,
		"warnings":        warnings,
	}, nil
}

func (s *Server) handleFindOpenAPI(args json.RawMessage) (any, error) {
	var a projectRootArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	root, _, err := resolveRoot(a.ProjectRoot)
	if err != nil {
		return nil, err
	}
	files, err := ai.FindOpenAPIFiles(root)
	if err != nil {
		return nil, err
	}
	return map[string]any{"files": files}, nil
}

func (s *Server) handleScanFiles(args json.RawMessage) (any, error) {
	var a projectRootArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	root, _, err := resolveRoot(a.ProjectRoot)
	if err != nil {
		return nil, err
	}
	res, err := ai.ScanProjectCode(root)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Server) handleReviewFiles(args json.RawMessage) (any, error) {
	var a reviewFilesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	root, _, err := resolveRoot(a.ProjectRoot)
	if err != nil {
		return nil, err
	}
	fileContents, readWarnings, err := ai.ReadProjectFiles(root, a.Paths)
	if err != nil {
		return nil, err
	}
	_ = readWarnings

	infos := make([]ai.CodeFileInfo, 0, len(fileContents))
	for path, content := range fileContents {
		infos = append(infos, ai.CodeFileInfo{Path: path, Size: int64(len(content))})
	}
	batches := ai.BuildReviewBatches(infos)
	if len(batches) == 0 {
		return nil, fmt.Errorf("没有可审查的文件")
	}

	type batchOutcome struct {
		Label      string `json:"label"`
		Summary    string `json:"summary"`
		IssueCount int    `json:"issueCount"`
	}
	var outcomes []batchOutcome
	var allIssues []ai.ReviewIssue
	var summaries []string

	for _, batch := range batches {
		contents := make(map[string]string, len(batch))
		for _, f := range batch {
			contents[f.Path] = fileContents[f.Path]
		}
		review, err := s.aiService.ReviewCodeStructured(contents)
		if err != nil {
			return nil, err
		}
		label := ai.BatchLabel(batch)
		outcomes = append(outcomes, batchOutcome{Label: label, Summary: review.Summary, IssueCount: len(review.Issues)})
		allIssues = append(allIssues, review.Issues...)
		summaries = append(summaries, fmt.Sprintf("### %s(%d 个问题)\n%s", label, len(review.Issues), review.Summary))
	}

	synthesis, err := s.aiService.SynthesizeReview(summaries, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"batches":   outcomes,
		"issues":    allIssues,
		"synthesis": synthesis,
	}, nil
}

func (s *Server) handlePlanBackend(args json.RawMessage) (any, error) {
	var a planBackendArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	root, modPath, err := resolveRoot(a.ProjectRoot)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.DDL) == "" {
		return nil, fmt.Errorf("ddl 不能为空")
	}
	ormType := a.ORMType
	if ormType == "" {
		ormType = "ent"
	}
	partitions, err := parsePartitionsArg(a.Partitions)
	if err != nil {
		return nil, err
	}
	if len(partitions) == 0 {
		return nil, fmt.Errorf("partitions 不能为空")
	}
	services := make([]string, 0, len(partitions))
	seen := map[string]bool{}
	for _, p := range partitions {
		if err := svcname.Validate(p.ServiceName); err != nil {
			return nil, fmt.Errorf("服务名非法: %v", err)
		}
		if !seen[p.ServiceName] {
			seen[p.ServiceName] = true
			services = append(services, p.ServiceName)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearPlanLocked()

	tmpDir, err := os.MkdirTemp("", "gowind-mcp-plan")
	if err != nil {
		return nil, fmt.Errorf("创建预览目录失败: %w", err)
	}

	// 生成器选项(与 GUI 预览路径同构)
	s.gen.CleanOptions()
	for _, p := range partitions {
		for _, tableName := range p.Tables {
			s.gen.AddOption(&generator.Option{TableName: tableName, Service: p.ServiceName})
		}
	}
	dbCfg := &database.DBConfig{Type: database.DbTypeMySQL, UseDSN: false, SQLContent: a.DDL}

	// entimport 要求 schema 目录在 Go 模块内;stub 生成后即删,防混入拷贝树
	stubGoMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(stubGoMod, []byte("module "+modPath+"\n"), 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("初始化预览目录失败: %w", err)
	}
	s.gen.SetSkipPostProcess(true)
	s.gen.SetGenerateTests(a.WithTests)
	genErr := s.gen.GenerateGrpcCode(s.ctx, *dbCfg, ormType, "per-table", tmpDir, modPath, []string{"grpc"})
	s.gen.SetGenerateTests(false)
	s.gen.SetSkipPostProcess(false)
	os.Remove(stubGoMod)
	if genErr != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("生成预览失败: %v", genErr)
	}

	files, werr := ai.WalkPlanFiles(tmpDir, root)
	if werr != nil {
		os.RemoveAll(tmpDir)
		return nil, werr
	}
	newCount, overwriteCount := 0, 0
	for _, f := range files {
		if f.IsNew {
			newCount++
		} else {
			overwriteCount++
		}
	}

	plan := &backendPlanState{id: ai.NewPlanID(), dir: tmpDir, services: services, root: root, modPath: modPath, ormType: ormType}
	s.plan = plan

	return map[string]any{
		"planId":         plan.id,
		"files":          files,
		"newCount":       newCount,
		"overwriteCount": overwriteCount,
		"hint":           "用 apply_backend(planId) 确认写入项目",
	}, nil
}

func (s *Server) handleApplyBackend(args json.RawMessage) (any, error) {
	var a applyBackendArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	s.mu.Lock()
	plan := s.plan
	s.mu.Unlock()
	if plan == nil || plan.id != a.PlanID {
		return nil, fmt.Errorf("生成计划不存在或已过期，请重新 plan_backend")
	}

	if err := ai.CopyTree(plan.root, plan.dir); err != nil {
		s.discardPlan("")
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}

	// 后处理链:Run* 返回 *CommandResult,以 Success 判定;失败不回滚已拷贝文件。
	fail := func(stage string, res *devtools.CommandResult) (any, error) {
		s.discardPlan("")
		return nil, fmt.Errorf("文件已写入，但 %s 失败: %s", stage, res.Error)
	}
	if res := devtools.RunGoModTidy(plan.root); !res.Success {
		return fail("go mod tidy", res)
	}
	if res := devtools.RunBufGenerate(plan.root); !res.Success {
		return fail("buf generate", res)
	}
	if plan.ormType == "ent" {
		for _, svc := range plan.services {
			if res := devtools.RunEntGenerate(plan.root, svc); !res.Success {
				return fail("ent generate("+svc+")", res)
			}
		}
	}
	for _, svc := range plan.services {
		if res := devtools.RunWire(plan.root, svc); !res.Success {
			return fail("wire("+svc+")", res)
		}
	}

	s.discardPlan(a.PlanID)
	return map[string]any{"success": true, "root": plan.root}, nil
}

// handleGenerateFrontend 从 OpenAPI 规范生成前端 CRUD 页面。
// openapi 缺省时自动查找项目内 openapi.yaml(取第一个命中);
// 相对 out 以 projectRoot 为基准解析。
func (s *Server) handleGenerateFrontend(args json.RawMessage) (any, error) {
	var a generateFrontendArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	root, _, err := resolveRoot(a.ProjectRoot)
	if err != nil {
		return nil, err
	}

	framework, ok := frontendgen.ParseFramework(a.Framework)
	if !ok {
		return nil, fmt.Errorf("无效的 framework %q(可选 vue-element / vue-vben / react)", a.Framework)
	}
	if strings.TrimSpace(a.Out) == "" {
		return nil, fmt.Errorf("out 不能为空(前端项目 src 目录)")
	}

	openapiRef := a.OpenAPI
	if openapiRef == "" {
		files, err := ai.FindOpenAPIFiles(root)
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("项目内未找到 openapi.yaml,请通过 openapi 参数显式指定")
		}
		openapiRef = files[0]
	}
	spec, err := ai.LoadOpenAPISpec(openapiRef)
	if err != nil {
		return nil, err
	}

	opts := frontendgen.Options{
		Spec:              spec,
		Framework:         framework,
		ServiceName:       a.ServiceName,
		Tags:              a.Tags,
		GenerateTypes:     a.Types,
		ErrorI18NFiles:    a.ErrorI18N,
		AutoRouterModules: true,
	}
	files, err := frontendgen.Generate(opts)
	if err != nil {
		return nil, err
	}

	if a.DryRun {
		manifest := make([]map[string]any, 0, len(files))
		for _, f := range files {
			manifest = append(manifest, map[string]any{
				"path":        f.Path,
				"type":        f.Type,
				"description": f.Description,
				"serviceName": f.ServiceName,
				"bytes":       len(f.Content),
			})
		}
		return map[string]any{"dryRun": true, "files": manifest, "count": len(manifest)}, nil
	}

	outDir := a.Out
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(root, outDir)
	}
	results, err := frontendgen.WriteFiles(files, outDir)
	if err != nil {
		return nil, err
	}
	return map[string]any{"results": results, "count": len(results), "out": outDir}, nil
}

// discardPlan 清理计划与临时目录(planID 传空丢弃任意;不匹配则保留)。
func (s *Server) discardPlan(planID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.plan != nil && (planID == "" || s.plan.id == planID) {
		s.clearPlanLocked()
	}
}

// clearPlanLocked 清当前计划(须持 mu)。
func (s *Server) clearPlanLocked() {
	if s.plan != nil {
		os.RemoveAll(s.plan.dir)
		s.plan = nil
	}
}

// parsePartitionsArg 划分入参:兼容裸数组与 {success,partitions} 对象
// (与 CLI parsePartitions 语义一致)。
func parsePartitionsArg(raw json.RawMessage) ([]ai.MicroservicePartition, error) {
	var arr []ai.MicroservicePartition
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var wrapper struct {
		Partitions []ai.MicroservicePartition `json:"partitions"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && wrapper.Partitions != nil {
		return wrapper.Partitions, nil
	}
	return nil, fmt.Errorf("partitions 解析失败: 应为 [{serviceName, tables[], description}] 数组")
}
