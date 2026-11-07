package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tx7do/go-utils/ddl_parser"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/ai"
	ce "github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/detect"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/devtools"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/svcname"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/frontendgen"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlkratos"
)

// App struct
type App struct {
	ctx context.Context

	// stateMu 保护 projectInfo/dbConfig:Wails 绑定调用各自在独立
	// goroutine 中执行,并发读写存在数据竞争。指针只在锁内整体替换,
	// 指向的结构体发布后不再修改。
	stateMu     sync.Mutex
	projectInfo *detect.ProjectInfo
	dbConfig    *database.DBConfig

	projectDetector *detect.ProjectDetector
	generator       *generator.Generator
	aiService       *ai.Service

	// backendPlan 当前待确认的后端生成计划(单份;新预览覆盖旧预览)。
	// planMu 串行化预览/应用/丢弃:预览含一次完整生成,耗时秒级。
	planMu      sync.Mutex
	backendPlan *backendPlanState
}

// backendPlanState 一份待确认的后端生成计划:完整生成树在临时目录里,
// Apply 时原样拷贝进项目。
type backendPlanState struct {
	id       string
	dir      string
	services []string // 参与后处理链的服务名(去重,按划分顺序)
	modPath  string   // 创建计划时的项目 ModPath,Apply 时校验未切换
	ormType  string
}

func (a *App) setProjectInfo(pi *detect.ProjectInfo) {
	a.stateMu.Lock()
	a.projectInfo = pi
	a.stateMu.Unlock()
}

func (a *App) getProjectInfo() *detect.ProjectInfo {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.projectInfo
}

func (a *App) setDBConfig(cfg *database.DBConfig) {
	a.stateMu.Lock()
	a.dbConfig = cfg
	a.stateMu.Unlock()
}

func (a *App) getDBConfig() *database.DBConfig {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.dbConfig
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		projectDetector: detect.NewProjectDetector(),
		generator:       generator.NewGenerator(),
		aiService:       ai.NewService(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.generator.SetLogger(&wailsLogger{app: a})
}

// wailsLogger 将生成过程日志转发到 wails runtime
type wailsLogger struct {
	app *App
}

func (l *wailsLogger) Infof(format string, args ...any) {
	runtime.LogInfof(l.app.ctx, format, args...)
}

func (l *wailsLogger) Errorf(format string, args ...any) {
	runtime.LogErrorf(l.app.ctx, format, args...)
}

// OpenProjectResult 描述一次 OpenProject 调用的结果。
// Status 取值：
//   - "opened"  已直接打开，Project 为项目信息；
//   - "choose"  所选目录无自身 go.mod 但发现多个子模块，需用户从 Candidates 中选择
//     （前端全局选择器据 project-modules-found 事件弹框，调用方此时不应报错也不清空当前项目）；
//   - "invalid" 既非模块根、也无子模块、且不属于任何上级模块。
type OpenProjectResult struct {
	Status     string                   `json:"Status"`
	Project    *detect.ProjectInfo      `json:"Project,omitempty"`
	Candidates []detect.ModuleCandidate `json:"Candidates,omitempty"`
}

// OpenProject 打开指定路径的项目。目录自身是模块根则直接打开；
// 否则向下发现候选子模块交给前端选择；再否则回退向上探测所属父模块。
func (a *App) OpenProject(projectPath string) *OpenProjectResult {
	// 清理路径：去除前后空格、换行符、控制字符等
	projectPath = strings.TrimSpace(projectPath)
	projectPath = strings.ReplaceAll(projectPath, "\r\n", "")
	projectPath = strings.ReplaceAll(projectPath, "\n", "")
	projectPath = strings.ReplaceAll(projectPath, "\r", "")
	projectPath = strings.TrimFunc(projectPath, func(r rune) bool {
		return r < 32 && r != '\t' // 保留制表符
	})

	if !detect.HasGoMod(projectPath) {
		if candidates, err := detect.FindModulesUnder(projectPath, detect.DefaultModuleSearchDepth); err == nil && len(candidates) > 0 {
			runtime.EventsEmit(a.ctx, "project-modules-found", candidates)
			return &OpenProjectResult{Status: "choose", Candidates: candidates}
		}
	}

	pi, err := a.projectDetector.Detect(projectPath)
	if err != nil || pi == nil || pi.ModPath == "" {
		if err != nil {
			runtime.LogErrorf(a.ctx, "项目检测失败：%v (路径：%q)", err, projectPath)
		}
		return &OpenProjectResult{Status: "invalid"}
	}
	// 切换模块时丢弃上一个项目的表配置与 DSN，否则生成器会把 A 项目的表写进 B 项目。
	// 前端不靠事件感知（避免先清空再赋值的闪烁），而是比较 projectInfo.ModPath 自行复位。
	if prev := a.getProjectInfo(); prev == nil || prev.ModPath != pi.ModPath {
		a.setDBConfig(nil)
		a.generator.CleanOptions()
	}
	a.setProjectInfo(pi)
	runtime.EventsEmit(a.ctx, "project-opened", pi)
	return &OpenProjectResult{Status: "opened", Project: pi}
}

// GetProjectInfo 返回当前打开的项目的信息。
func (a *App) GetProjectInfo() *detect.ProjectInfo {
	return a.getProjectInfo()
}

// SelectFolder 打开文件夹选择对话框，返回用户选择的文件夹路径。
func (a *App) SelectFolder() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "请选择一个文件夹",
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// GetGeneratorOptions 获取代码生成选项
func (a *App) GetGeneratorOptions() []*generator.Option {
	return a.generator.GetOptions()
}

// SetGeneratorOption 设置代码生成选项
func (a *App) SetGeneratorOption(options generator.GeneratorOptions) {
	a.generator.SetOptions(options)
}

// SetGenerateTests 开关 CRUD service 的单测骨架生成(fake 仓储)
func (a *App) SetGenerateTests(enable bool) {
	a.generator.SetGenerateTests(enable)
}

// EditGeneratorOption 编辑代码生成选项
func (a *App) EditGeneratorOption(o *generator.Option) {
	a.generator.EditOption(o)
}

// TestDatabaseConnection 测试数据库连接
func (a *App) TestDatabaseConnection(cfg database.DBConfig) (*database.ConnectionResult, error) {
	return database.TestConnection(cfg)
}

// GetDatabaseTables 获取表列表
func (a *App) GetDatabaseTables(cfg database.DBConfig) ([]database.TableInfo, error) {
	conn, err := database.Connect(cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return database.GetTables(conn, cfg.Type)
}

// GetTableColumns 获取某一个表的列信息
func (a *App) GetTableColumns(cfg database.DBConfig, tableName string) ([]database.ColumnInfo, error) {
	conn, err := database.Connect(cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return database.GetColumns(conn, cfg.Type, tableName)
}

// ImportSqlTables 导入 SQL 语句中的表名列表
func (a *App) ImportSqlTables(sqlContent string) string {
	if sqlContent == "" {
		runtime.LogErrorf(a.ctx, "SQL 内容为空，无法导入")
		return "SQL 内容为空，无法导入"
	}
	tables, err := ddlparser.ParseCreateTables(sqlContent)
	if err != nil {
		runtime.LogErrorf(a.ctx, "解析 SQL 语句失败: %v", err)
		return "解析 SQL 语句失败"
	}

	var tableNames []string
	for _, t := range tables {
		runtime.LogDebug(a.ctx, "解析到 CREATE TABLE 语句")
		runtime.LogDebugf(a.ctx, "表名: %v\n", t.Name)
		tableNames = append(tableNames, t.Name)
	}

	// 设置数据库配置，使用 SQL 内容作为数据源
	a.setDBConfig(&database.DBConfig{
		Type:       "mysql",
		UseDSN:     false,
		SQLContent: sqlContent,
	})

	a.generator.CleanOptions()
	for _, tableName := range tableNames {
		opt := &generator.Option{
			TableName: tableName,
		}
		a.generator.AddOption(opt)
	}

	runtime.EventsEmit(a.ctx, "table-imported")

	return ""
}

// ImportDatabaseTables 导入数据库表中的表名列表
func (a *App) ImportDatabaseTables(cfg database.DBConfig) string {
	conn, err := database.Connect(cfg)
	if err != nil {
		runtime.LogErrorf(a.ctx, "连接数据库失败: %v", err)
		return "连接数据库失败"
	}
	defer conn.Close()

	tables, err := database.GetTables(conn, cfg.Type)
	if err != nil {
		runtime.LogErrorf(a.ctx, "获取数据库表失败: %v", err)
		return "获取数据库表失败"
	}

	//runtime.LogInfof(a.ctx, "[%v] 导入的表名列表: [%v]", cfg.Type, tables)

	a.generator.CleanOptions()
	for _, t := range tables {
		opt := &generator.Option{
			TableName: t.Name,
		}
		a.generator.AddOption(opt)
	}

	runtime.EventsEmit(a.ctx, "table-imported")

	return ""
}

// ImportGoSchemaTables 从 Go 源码数据源导入表清单：ent://<ent schema 目录> 或
// gorm://<gorm model 目录>。与 ImportDatabaseTables 的区别是不连库——表结构由
// go/ast 解析源码得到；scheme 与 ORM 必须配对，否则 sqlkratos 会拖到生成阶段才报错。
func (a *App) ImportGoSchemaTables(source string, ormType string) string {
	source = strings.TrimSpace(source)
	ormType = strings.TrimSpace(ormType)

	var scheme string
	switch {
	case strings.HasPrefix(source, "ent://"):
		scheme = "ent"
	case strings.HasPrefix(source, "gorm://"):
		scheme = "gorm"
	default:
		return "Go 源码数据源需写成 ent://<schema 目录> 或 gorm://<model 目录>"
	}
	if scheme != ormType {
		return fmt.Sprintf("%s:// 数据源只能搭配 %s ORM（当前为 %q）", scheme, scheme, ormType)
	}

	dir := strings.TrimPrefix(source, scheme+"://")
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Sprintf("目录不存在：%s", dir)
	}

	names, err := planGoSchemaTables(a.ctx, source, ormType)
	if err != nil {
		return fmt.Sprintf("解析 Go 源码失败：%v", err)
	}
	if len(names) == 0 {
		return "该目录下没有解析到任何模型"
	}

	// Driver 留空无妨：source 自带 scheme，ensureDSNScheme 会原样透传给 sqlkratos。
	a.setDBConfig(&database.DBConfig{UseDSN: true, DSN: source})

	a.generator.CleanOptions()
	for _, name := range names {
		a.generator.AddOption(&generator.Option{TableName: name})
	}

	runtime.EventsEmit(a.ctx, "table-imported")

	return ""
}

// planGoSchemaTables 只读解析 Go 源码数据源将被处理的表名。
// Servers 必须非空（generateProtobufCode 只在 grpc/rest 分支执行转换），且转换会把
// proto 写到 OutputPath 下，故指向临时目录保证零副作用。
func planGoSchemaTables(ctx context.Context, source string, ormType string) ([]string, error) {
	tmp, err := os.MkdirTemp("", "gowind-schema-preview")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	tables, err := sqlkratos.PlanTables(ctx, sqlkratos.GeneratorOptions{
		Source:               source,
		OrmType:              ormType,
		OutputPath:           tmp,
		ModuleName:           "preview",
		SourceModuleName:     "preview",
		ModuleVersion:        "v1",
		ProjectName:          "preview",
		ServiceName:          "preview",
		Servers:              []string{"grpc"},
		ProtoPackageStrategy: "per-table",
	})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(tables))
	for _, t := range tables {
		names = append(names, t.Name)
	}
	return names, nil
}

// SetDBConfig 设置数据库连接配置
func (a *App) SetDBConfig(cfg database.DBConfig) {
	a.setDBConfig(&cfg)
}

// GetDBConfig 获取数据库连接配置
func (a *App) GetDBConfig() *database.DBConfig {
	return a.getDBConfig()
}

func (a *App) CleanConfig() {
	a.setProjectInfo(nil)
	a.setDBConfig(nil)
	a.generator.CleanOptions()

	runtime.EventsEmit(a.ctx, "config-cleaned")
}

// GenerateGrpcCode 生成代码
func (a *App) GenerateGrpcCode(ormType string, protoPackageStrategy string, servers []string) string {
	if ormType == "" {
		runtime.LogErrorf(a.ctx, "ORM 类型不能为空")
		return "ORM 类型不能为空"
	}

	pi := a.getProjectInfo()
	if pi == nil {
		runtime.LogErrorf(a.ctx, "未打开项目，无法生成代码")
		return "未打开项目，无法生成代码"
	}

	dbCfg := a.getDBConfig()
	if dbCfg == nil {
		runtime.LogErrorf(a.ctx, "未配置数据库连接，无法生成代码")
		return "未配置数据库连接，无法生成代码"
	}

	if protoPackageStrategy == "" {
		protoPackageStrategy = "per-table"
	}

	runtime.LogDebugf(a.ctx, "生成代码，ORM 类型: %v，Proto 包策略: %v，传输层: %v", ormType, protoPackageStrategy, servers)

	if err := a.generator.GenerateGrpcCode(
		a.ctx,
		*dbCfg,
		ormType,
		protoPackageStrategy,
		pi.Root,
		pi.ModPath,
		servers,
	); err != nil {
		runtime.LogErrorf(a.ctx, "生成代码失败: %v", err)
		return fmt.Sprintf("生成代码失败: %v", err)
	}

	runtime.EventsEmit(a.ctx, "code-generated")

	return ""
}

// GenerateRestCode 生成代码
func (a *App) GenerateRestCode(serviceName string, protoPackageStrategy string) string {
	// serviceName 会进 GeneratorOptions.ServiceName,而生成器拿它拼输出目录。
	if err := svcname.Validate(serviceName); err != nil {
		runtime.LogErrorf(a.ctx, "服务名称无效: %v", err)
		return err.Error()
	}

	pi := a.getProjectInfo()
	if pi == nil {
		runtime.LogErrorf(a.ctx, "未打开项目，无法生成代码")
		return "未打开项目，无法生成代码"
	}

	dbCfg := a.getDBConfig()
	if dbCfg == nil {
		runtime.LogErrorf(a.ctx, "未配置数据库连接，无法生成代码")
		return "未配置数据库连接，无法生成代码"
	}

	if protoPackageStrategy == "" {
		protoPackageStrategy = "per-table"
	}

	runtime.LogDebugf(a.ctx, "生成代码，服务名称: %v，Proto 包策略: %v", serviceName, protoPackageStrategy)

	if err := a.generator.GenerateRestCode(
		a.ctx,
		serviceName,
		"", // REST服务不生成ORM代码
		protoPackageStrategy,
		*dbCfg,
		pi.Root,
		pi.ModPath,
	); err != nil {
		runtime.LogErrorf(a.ctx, "生成代码失败: %v", err)
		return fmt.Sprintf("生成代码失败: %v", err)
	}

	runtime.EventsEmit(a.ctx, "code-generated")

	return ""
}

// FrontendGenParams 前端代码生成参数
type FrontendGenParams struct {
	// OpenapiYaml OpenAPI 规范文本（YAML/JSON）
	OpenapiYaml string `json:"openapiYaml"`
	// Framework 目标框架: vue-element / vue-vben / react
	Framework string `json:"framework"`
	// Tags 要生成的服务 tag 名；空 = 全部
	Tags []string `json:"tags"`
	// GenerateTypes 文件类型（composable/page/drawer/router/locale；react 的 composable 自动映射为 hooks）
	GenerateTypes []string `json:"generateTypes"`
	// ServiceName 生成代码的服务名（默认 admin）
	ServiceName string `json:"serviceName"`
	// ModulePathMap 文件名 -> 模块路径（如 role -> permission/role）
	ModulePathMap map[string]string `json:"modulePathMap"`
	// AutoRouterModules 未显式提供路由分组时按 basePath 自动检测
	AutoRouterModules bool `json:"autoRouterModules"`
}

// FrontendPreviewResult 前端代码生成预览结果
type FrontendPreviewResult struct {
	Files []frontendgen.GeneratedFile `json:"files"`
	Error string                      `json:"error,omitempty"`
}

// FrontendWriteResult 前端代码生成写盘结果
type FrontendWriteResult struct {
	Results []frontendgen.WriteResult `json:"results"`
	Error   string                    `json:"error,omitempty"`
}

// FrontendServicesResult OpenAPI 服务解析结果
type FrontendServicesResult struct {
	Services []*frontendgen.ParsedService `json:"services"`
	Error    string                       `json:"error,omitempty"`
}

// ParseFrontendServices 解析 OpenAPI 规范，返回服务列表（用于生成前的勾选）。
func (a *App) ParseFrontendServices(openapiYaml string) *FrontendServicesResult {
	spec, err := frontendgen.ParseOpenAPIYAML([]byte(openapiYaml))
	if err != nil {
		runtime.LogErrorf(a.ctx, "解析 OpenAPI 失败: %v", err)
		return &FrontendServicesResult{Error: err.Error()}
	}
	return &FrontendServicesResult{Services: frontendgen.ExtractServices(spec)}
}

// PreviewFrontendCode 预览前端代码生成结果（不写盘）。
func (a *App) PreviewFrontendCode(params FrontendGenParams) *FrontendPreviewResult {
	files, err := a.buildFrontendFiles(params)
	if err != nil {
		runtime.LogErrorf(a.ctx, "前端代码生成失败: %v", err)
		return &FrontendPreviewResult{Error: err.Error()}
	}
	return &FrontendPreviewResult{Files: files}
}

// GenerateFrontendCode 生成前端代码并写入目标目录（outDir 为前端项目 src 目录）。
// 返回的 Error 为空串表示成功。
func (a *App) GenerateFrontendCode(outDir string, params FrontendGenParams) *FrontendWriteResult {
	files, err := a.buildFrontendFiles(params)
	if err != nil {
		runtime.LogErrorf(a.ctx, "前端代码生成失败: %v", err)
		return &FrontendWriteResult{Error: err.Error()}
	}

	results, err := frontendgen.WriteFiles(files, outDir)
	if err != nil {
		runtime.LogErrorf(a.ctx, "前端代码写盘失败: %v", err)
		return &FrontendWriteResult{Error: err.Error()}
	}

	runtime.EventsEmit(a.ctx, "frontend-code-generated", results)

	return &FrontendWriteResult{Results: results}
}

func (a *App) buildFrontendFiles(params FrontendGenParams) ([]frontendgen.GeneratedFile, error) {
	framework, ok := frontendgen.ParseFramework(params.Framework)
	if !ok {
		return nil, fmt.Errorf("不支持的前端框架: %s（可选 vue-element / vue-vben / react）", params.Framework)
	}

	spec, err := frontendgen.ParseOpenAPIYAML([]byte(params.OpenapiYaml))
	if err != nil {
		return nil, fmt.Errorf("解析 OpenAPI 失败: %w", err)
	}

	return frontendgen.Generate(frontendgen.Options{
		Spec:              spec,
		Framework:         framework,
		Tags:              params.Tags,
		ServiceName:       params.ServiceName,
		ModulePathMap:     params.ModulePathMap,
		GenerateTypes:     params.GenerateTypes,
		AutoRouterModules: params.AutoRouterModules,
	})
}

// ==================== AI 助手相关方法 ====================

// GetAIConfig 获取 AI 配置
func (a *App) GetAIConfig() *ai.Config {
	return a.aiService.GetConfig()
}

// SetAIConfig 设置 AI 配置
func (a *App) SetAIConfig(config ai.Config) {
	a.aiService.SetConfig(&config)
}

// GetAIProviderPresets 获取 AI 服务商预设列表
func (a *App) GetAIProviderPresets() []ai.AIProviderPreset {
	return ai.GetProviderPresets()
}

// AICancelTask 取消一个在途 AI 任务("ddl"/"partition"/"review")。
// 在途调用自己的返回路径会以"已取消"失败收场,前端据此复位加载态。
func (a *App) AICancelTask(task string) bool {
	return a.aiService.CancelTask(task)
}

// TestAIConnection 测试 AI 连接
func (a *App) TestAIConnection() *ai.StepResult {
	result, err := a.aiService.TestConnection()
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 连接测试失败: %v", err)
		return &ai.StepResult{Success: false, Error: err.Error()}
	}
	return result
}

// AIGenerateDDL 根据需求文档使用 AI 生成 DDL
func (a *App) AIGenerateDDL(requirements string) *ai.StepResult {
	if requirements == "" {
		runtime.LogErrorf(a.ctx, "需求文档不能为空")
		return &ai.StepResult{Success: false, Error: "需求文档不能为空"}
	}

	result, err := a.aiService.GenerateDDL(requirements)
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 生成 DDL 失败: %v", err)
		return &ai.StepResult{Success: false, Error: err.Error()}
	}

	// 与流式版本一致:需求与 DDL 自动落盘。
	a.saveAIArtifactQuietly(ai.ArtifactRequirements, requirements)
	a.saveAIArtifactQuietly(ai.ArtifactDDL, result.Content)
	runtime.EventsEmit(a.ctx, "ai-ddl-generated")
	return result
}

// AIGenerateDDLStream AIGenerateDDL 的流式版本:生成过程中的增量内容经
// "ai:stream" 事件逐块推送(task 标识任务类型,attempt 标识第几次尝试——
// 校验失败自动修复重试时前端须清空已累积内容),结束发 "ai:done";
// 返回值为完整结果。
func (a *App) AIGenerateDDLStream(requirements string) *ai.StepResult {
	if requirements == "" {
		runtime.LogErrorf(a.ctx, "需求文档不能为空")
		return &ai.StepResult{Success: false, Error: "需求文档不能为空"}
	}

	result, err := a.aiService.GenerateDDLValidated(requirements, func(attempt int, delta string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "ai:stream", map[string]any{"task": "ddl", "delta": delta, "attempt": attempt})
		}
	})
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 生成 DDL 失败: %v", err)
		runtime.EventsEmit(a.ctx, "ai:done", map[string]any{"task": "ddl", "success": false, "error": err.Error()})
		return &ai.StepResult{Success: false, Error: err.Error()}
	}

	runtime.EventsEmit(a.ctx, "ai:done", map[string]any{"task": "ddl", "success": true})
	// 产物自动落盘:需求与 DDL 一起存,后者是重跑划分/生成链的输入。
	a.saveAIArtifactQuietly(ai.ArtifactRequirements, requirements)
	a.saveAIArtifactQuietly(ai.ArtifactDDL, result.Content)
	runtime.EventsEmit(a.ctx, "ai-ddl-generated")
	return result
}

// AIPartitionMicroservices 根据 DDL 使用 AI 建议微服务划分
func (a *App) AIPartitionMicroservices(ddl string) *ai.PartitionResult {
	if ddl == "" {
		runtime.LogErrorf(a.ctx, "DDL 不能为空")
		return &ai.PartitionResult{Success: false, Error: "DDL 不能为空"}
	}

	partitions, ddlTables, warnings, err := a.aiService.PartitionMicroservicesValidatedStream(ddl, func(attempt int, delta string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "ai:stream", map[string]any{"task": "partition", "delta": delta, "attempt": attempt})
		}
	})
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 微服务划分失败: %v", err)
		return &ai.PartitionResult{Success: false, Error: err.Error()}
	}

	// 划分结果自动落盘,供换项目/重开应用后恢复。
	if pi := a.getProjectInfo(); pi != nil {
		if _, err := ai.SavePartitionsArtifact(pi.Root, partitions); err != nil {
			runtime.LogWarningf(a.ctx, "划分结果落盘失败: %v", err)
		}
	}

	return &ai.PartitionResult{Success: true, Partitions: partitions, AvailableTables: ddlTables, Warnings: warnings}
}

// AIValidateDDL 校验 DDL 可解析并返回表名清单与语法诊断(带行列)。
// 前端编辑划分时进入编辑态用它取"未分配池"的基准清单;DDL 卡片的
// "校验语法"按钮用它把诊断画进 Monaco 编辑器。
func (a *App) AIValidateDDL(ddl string) *ai.DDLValidateResult {
	tables, diags, err := ai.ValidateDDLDetailed(ddl)
	if err != nil {
		return &ai.DDLValidateResult{Success: false, Tables: tables, Diagnostics: diags, Error: err.Error()}
	}
	return &ai.DDLValidateResult{Success: true, Tables: tables}
}

// AIGenerateBackendCode AI 辅助生成后端代码(直连旧路径,保留给 CLI/兼容;
// GUI 走 AIPlanBackendCode → AIApplyBackendCode 的预览确认流)
func (a *App) AIGenerateBackendCode(ddl string, ormType string, partitions []ai.MicroservicePartition) string {
	pi := a.getProjectInfo()
	if pi == nil {
		runtime.LogErrorf(a.ctx, "未打开项目，无法生成代码")
		return "未打开项目，无法生成代码"
	}

	if ddl == "" {
		runtime.LogErrorf(a.ctx, "DDL 不能为空")
		return "DDL 不能为空"
	}

	dbCfg := a.assembleBackendOptions(ddl, partitions)

	// 生成 gRPC 代码
	if err := a.generator.GenerateGrpcCode(
		a.ctx,
		*dbCfg,
		ormType,
		"per-table", // AI 辅助生成默认使用每表独立包
		pi.Root,
		pi.ModPath,
		[]string{"grpc"},
	); err != nil {
		runtime.LogErrorf(a.ctx, "AI 辅助生成后端代码失败: %v", err)
		return fmt.Sprintf("生成后端代码失败: %v", err)
	}

	runtime.EventsEmit(a.ctx, "ai-backend-generated")
	return ""
}

// ==================== AI 后端生成 Plan/确认 ====================

// assembleBackendOptions 装配 DDL 数据源与生成器选项(AI 直连路径与 Plan 路径共用)。
func (a *App) assembleBackendOptions(ddl string, partitions []ai.MicroservicePartition) *database.DBConfig {
	a.setDBConfig(&database.DBConfig{
		Type:       "mysql",
		UseDSN:     false,
		SQLContent: ddl,
	})
	a.generator.CleanOptions()
	for _, p := range partitions {
		for _, tableName := range p.Tables {
			a.generator.AddOption(&generator.Option{
				TableName: tableName,
				Service:   p.ServiceName,
			})
		}
	}
	return a.getDBConfig()
}

// backendPostProcessSteps 确认写入后将执行的后处理步骤说明。
func backendPostProcessSteps(ormType string, services []string) []string {
	steps := []string{"go mod tidy", "buf generate"}
	if ormType == "ent" {
		for _, svc := range services {
			steps = append(steps, "ent generate ("+svc+")")
		}
	}
	for _, svc := range services {
		steps = append(steps, "wire ("+svc+")")
	}
	return steps
}

// clearBackendPlanLocked 清理当前计划(须持 planMu)。
func (a *App) clearBackendPlanLocked() {
	if a.backendPlan != nil {
		os.RemoveAll(a.backendPlan.dir)
		a.backendPlan = nil
	}
}

// AIPlanBackendCode 预演后端生成:完整生成树落在临时目录(跳过后处理链),
// 返回相对项目根的文件清单与新增/覆盖标注,供前端确认后再真正写入。
// 同一时刻只保留一份计划;再次预览会丢弃上一份。
func (a *App) AIPlanBackendCode(ddl string, ormType string, partitions []ai.MicroservicePartition) *ai.BackendPlanResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &ai.BackendPlanResult{Success: false, Error: "未打开项目"}
	}
	if ddl == "" {
		return &ai.BackendPlanResult{Success: false, Error: "DDL 不能为空"}
	}
	if len(partitions) == 0 {
		return &ai.BackendPlanResult{Success: false, Error: "请先完成微服务划分"}
	}

	// 服务名统一把关:编辑划分可能引入怪名,预览(将来还要落盘)前先拒收。
	services := make([]string, 0, len(partitions))
	seen := map[string]bool{}
	for _, p := range partitions {
		if err := svcname.Validate(p.ServiceName); err != nil {
			return &ai.BackendPlanResult{Success: false, Error: fmt.Sprintf("服务名非法: %v", err)}
		}
		if !seen[p.ServiceName] {
			seen[p.ServiceName] = true
			services = append(services, p.ServiceName)
		}
	}

	a.planMu.Lock()
	defer a.planMu.Unlock()
	a.clearBackendPlanLocked()

	tmpDir, err := os.MkdirTemp("", "gowind-backend-plan")
	if err != nil {
		return &ai.BackendPlanResult{Success: false, Error: fmt.Sprintf("创建预览目录失败: %v", err)}
	}

	dbCfg := a.assembleBackendOptions(ddl, partitions)
	// entimport 要求 schema 目录处于 Go 模块内;预览目录是空的,补一个与真实
	// 项目同 module path 的最小 go.mod。生成完成后必须立即删除——它若留在
	// 树里,Apply 拷贝时会把用户真实的 go.mod 覆盖掉。
	stubGoMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(stubGoMod, []byte("module "+pi.ModPath+"\n"), 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return &ai.BackendPlanResult{Success: false, Error: fmt.Sprintf("初始化预览目录失败: %v", err)}
	}
	// 预览阶段只跑 sqlkratos 生成(纯模板渲染,不依赖外部工具);
	// 后处理链需要真实项目的 go.mod/buf.yaml 模板,留到 Apply 在项目根执行。
	a.generator.SetSkipPostProcess(true)
	genErr := a.generator.GenerateGrpcCode(a.ctx, *dbCfg, ormType, "per-table", tmpDir, pi.ModPath, []string{"grpc"})
	a.generator.SetSkipPostProcess(false)
	os.Remove(stubGoMod)
	if genErr != nil {
		os.RemoveAll(tmpDir)
		if a.ctx != nil {
			runtime.LogErrorf(a.ctx, "后端生成预览失败: %v", genErr)
		}
		return &ai.BackendPlanResult{Success: false, Error: fmt.Sprintf("生成预览失败: %v", genErr)}
	}

	files, werr := ai.WalkPlanFiles(tmpDir, pi.Root)
	if werr != nil {
		os.RemoveAll(tmpDir)
		return &ai.BackendPlanResult{Success: false, Error: werr.Error()}
	}

	newCount, overwriteCount := 0, 0
	for _, f := range files {
		if f.IsNew {
			newCount++
		} else {
			overwriteCount++
		}
	}

	plan := &backendPlanState{
		id:       ai.NewPlanID(),
		dir:      tmpDir,
		services: services,
		modPath:  pi.ModPath,
		ormType:  ormType,
	}
	a.backendPlan = plan

	return &ai.BackendPlanResult{
		Success:        true,
		PlanID:         plan.id,
		Files:          files,
		NewCount:       newCount,
		OverwriteCount: overwriteCount,
		PostProcess:    backendPostProcessSteps(ormType, services),
	}
}

// AIApplyBackendCode 把预览计划写入项目:逐字拷贝预览树(与所见清单严格一致),
// 再按顺序执行后处理链(go mod tidy / buf generate / ent generate / wire)。
// 返回空串表示成功,非空为错误信息。
func (a *App) AIApplyBackendCode(planID string) string {
	pi := a.getProjectInfo()
	if pi == nil {
		return "未打开项目"
	}

	a.planMu.Lock()
	plan := a.backendPlan
	a.planMu.Unlock()
	if plan == nil || plan.id != planID {
		return "生成计划不存在或已过期，请重新预览"
	}
	if plan.modPath != pi.ModPath {
		a.AIDiscardBackendPlan(plan.id)
		return "项目已切换，原计划已作废，请重新预览"
	}

	if err := ai.CopyTree(pi.Root, plan.dir); err != nil {
		runtime.LogErrorf(a.ctx, "应用生成计划失败(拷贝阶段): %v", err)
		a.AIDiscardBackendPlan(plan.id)
		return fmt.Sprintf("写入文件失败: %v", err)
	}

	// 后处理链与 generator.GenerateGrpcCode 内联链一致,在真实项目根执行
	// (Run* 返回 *CommandResult,以 Success 字段判定成败);失败不回滚已拷贝
	// 文件,错误信息注明"文件已写入"以便用户定位。
	if result := devtools.RunGoModTidy(pi.Root); !result.Success {
		a.AIDiscardBackendPlan(plan.id)
		return fmt.Sprintf("文件已写入，但 go mod tidy 失败: %s", result.Error)
	}
	if result := devtools.RunBufGenerate(pi.Root); !result.Success {
		a.AIDiscardBackendPlan(plan.id)
		return fmt.Sprintf("文件已写入，但 buf generate 失败: %s", result.Error)
	}
	if plan.ormType == "ent" {
		for _, svc := range plan.services {
			if result := devtools.RunEntGenerate(pi.Root, svc); !result.Success {
				a.AIDiscardBackendPlan(plan.id)
				return fmt.Sprintf("文件已写入，但 ent generate(%s) 失败: %s", svc, result.Error)
			}
		}
	}
	for _, svc := range plan.services {
		if result := devtools.RunWire(pi.Root, svc); !result.Success {
			a.AIDiscardBackendPlan(plan.id)
			return fmt.Sprintf("文件已写入，但 wire(%s) 失败: %s", svc, result.Error)
		}
	}

	a.AIDiscardBackendPlan(plan.id)
	runtime.EventsEmit(a.ctx, "code-generated")
	runtime.EventsEmit(a.ctx, "ai-backend-generated")
	return ""
}

// AIDiscardBackendPlan 丢弃当前预览计划并清理临时目录(planID 传空串丢弃任意)。
func (a *App) AIDiscardBackendPlan(planID string) {
	a.planMu.Lock()
	defer a.planMu.Unlock()
	if a.backendPlan != nil && (planID == "" || a.backendPlan.id == planID) {
		a.clearBackendPlanLocked()
	}
}

// AIReadBackendPlanFile 读取预览树内单个文件内容,供前端 drawer 预览。
func (a *App) AIReadBackendPlanFile(planID string, relPath string) *ai.StepResult {
	a.planMu.Lock()
	plan := a.backendPlan
	a.planMu.Unlock()
	if plan == nil || plan.id != planID {
		return &ai.StepResult{Success: false, Error: "生成计划不存在或已过期"}
	}
	content, err := ai.ReadPlanFile(plan.dir, relPath)
	if err != nil {
		return &ai.StepResult{Success: false, Error: err.Error()}
	}
	return &ai.StepResult{Success: true, Content: content}
}

// AIFindOpenAPIFiles 在项目中查找 OpenAPI 文件
func (a *App) AIFindOpenAPIFiles() *ai.OpenAPIResult {
	pi := a.getProjectInfo()
	if pi == nil {
		runtime.LogErrorf(a.ctx, "未打开项目")
		return &ai.OpenAPIResult{Success: false, Error: "未打开项目"}
	}

	files, err := ai.FindOpenAPIFiles(pi.Root)
	if err != nil {
		runtime.LogErrorf(a.ctx, "查找 OpenAPI 文件失败: %v", err)
		return &ai.OpenAPIResult{Success: false, Error: err.Error()}
	}

	if len(files) == 0 {
		return &ai.OpenAPIResult{Success: true, Files: files, Message: "未找到 OpenAPI 文件"}
	}

	return &ai.OpenAPIResult{Success: true, Files: files}
}

// AIReviewCode 使用 AI 审查项目代码
func (a *App) AIReviewCode(fileContents map[string]string) *ai.StepResult {
	if len(fileContents) == 0 {
		runtime.LogErrorf(a.ctx, "没有可审查的代码文件")
		return &ai.StepResult{Success: false, Error: "没有可审查的代码文件"}
	}

	result, err := a.aiService.ReviewCode(fileContents)
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 代码审查失败: %v", err)
		return &ai.StepResult{Success: false, Error: err.Error()}
	}

	return result
}

// AIReviewCodeStream AIReviewCode 的流式版本:审查意见经 "ai:stream" 事件
// 逐块推送,结束发 "ai:done";返回值为完整结果。
func (a *App) AIReviewCodeStream(fileContents map[string]string) *ai.StepResult {
	if len(fileContents) == 0 {
		runtime.LogErrorf(a.ctx, "没有可审查的代码文件")
		return &ai.StepResult{Success: false, Error: "没有可审查的代码文件"}
	}

	result, err := a.aiService.ReviewCodeStream(fileContents, func(delta string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "ai:stream", map[string]string{"task": "review", "delta": delta})
		}
	})
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 代码审查失败: %v", err)
		runtime.EventsEmit(a.ctx, "ai:done", map[string]any{"task": "review", "success": false, "error": err.Error()})
		return &ai.StepResult{Success: false, Error: err.Error()}
	}

	runtime.EventsEmit(a.ctx, "ai:done", map[string]any{"task": "review", "success": true})
	return result
}

// ==================== 项目代码扫描与审查 ====================

// AIReviewScan 扫描项目内可审查的代码文件:app/ 下 Go 源码(排除测试与
// protoc 生成物)与 api/protos/ 下 proto,返回相对路径清单供前端勾选。
func (a *App) AIReviewScan() *ai.ReviewScanResult {
	pi := a.getProjectInfo()
	if pi == nil {
		runtime.LogErrorf(a.ctx, "未打开项目")
		return &ai.ReviewScanResult{Success: false, Error: "未打开项目"}
	}

	res, err := ai.ScanProjectCode(pi.Root)
	if err != nil {
		runtime.LogErrorf(a.ctx, "扫描项目代码失败: %v", err)
		return &ai.ReviewScanResult{Success: false, Error: err.Error()}
	}
	return res
}

// AIReviewProject 对选中的项目文件做 AI 审查(map-reduce):
// 按服务贪心装箱分批,逐批结构化审查(每批发 ai:review-batch 进度事件),
// 全部完成后流式汇总总评(ai:stream),最后 md+json 双落盘 .gowind/ai/。
// 旧 AIReviewCodeStream 保留给 CLI 与自定义文件集场景。
// 取消粒度说明:批次执行中即时生效;批次间隙(毫秒级)可能错过一次取消。
func (a *App) AIReviewProject(paths []string) *ai.ReviewProjectResult {
	pi := a.getProjectInfo()
	if pi == nil {
		runtime.LogErrorf(a.ctx, "未打开项目")
		return &ai.ReviewProjectResult{Success: false, Error: "未打开项目"}
	}

	fileContents, warnings, err := ai.ReadProjectFiles(pi.Root, paths)
	if err != nil {
		runtime.LogErrorf(a.ctx, "读取待审查文件失败: %v", err)
		return &ai.ReviewProjectResult{Success: false, Error: err.Error()}
	}
	for _, w := range warnings {
		runtime.LogWarningf(a.ctx, "审查文件: %s", w)
	}

	// 从已读内容重建装箱所需的大小信息
	infos := make([]ai.CodeFileInfo, 0, len(fileContents))
	for path, content := range fileContents {
		infos = append(infos, ai.CodeFileInfo{Path: path, Size: int64(len(content))})
	}
	batches := ai.BuildReviewBatches(infos)
	if len(batches) == 0 {
		return &ai.ReviewProjectResult{Success: false, Error: "没有可审查的文件"}
	}

	result := &ai.ReviewProjectResult{Success: true}
	var allIssues []ai.ReviewIssue
	var summaries []string

	for i, batch := range batches {
		batchContents := make(map[string]string, len(batch))
		for _, f := range batch {
			batchContents[f.Path] = fileContents[f.Path]
		}

		review, err := a.aiService.ReviewCodeStructured(batchContents)
		if err != nil {
			runtime.LogErrorf(a.ctx, "AI 代码审查失败(批次 %d/%d): %v", i+1, len(batches), err)
			runtime.EventsEmit(a.ctx, "ai:done", map[string]any{"task": "review", "success": false, "error": err.Error()})
			return &ai.ReviewProjectResult{Success: false, Batches: result.Batches, Issues: allIssues, Error: err.Error()}
		}

		outcome := ai.ReviewBatchOutcome{
			Label:      ai.BatchLabel(batch),
			Summary:    review.Summary,
			IssueCount: len(review.Issues),
		}
		result.Batches = append(result.Batches, outcome)
		allIssues = append(allIssues, review.Issues...)
		summaries = append(summaries, fmt.Sprintf("### %s(%d 个问题)\n%s", outcome.Label, outcome.IssueCount, outcome.Summary))

		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "ai:review-batch", map[string]any{
				"index":      i + 1,
				"total":      len(batches),
				"label":      outcome.Label,
				"issueCount": outcome.IssueCount,
				"summary":    outcome.Summary,
			})
		}
	}
	result.Issues = allIssues

	synthesis, err := a.aiService.SynthesizeReview(summaries, func(delta string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "ai:stream", map[string]any{"task": "review", "delta": delta})
		}
	})
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 审查汇总失败: %v", err)
		runtime.EventsEmit(a.ctx, "ai:done", map[string]any{"task": "review", "success": false, "error": err.Error()})
		return &ai.ReviewProjectResult{Success: false, Batches: result.Batches, Issues: allIssues, Error: err.Error()}
	}
	result.Synthesis = synthesis

	// md+json 双落盘;失败不影响审查结果本身。
	mdPath, jsonPath, serr := ai.SaveReviewReports(pi.Root, result.Batches, allIssues, synthesis)
	if serr != nil {
		runtime.LogWarningf(a.ctx, "审查报告落盘失败: %v", serr)
	} else {
		result.ReportPath = mdPath
		runtime.LogInfof(a.ctx, "审查报告已保存: %s, %s", mdPath, jsonPath)
		runtime.EventsEmit(a.ctx, "ai-review-saved", map[string]any{"path": mdPath})
	}

	runtime.EventsEmit(a.ctx, "ai:done", map[string]any{"task": "review", "success": true})
	return result
}

// ==================== AI 产物落盘 ====================

// saveAIArtifactQuietly 把 AI 产物落盘到项目 .gowind/ai/ 目录。
// 落盘失败仅告警不阻断主流程(与 AI 配置持久化同策略)。
func (a *App) saveAIArtifactQuietly(kind ai.ArtifactKind, content string) {
	pi := a.getProjectInfo()
	if pi == nil {
		return
	}
	if _, err := ai.SaveArtifact(pi.Root, kind, content); err != nil {
		runtime.LogWarningf(a.ctx, "AI 产物(%s)落盘失败: %v", kind, err)
	}
}

// SaveAIArtifact 把用户手动编辑的产物(需求/DDL)保存到项目 .gowind/ai/ 目录。
func (a *App) SaveAIArtifact(kind string, content string) *ai.ArtifactSaveResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &ai.ArtifactSaveResult{Success: false, Error: "未打开项目"}
	}
	path, err := ai.SaveArtifact(pi.Root, ai.ArtifactKind(kind), content)
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 产物保存失败: %v", err)
		return &ai.ArtifactSaveResult{Success: false, Error: err.Error()}
	}
	return &ai.ArtifactSaveResult{Success: true, Path: path}
}

// LoadAIArtifacts 读取项目内已保存的 AI 产物(打开项目/进入面板时恢复现场)。
func (a *App) LoadAIArtifacts() *ai.Artifacts {
	pi := a.getProjectInfo()
	if pi == nil {
		return &ai.Artifacts{}
	}
	return ai.LoadArtifacts(pi.Root)
}

// LoadAIReviewReport 读取一份已落盘的审查报告内容,供前端回看。
func (a *App) LoadAIReviewReport(name string) *ai.StepResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &ai.StepResult{Success: false, Error: "未打开项目"}
	}
	content, err := ai.LoadReviewReport(pi.Root, name)
	if err != nil {
		runtime.LogErrorf(a.ctx, "读取审查报告失败: %v", err)
		return &ai.StepResult{Success: false, Error: err.Error()}
	}
	return &ai.StepResult{Success: true, Content: content}
}

// ==================== 远程配置相关方法 ====================

// GetRemoteConfigTypes 获取支持的远程配置中心类型
func (a *App) GetRemoteConfigTypes() []map[string]string {
	return ce.GetSupportedTypes()
}

// GetConfigServices 获取项目中的服务配置信息
func (a *App) GetConfigServices() ([]ce.ServiceInfo, error) {
	pi := a.getProjectInfo()
	if pi == nil {
		return nil, fmt.Errorf("未打开项目")
	}
	return ce.GetServiceList(pi.Root)
}

// ExportConfigToRemote 导出所有服务配置到远程配置中心
func (a *App) ExportConfigToRemote(cfg ce.RemoteConfig) *ce.ExportResult {
	pi := a.getProjectInfo()
	if pi == nil {
		runtime.LogErrorf(a.ctx, "未打开项目")
		return &ce.ExportResult{Success: false, Error: "未打开项目"}
	}

	if errMsg := cfg.Validate(); errMsg != "" {
		runtime.LogErrorf(a.ctx, "配置验证失败: %s", errMsg)
		return &ce.ExportResult{Success: false, Error: errMsg}
	}

	err := ce.ExportAll(&cfg, pi.Root)
	if err != nil {
		runtime.LogErrorf(a.ctx, "导出配置失败: %v", err)
		return &ce.ExportResult{Success: false, Error: err.Error()}
	}

	runtime.EventsEmit(a.ctx, "config-exported")
	return &ce.ExportResult{Success: true}
}

// ==================== 项目管理相关方法 ====================

// GetDevServices 获取项目中的服务列表（详细信息）
func (a *App) GetDevServices() ([]devtools.ServiceInfo, error) {
	pi := a.getProjectInfo()
	if pi == nil {
		return nil, fmt.Errorf("未打开项目")
	}
	return devtools.GetServices(pi.Root)
}

// CreateProject 创建新项目
func (a *App) CreateProject(opts devtools.CreateProjectOptions) *devtools.CommandResult {
	return devtools.CreateProject(a.ctx, opts)
}

// AddService 向已有项目添加新服务
func (a *App) AddService(opts devtools.AddServiceOptions) *devtools.CommandResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	return devtools.AddService(pi.Root, opts)
}

// ==================== 开发工具相关方法 ====================

// DevRunService 运行指定服务（在终端窗口中运行）
func (a *App) DevRunService(serviceName string) *devtools.CommandResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	return devtools.RunServiceInTerminal(pi.Root, serviceName)
}

// DevBufGenerate 运行 buf generate
func (a *App) DevBufGenerate() *devtools.CommandResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	return devtools.RunBufGenerate(pi.Root)
}

// DevEntGenerate 运行 ent generate
func (a *App) DevEntGenerate(serviceName string) *devtools.CommandResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	if serviceName == "" {
		return devtools.RunEntGenerateAll(pi.Root)
	}
	return devtools.RunEntGenerate(pi.Root, serviceName)
}

// DevWireGenerate 运行 wire
func (a *App) DevWireGenerate(serviceName string) *devtools.CommandResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	if serviceName == "" {
		return devtools.RunWireAll(pi.Root)
	}
	return devtools.RunWire(pi.Root, serviceName)
}

// DevGoModTidy 运行 go mod tidy
func (a *App) DevGoModTidy() *devtools.CommandResult {
	pi := a.getProjectInfo()
	if pi == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	return devtools.RunGoModTidy(pi.Root)
}

// ExportOneServiceConfig 导出单个服务的配置到远程配置中心
func (a *App) ExportOneServiceConfig(cfg ce.RemoteConfig, serviceName string) *ce.ExportResult {
	pi := a.getProjectInfo()
	if pi == nil {
		runtime.LogErrorf(a.ctx, "未打开项目")
		return &ce.ExportResult{Success: false, Error: "未打开项目"}
	}

	if errMsg := cfg.Validate(); errMsg != "" {
		runtime.LogErrorf(a.ctx, "配置验证失败: %s", errMsg)
		return &ce.ExportResult{Success: false, Error: errMsg}
	}

	err := ce.ExportOne(&cfg, pi.Root, serviceName)
	if err != nil {
		runtime.LogErrorf(a.ctx, "导出服务 %s 配置失败: %v", serviceName, err)
		return &ce.ExportResult{Success: false, Error: err.Error(), Service: serviceName}
	}

	runtime.EventsEmit(a.ctx, "config-exported")
	return &ce.ExportResult{Success: true, Service: serviceName}
}
