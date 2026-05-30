package main

import (
	"context"

	ddlparser "github.com/tx7do/go-utils/ddl_parser"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/detect"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
)

// App struct
type App struct {
	ctx context.Context

	projectInfo *detect.ProjectInfo
	dbConfig    *database.DBConfig

	projectDetector *detect.ProjectDetector
	generator       *generator.Generator
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		projectDetector: detect.NewProjectDetector(),
		generator:       generator.NewGenerator(),
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
