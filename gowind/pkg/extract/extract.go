package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/service"
)

// Options 提取选项
type Options struct {
	// 项目根路径
	RootPath string
	// 项目模块路径 (go.mod 中的 module)
	ModulePath string
	// 项目名称 (从 module path 提取的最后一段)
	ProjectName string
	// 源服务名
	SourceService string
	// 目标服务名
	TargetService string
	// 要提取的模型名列表 (单数形式, 如 "role", "user")
	Models []string
	// ORM 类型: "ent" | "gorm"
	OrmType string
	// 服务器类型 (用于创建新服务时): ["grpc"], ["grpc","rest"], ...
	Servers []string
	// 数据库客户端类型 (用于创建新服务时): ["ent"], ["gorm"], ...
	DbClients []string
	// 是否保留源文件（默认 false = 删除）
	KeepSource bool
}

// Extractor 从源服务提取模块到目标服务
type Extractor struct {
	opts  Options
	goGen *generators.GoGenerator
}

// NewExtractor 创建提取器
func NewExtractor(opts Options) *Extractor {
	return &Extractor{
		opts:  opts,
		goGen: generators.NewGoGenerator(),
	}
}

// Run 执行提取
func (e *Extractor) Run() error {
	// 检测目标服务是否存在，不存在则自动创建
	if err := e.ensureTargetService(); err != nil {
		return fmt.Errorf("ensure target service: %w", err)
	}

	for _, model := range e.opts.Models {
		if err := e.extractModel(model); err != nil {
			return fmt.Errorf("extract model %q: %w", model, err)
		}
		fmt.Printf("  Extracted model: %s\n", model)
	}

	// 提取完成后，更新目标端 server 注册
	if err := e.updateTargetServer(); err != nil {
		return fmt.Errorf("update target server: %w", err)
	}

	if !e.opts.KeepSource {
		if err := e.cleanupSource(); err != nil {
			return fmt.Errorf("cleanup source: %w", err)
		}
	}

	return nil
}

// ensureTargetService 检测目标服务是否存在，不存在则自动创建服务脚手架
func (e *Extractor) ensureTargetService() error {
	servicePath := filepath.Join(e.opts.RootPath, "app", e.opts.TargetService, "service")
	if isDirExists(servicePath) {
		return nil // 目标服务已存在
	}

	fmt.Printf("  Target service [%s] does not exist, creating...\n", e.opts.TargetService)

	// 默认值
	servers := e.opts.Servers
	if len(servers) == 0 {
		servers = []string{"grpc"}
	}
	dbClients := e.opts.DbClients
	if len(dbClients) == 0 {
		dbClients = []string{e.opts.OrmType}
	}

	opts := service.GeneratorOptions{
		GenerateMain:     true,
		GenerateServer:   true,
		GenerateService:  true,
		GenerateData:     true,
		GenerateMakefile: true,
		GenerateConfigs:  true,

		ProjectName:   e.opts.ProjectName,
		ProjectModule: e.opts.ModulePath,
		ServiceName:   e.opts.TargetService,

		Servers:   servers,
		DbClients: dbClients,

		OutputPath: e.opts.RootPath,
	}

	if err := service.Generate(context.Background(), opts); err != nil {
		return fmt.Errorf("create target service %q: %w", e.opts.TargetService, err)
	}

	fmt.Printf("  Target service [%s] created.\n", e.opts.TargetService)
	return nil
}

// ==============================
// 单模型提取
// ==============================

func (e *Extractor) extractModel(model string) error {
	// 1. 提取 schema (ent/gorm)
	if err := e.extractSchema(model); err != nil {
		return err
	}

	// 2. 提取 repo (data 层)
	if err := e.extractRepo(model); err != nil {
		return err
	}

	// 3. 提取 service
	if err := e.extractService(model); err != nil {
		return err
	}

	// 4. 在目标端 wire providers 中注入 New 函数
	if err := e.addTargetWireProviders(model); err != nil {
		return err
	}

	return nil
}

// ==============================
// schema 提取
// ==============================

func (e *Extractor) extractSchema(model string) error {
	switch e.opts.OrmType {
	case "ent":
		return e.extractEntSchema(model)
	case "gorm":
		return e.extractGormSchema(model)
	}
	return nil
}

func (e *Extractor) extractEntSchema(model string) error {
	fileName := stringcase.SnakeCase(model) + ".go"
	srcFile := filepath.Join(e.sourceServicePath(), "internal", "data", "ent", "schema", fileName)
	dstFile := filepath.Join(e.targetServicePath(), "internal", "data", "ent", "schema", fileName)

	return e.copyAndReplaceImport(srcFile, dstFile)
}

func (e *Extractor) extractGormSchema(model string) error {
	fileName := stringcase.SnakeCase(model) + ".go"
	srcFile := filepath.Join(e.sourceServicePath(), "internal", "data", "gorm", "schema", fileName)
	dstFile := filepath.Join(e.targetServicePath(), "internal", "data", "gorm", "schema", fileName)

	if err := e.copyAndReplaceImport(srcFile, dstFile); err != nil {
		return err
	}

	daoFileName := stringcase.SnakeCase(model) + "_dao.go"
	srcDao := filepath.Join(e.sourceServicePath(), "internal", "data", "gorm", "dao", daoFileName)
	dstDao := filepath.Join(e.targetServicePath(), "internal", "data", "gorm", "dao", daoFileName)

	return e.copyAndReplaceImport(srcDao, dstDao)
}

// ==============================
// repo 提取
// ==============================

func (e *Extractor) extractRepo(model string) error {
	fileName := stringcase.SnakeCase(model) + "_repo.go"
	srcFile := filepath.Join(e.sourceServicePath(), "internal", "data", fileName)
	dstFile := filepath.Join(e.targetServicePath(), "internal", "data", fileName)

	return e.copyAndReplaceImport(srcFile, dstFile)
}

// ==============================
// service 提取
// ==============================

func (e *Extractor) extractService(model string) error {
	fileName := stringcase.SnakeCase(model) + "_service.go"
	srcFile := filepath.Join(e.sourceServicePath(), "internal", "service", fileName)
	dstFile := filepath.Join(e.targetServicePath(), "internal", "service", fileName)

	return e.copyAndReplaceImport(srcFile, dstFile)
}

// ==============================
// wire providers 注入
// ==============================

func (e *Extractor) addTargetWireProviders(model string) error {
	modelPascal := stringcase.ToPascalCase(model)

	dataProviderFile := filepath.Join(e.targetServicePath(), "internal", "data", "providers", "wire_set.go")
	repoFunc := "data.New" + modelPascal + "Repo"
	if err := e.upsertProvider(dataProviderFile, repoFunc); err != nil {
		return fmt.Errorf("add repo to data providers: %w", err)
	}

	svcProviderFile := filepath.Join(e.targetServicePath(), "internal", "service", "providers", "wire_set.go")
	svcFunc := "service.New" + modelPascal + "Service"
	if err := e.upsertProvider(svcProviderFile, svcFunc); err != nil {
		return fmt.Errorf("add service to service providers: %w", err)
	}

	return nil
}

// ==============================
// server 注册
// ==============================

func (e *Extractor) updateTargetServer() error {
	grpcServerFile := filepath.Join(e.targetServicePath(), "internal", "server", "grpc_server.go")
	if isFileExists(grpcServerFile) {
		if err := e.addServiceToGrpcServer(grpcServerFile); err != nil {
			return fmt.Errorf("update grpc server: %w", err)
		}
	}

	restServerFile := filepath.Join(e.targetServicePath(), "internal", "server", "rest_server.go")
	if isFileExists(restServerFile) {
		if err := e.addServiceToRestServer(restServerFile); err != nil {
			return fmt.Errorf("update rest server: %w", err)
		}
	}

	return nil
}

// ==============================
// 源端清理
// ==============================

func (e *Extractor) cleanupSource() error {
	for _, model := range e.opts.Models {
		if err := e.cleanupSourceModel(model); err != nil {
			return err
		}
	}

	if err := e.removeSourceServerRegistrations(); err != nil {
		return err
	}

	return nil
}

func (e *Extractor) cleanupSourceModel(model string) error {
	modelSnake := stringcase.SnakeCase(model)
	modelPascal := stringcase.ToPascalCase(model)

	switch e.opts.OrmType {
	case "ent":
		schemaFile := filepath.Join(e.sourceServicePath(), "internal", "data", "ent", "schema", modelSnake+".go")
		_ = os.Remove(schemaFile)
	case "gorm":
		schemaFile := filepath.Join(e.sourceServicePath(), "internal", "data", "gorm", "schema", modelSnake+".go")
		_ = os.Remove(schemaFile)
		daoFile := filepath.Join(e.sourceServicePath(), "internal", "data", "gorm", "dao", modelSnake+"_dao.go")
		_ = os.Remove(daoFile)
	}

	repoFile := filepath.Join(e.sourceServicePath(), "internal", "data", modelSnake+"_repo.go")
	_ = os.Remove(repoFile)

	svcFile := filepath.Join(e.sourceServicePath(), "internal", "service", modelSnake+"_service.go")
	_ = os.Remove(svcFile)

	dataProviderFile := filepath.Join(e.sourceServicePath(), "internal", "data", "providers", "wire_set.go")
	_ = e.removeProvider(dataProviderFile, "data.New"+modelPascal+"Repo")

	svcProviderFile := filepath.Join(e.sourceServicePath(), "internal", "service", "providers", "wire_set.go")
	_ = e.removeProvider(svcProviderFile, "service.New"+modelPascal+"Service")

	return nil
}

// ==============================
// 辅助方法
// ==============================

func (e *Extractor) sourceServicePath() string {
	return filepath.Join(e.opts.RootPath, "app", e.opts.SourceService, "service")
}

func (e *Extractor) targetServicePath() string {
	return filepath.Join(e.opts.RootPath, "app", e.opts.TargetService, "service")
}

func (e *Extractor) copyAndReplaceImport(srcFile, dstFile string) error {
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("read source file %s: %w", srcFile, err)
	}

	content := string(data)

	oldImport := fmt.Sprintf("%s/app/%s/service", e.opts.ModulePath, e.opts.SourceService)
	newImport := fmt.Sprintf("%s/app/%s/service", e.opts.ModulePath, e.opts.TargetService)
	content = strings.ReplaceAll(content, oldImport, newImport)

	oldApi := fmt.Sprintf("%s/api/gen/go/%s/", e.opts.ModulePath, e.opts.SourceService)
	newApi := fmt.Sprintf("%s/api/gen/go/%s/", e.opts.ModulePath, e.opts.TargetService)
	content = strings.ReplaceAll(content, oldApi, newApi)

	dstDir := filepath.Dir(dstFile)
	if err = os.MkdirAll(dstDir, os.ModePerm); err != nil {
		return fmt.Errorf("create target dir %s: %w", dstDir, err)
	}

	if err = os.WriteFile(dstFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write target file %s: %w", dstFile, err)
	}

	return nil
}

func (e *Extractor) upsertProvider(filePath string, functionCall string) error {
	if !isFileExists(filePath) {
		return fmt.Errorf("provider file not found: %s", filePath)
	}
	return e.goGen.UpsertProviderSetFunction(filePath, functionCall)
}

func (e *Extractor) removeProvider(filePath string, functionCall string) error {
	if !isFileExists(filePath) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == functionCall || trimmed == functionCall+"," {
			continue
		}
		newLines = append(newLines, line)
	}

	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(filePath, []byte(newContent), 0644)
}

// ==============================
// grpc server 注入
// ==============================

func (e *Extractor) addServiceToGrpcServer(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)

	for _, model := range e.opts.Models {
		modelPascal := stringcase.ToPascalCase(model)
		modelCamel := stringcase.LowerCamelCase(model)
		svcVar := modelCamel + "Service"
		svcType := "*service." + modelPascal + "Service"
		moduleName := e.opts.TargetService

		paramLine := fmt.Sprintf("\t%s %s,", svcVar, svcType)
		if strings.Contains(content, svcVar+" "+svcType) {
			continue
		}

		content, err = injectBeforeMarker(content, ") (*grpc.Server", paramLine)
		if err != nil {
			return err
		}

		registerLine := fmt.Sprintf("\t%sV1.Register%sServer(srv, %s)", moduleName, modelPascal, svcVar)
		if !strings.Contains(content, registerLine) {
			content = strings.Replace(content,
				"\treturn srv, nil",
				registerLine+"\n\n\treturn srv, nil",
				1)
		}

		protoImport := fmt.Sprintf("\t%sV1 \"%s/api/gen/go/%s/service/v1\"", moduleName, e.opts.ModulePath, moduleName)
		if !strings.Contains(content, protoImport) {
			content = strings.Replace(content,
				"\""+e.opts.ModulePath+"/app/"+e.opts.TargetService+"/service/internal/service\"",
				"\""+e.opts.ModulePath+"/app/"+e.opts.TargetService+"/service/internal/service\"\n"+protoImport,
				1)
		}
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// ==============================
// rest server 注入
// ==============================

func (e *Extractor) addServiceToRestServer(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)

	for _, model := range e.opts.Models {
		modelPascal := stringcase.ToPascalCase(model)
		modelCamel := stringcase.LowerCamelCase(model)
		svcVar := modelCamel + "Service"
		svcType := "*service." + modelPascal + "Service"
		moduleName := e.opts.TargetService

		paramLine := fmt.Sprintf("\t%s %s,", svcVar, svcType)
		if strings.Contains(content, svcVar+" "+svcType) {
			continue
		}

		content, err = injectBeforeMarker(content, ") (*http.Server", paramLine)
		if err != nil {
			return err
		}

		registerLine := fmt.Sprintf("\t%sV1.Register%sHTTPServer(srv, %s)", moduleName, modelPascal, svcVar)
		if !strings.Contains(content, registerLine) {
			content = strings.Replace(content,
				"\treturn srv, nil",
				registerLine+"\n\n\treturn srv, nil",
				1)
		}

		protoImport := fmt.Sprintf("\t%sV1 \"%s/api/gen/go/%s/service/v1\"", moduleName, e.opts.ModulePath, moduleName)
		if !strings.Contains(content, protoImport) {
			content = strings.Replace(content,
				"\""+e.opts.ModulePath+"/app/"+e.opts.TargetService+"/service/internal/service\"",
				"\""+e.opts.ModulePath+"/app/"+e.opts.TargetService+"/service/internal/service\"\n"+protoImport,
				1)
		}
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// ==============================
// 源端 server 清理
// ==============================

func (e *Extractor) removeSourceServerRegistrations() error {
	grpcServerFile := filepath.Join(e.sourceServicePath(), "internal", "server", "grpc_server.go")
	if isFileExists(grpcServerFile) {
		if err := e.removeServiceFromServer(grpcServerFile, "Server"); err != nil {
			return err
		}
	}

	restServerFile := filepath.Join(e.sourceServicePath(), "internal", "server", "rest_server.go")
	if isFileExists(restServerFile) {
		if err := e.removeServiceFromServer(restServerFile, "HTTPServer"); err != nil {
			return err
		}
	}

	return nil
}

func (e *Extractor) removeServiceFromServer(filePath string, registerSuffix string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	moduleName := e.opts.SourceService

	for _, model := range e.opts.Models {
		modelPascal := stringcase.ToPascalCase(model)
		modelCamel := stringcase.LowerCamelCase(model)
		svcVar := modelCamel + "Service"

		paramPattern := svcVar + " *service." + modelPascal + "Service"
		registerPattern := moduleName + "V1.Register" + modelPascal + registerSuffix

		filtered := make([]string, 0, len(lines))
		for _, line := range lines {
			if strings.Contains(line, paramPattern) {
				continue
			}
			if strings.Contains(line, registerPattern) {
				continue
			}
			filtered = append(filtered, line)
		}
		lines = filtered
	}

	newContent := strings.Join(lines, "\n")
	return os.WriteFile(filePath, []byte(newContent), 0644)
}

// ==============================
// 公共工具函数
// ==============================

// DetectOrmType 自动检测源服务使用的 ORM 类型
func DetectOrmType(servicePath string) string {
	entSchemaPath := filepath.Join(servicePath, "internal", "data", "ent", "schema")
	if isDirExists(entSchemaPath) {
		return "ent"
	}

	gormSchemaPath := filepath.Join(servicePath, "internal", "data", "gorm", "schema")
	if isDirExists(gormSchemaPath) {
		return "gorm"
	}

	return ""
}

// injectBeforeMarker 在 marker 字符串前插入一行文本
func injectBeforeMarker(content string, marker string, line string) (string, error) {
	idx := strings.Index(content, marker)
	if idx < 0 {
		return "", fmt.Errorf("marker %q not found in file", marker)
	}
	return content[:idx] + line + "\n" + content[idx:], nil
}

// isFileExists 检查文件是否存在且不是目录
func isFileExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fi.IsDir()
}

// isDirExists 检查目录是否存在
func isDirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}
