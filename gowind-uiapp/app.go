package main

import (
	"context"
	"fmt"

	"github.com/tx7do/go-utils/ddl_parser"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/ai"
	ce "github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/configexporter"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/detect"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/devtools"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
)

// App struct
type App struct {
	ctx context.Context

	projectInfo *detect.ProjectInfo
	dbConfig    *database.DBConfig

	projectDetector *detect.ProjectDetector
	generator       *generator.Generator
	aiService       *ai.Service
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
}

// OpenProject 打开指定路径的项目，并返回项目的信息。
func (a *App) OpenProject(projectPath string) *detect.ProjectInfo {
	var err error
	var pi *detect.ProjectInfo
	pi, err = a.projectDetector.Detect(projectPath)
	if err != nil {
		return nil
	}
	a.projectInfo = pi

	runtime.EventsEmit(a.ctx, "project-opened", pi)

	return pi
}

// GetProjectInfo 返回当前打开的项目的信息。
func (a *App) GetProjectInfo() *detect.ProjectInfo {
	return a.projectInfo
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

	//runtime.LogInfof(a.ctx, "导入的表名列表: [%v]", tableNames)

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

// SetDBConfig 设置数据库连接配置
func (a *App) SetDBConfig(cfg database.DBConfig) {
	a.dbConfig = &cfg
}

// GetDBConfig 获取数据库连接配置
func (a *App) GetDBConfig() *database.DBConfig {
	return a.dbConfig
}

func (a *App) CleanConfig() {
	a.projectInfo = nil
	a.dbConfig = nil
	a.generator.CleanOptions()

	runtime.EventsEmit(a.ctx, "config-cleaned")
}

// GenerateGrpcCode 生成代码
func (a *App) GenerateGrpcCode(ormType string) string {
	if ormType == "" {
		runtime.LogErrorf(a.ctx, "ORM 类型不能为空")
		return "ORM 类型不能为空"
	}

	if a.projectInfo == nil {
		runtime.LogErrorf(a.ctx, "未打开项目，无法生成代码")
		return "未打开项目，无法生成代码"
	}

	if a.dbConfig == nil {
		runtime.LogErrorf(a.ctx, "未配置数据库连接，无法生成代码")
		return "未配置数据库连接，无法生成代码"
	}

	runtime.LogDebugf(a.ctx, "生成代码，ORM 类型: %v", ormType)

	if err := a.generator.GenerateGrpcCode(
		a.ctx,
		*a.dbConfig,
		ormType,
		a.projectInfo.Root,
		a.projectInfo.ModPath,
	); err != nil {
		runtime.LogErrorf(a.ctx, "生成代码失败: %v", err)
		return "生成代码失败"
	}

	runtime.EventsEmit(a.ctx, "code-generated")

	return ""
}

// GenerateRestCode 生成代码
func (a *App) GenerateRestCode(serviceName string) string {
	if len(serviceName) == 0 {
		runtime.LogErrorf(a.ctx, "服务名称不能为空")
		return "服务名称不能为空"
	}

	if a.projectInfo == nil {
		runtime.LogErrorf(a.ctx, "未打开项目，无法生成代码")
		return "未打开项目，无法生成代码"
	}

	if a.dbConfig == nil {
		runtime.LogErrorf(a.ctx, "未配置数据库连接，无法生成代码")
		return "未配置数据库连接，无法生成代码"
	}

	runtime.LogDebugf(a.ctx, "生成代码，服务名称: %v", serviceName)

	if err := a.generator.GenerateRestCode(
		a.ctx,
		serviceName,
		"", // REST服务不生成ORM代码
		*a.dbConfig,
		a.projectInfo.Root,
		a.projectInfo.ModPath,
	); err != nil {
		runtime.LogErrorf(a.ctx, "生成代码失败: %v", err)
		return "生成代码失败"
	}

	runtime.EventsEmit(a.ctx, "code-generated")

	return ""
}

func (a *App) GenerateFrontendCode(serviceName string, frontendType string) string {
	if len(serviceName) == 0 {
		runtime.LogErrorf(a.ctx, "服务名称不能为空")
		return "服务名称不能为空"
	}

	if len(frontendType) == 0 {
		runtime.LogErrorf(a.ctx, "前端类型不能为空")
		return "前端类型不能为空"
	}

	if a.projectInfo == nil {
		runtime.LogErrorf(a.ctx, "未打开项目，无法生成代码")
		return "未打开项目，无法生成代码"
	}

	if a.dbConfig == nil {
		runtime.LogErrorf(a.ctx, "未配置数据库连接，无法生成代码")
		return "未配置数据库连接，无法生成代码"
	}

	runtime.LogErrorf(a.ctx, "前端代码生成功能尚未实现")
	return "前端代码生成功能尚未实现"
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

	runtime.EventsEmit(a.ctx, "ai-ddl-generated")
	return result
}

// AIPartitionMicroservices 根据 DDL 使用 AI 建议微服务划分
func (a *App) AIPartitionMicroservices(ddl string) *ai.PartitionResult {
	if ddl == "" {
		runtime.LogErrorf(a.ctx, "DDL 不能为空")
		return &ai.PartitionResult{Success: false, Error: "DDL 不能为空"}
	}

	partitions, err := a.aiService.PartitionMicroservices(ddl)
	if err != nil {
		runtime.LogErrorf(a.ctx, "AI 微服务划分失败: %v", err)
		return &ai.PartitionResult{Success: false, Error: err.Error()}
	}

	return &ai.PartitionResult{Success: true, Partitions: partitions}
}

// AIGenerateBackendCode AI 辅助生成后端代码
func (a *App) AIGenerateBackendCode(ddl string, ormType string, partitions []ai.MicroservicePartition) string {
	if a.projectInfo == nil {
		runtime.LogErrorf(a.ctx, "未打开项目，无法生成代码")
		return "未打开项目，无法生成代码"
	}

	if ddl == "" {
		runtime.LogErrorf(a.ctx, "DDL 不能为空")
		return "DDL 不能为空"
	}

	// 设置数据库配置，使用 SQL 内容作为数据源
	a.dbConfig = &database.DBConfig{
		Type:       "mysql",
		UseDSN:     false,
		SQLContent: ddl,
	}

	// 清空并设置生成器选项
	a.generator.CleanOptions()
	for _, p := range partitions {
		for _, tableName := range p.Tables {
			opt := &generator.Option{
				TableName: tableName,
				Service:   p.ServiceName,
			}
			a.generator.AddOption(opt)
		}
	}

	// 生成 gRPC 代码
	if err := a.generator.GenerateGrpcCode(
		a.ctx,
		*a.dbConfig,
		ormType,
		a.projectInfo.Root,
		a.projectInfo.ModPath,
	); err != nil {
		runtime.LogErrorf(a.ctx, "AI 辅助生成后端代码失败: %v", err)
		return fmt.Sprintf("生成后端代码失败: %v", err)
	}

	runtime.EventsEmit(a.ctx, "ai-backend-generated")
	return ""
}

// AIFindOpenAPIFiles 在项目中查找 OpenAPI 文件
func (a *App) AIFindOpenAPIFiles() *ai.OpenAPIResult {
	if a.projectInfo == nil {
		runtime.LogErrorf(a.ctx, "未打开项目")
		return &ai.OpenAPIResult{Success: false, Error: "未打开项目"}
	}

	files, err := ai.FindOpenAPIFiles(a.projectInfo.Root)
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

// ==================== 远程配置相关方法 ====================

// GetRemoteConfigTypes 获取支持的远程配置中心类型
func (a *App) GetRemoteConfigTypes() []map[string]string {
	return ce.GetSupportedTypes()
}

// GetConfigServices 获取项目中的服务配置信息
func (a *App) GetConfigServices() ([]ce.ServiceInfo, error) {
	if a.projectInfo == nil {
		return nil, fmt.Errorf("未打开项目")
	}
	return ce.GetServiceList(a.projectInfo.Root)
}

// ExportConfigToRemote 导出所有服务配置到远程配置中心
func (a *App) ExportConfigToRemote(cfg ce.RemoteConfig) *ce.ExportResult {
	if a.projectInfo == nil {
		runtime.LogErrorf(a.ctx, "未打开项目")
		return &ce.ExportResult{Success: false, Error: "未打开项目"}
	}

	if errMsg := cfg.Validate(); errMsg != "" {
		runtime.LogErrorf(a.ctx, "配置验证失败: %s", errMsg)
		return &ce.ExportResult{Success: false, Error: errMsg}
	}

	err := ce.ExportAll(
		string(cfg.Type),
		cfg.Endpoint,
		cfg.ProjectName,
		a.projectInfo.Root,
		cfg.Group,
		cfg.Env,
		cfg.NamespaceId,
	)
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
	if a.projectInfo == nil {
		return nil, fmt.Errorf("未打开项目")
	}
	return devtools.GetServices(a.projectInfo.Root)
}

// CreateProject 创建新项目
func (a *App) CreateProject(opts devtools.CreateProjectOptions) *devtools.CommandResult {
	return devtools.CreateProject(a.ctx, opts)
}

// AddService 向已有项目添加新服务
func (a *App) AddService(opts devtools.AddServiceOptions) *devtools.CommandResult {
	if a.projectInfo == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	return devtools.AddService(a.projectInfo.Root, opts)
}

// ==================== 开发工具相关方法 ====================

// DevRunService 运行指定服务（在终端窗口中运行）
func (a *App) DevRunService(serviceName string) *devtools.CommandResult {
	if a.projectInfo == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	return devtools.RunServiceInTerminal(a.projectInfo.Root, serviceName)
}

// DevBufGenerate 运行 buf generate
func (a *App) DevBufGenerate() *devtools.CommandResult {
	if a.projectInfo == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	return devtools.RunBufGenerate(a.projectInfo.Root)
}

// DevEntGenerate 运行 ent generate
func (a *App) DevEntGenerate(serviceName string) *devtools.CommandResult {
	if a.projectInfo == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	if serviceName == "" {
		return devtools.RunEntGenerateAll(a.projectInfo.Root)
	}
	return devtools.RunEntGenerate(a.projectInfo.Root, serviceName)
}

// DevWireGenerate 运行 wire
func (a *App) DevWireGenerate(serviceName string) *devtools.CommandResult {
	if a.projectInfo == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	if serviceName == "" {
		return devtools.RunWireAll(a.projectInfo.Root)
	}
	return devtools.RunWire(a.projectInfo.Root, serviceName)
}

// DevGoModTidy 运行 go mod tidy
func (a *App) DevGoModTidy() *devtools.CommandResult {
	if a.projectInfo == nil {
		return &devtools.CommandResult{Success: false, Error: "未打开项目"}
	}
	return devtools.RunGoModTidy(a.projectInfo.Root)
}

// ExportOneServiceConfig 导出单个服务的配置到远程配置中心
func (a *App) ExportOneServiceConfig(cfg ce.RemoteConfig, serviceName string) *ce.ExportResult {
	if a.projectInfo == nil {
		runtime.LogErrorf(a.ctx, "未打开项目")
		return &ce.ExportResult{Success: false, Error: "未打开项目"}
	}

	if errMsg := cfg.Validate(); errMsg != "" {
		runtime.LogErrorf(a.ctx, "配置验证失败: %s", errMsg)
		return &ce.ExportResult{Success: false, Error: errMsg}
	}

	err := ce.ExportOne(
		string(cfg.Type),
		cfg.Endpoint,
		cfg.ProjectName,
		a.projectInfo.Root,
		cfg.Group,
		cfg.Env,
		cfg.NamespaceId,
		serviceName,
	)
	if err != nil {
		runtime.LogErrorf(a.ctx, "导出服务 %s 配置失败: %v", serviceName, err)
		return &ce.ExportResult{Success: false, Error: err.Error(), Service: serviceName}
	}

	runtime.EventsEmit(a.ctx, "config-exported")
	return &ce.ExportResult{Success: true, Service: serviceName}
}
