package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
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

		// 目标服务继承源服务的装配形态:wire 工程内迁移保持 wire,其余采用手写装配。
		UseWireDI: generators.WireProvidersExist(e.sourceServicePath()) &&
			generators.FindWiringFile(filepath.Join(e.sourceServicePath(), "cmd", "server"), "") == "",

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

	// 4. 在目标端登记模块构造行(wire provider 集 或 装配文件锚点,按目标服务形态)
	tctx := e.targetWiringContextFor(model)
	if err := e.addTargetRegistrations(model, tctx); err != nil {
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
// 目标端装配形态
// ==============================

// targetContext 描述目标服务的依赖装配形态。仓库型模块注入仓储+服务构造行;
// BFF 型(拷贝来的服务文件不引用 internal/data)注入服务构造行,其数据源变量为服务客户端。
type targetContext struct {
	wiringFile   string
	wireMode     bool
	ormClientVar string
	useClient    bool
}

// importsInternalData 判定服务文件是否引用了本服务的 internal/data 包(含子包 internal/data/ent、
// internal/data/dao)。引用形如 "mod/app/x/service/internal/data"——引号只可能落在路径两端,
// 因此判据必须是尾引号或中段斜杠边界;拿 `"/internal/data"` 当子串永远命中不了。
func importsInternalData(src string) bool {
	return strings.Contains(src, `/internal/data"`) || strings.Contains(src, `/internal/data/`)
}

func (e *Extractor) targetWiringContextFor(model string) targetContext {
	var tctx targetContext

	serviceFile := filepath.Join(e.targetServicePath(), "internal", "service", stringcase.SnakeCase(model)+"_service.go")
	tctx.useClient = true
	if raw, err := os.ReadFile(serviceFile); err == nil {
		tctx.useClient = !importsInternalData(string(raw))
	}

	pref := ""
	if !tctx.useClient {
		pref = e.opts.OrmType
	}
	tctx.wiringFile = generators.FindWiringFile(filepath.Join(e.targetServicePath(), "cmd", "server"), pref)
	tctx.wireMode = tctx.wiringFile == "" && generators.WireProvidersExist(e.targetServicePath())
	if tctx.wiringFile != "" && !tctx.useClient && e.opts.OrmType != "" {
		tctx.ormClientVar = generators.DetectOrmClientVar(tctx.wiringFile, e.opts.OrmType)
	}
	return tctx
}

// ==============================
// 目标端模块登记
// ==============================

func (e *Extractor) addTargetRegistrations(model string, tctx targetContext) error {
	modelPascal := stringcase.ToPascalCase(model)

	// 旧式 wire 服务:写入 provider 集。
	if tctx.wireMode {
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

	// 手写装配服务:锚点注入构造行。
	if tctx.wiringFile == "" {
		return fmt.Errorf("target service %s has neither wiring anchors nor wire providers", e.opts.TargetService)
	}

	if !tctx.useClient && e.opts.OrmType != "" {
		repoLine, repoSkip := generators.BuildRepoWiringLine(model, tctx.ormClientVar)
		if err := generators.ApplyAnchorPatches(generators.AnchorPatch{
			Path:    tctx.wiringFile,
			Anchors: generators.WiringAnchorCandidates("repo"),
			Lines:   []string{repoLine},
			SkipIf:  repoSkip,
		}); err != nil {
			return err
		}
		if err := e.goGen.EnsureImport(tctx.wiringFile,
			fmt.Sprintf("%s/app/%s/service/internal/data", e.opts.ModulePath, strings.ToLower(e.opts.TargetService))); err != nil {
			return err
		}
	}

	serviceLine, serviceSkip := generators.BuildServiceWiringLine(model, tctx.useClient)
	if err := generators.ApplyAnchorPatches(generators.AnchorPatch{
		Path:    tctx.wiringFile,
		Anchors: generators.WiringAnchorCandidates("service"),
		Lines:   []string{serviceLine},
		SkipIf:  serviceSkip,
	}); err != nil {
		return err
	}
	return e.goGen.EnsureImport(tctx.wiringFile,
		fmt.Sprintf("%s/app/%s/service/internal/service", e.opts.ModulePath, strings.ToLower(e.opts.TargetService)))
}

// ==============================
// server 注册
// ==============================

func (e *Extractor) updateTargetServer() error {
	// 目标端装配形态按首个模型判定(同一次提取的所有模型同型)。
	var tctx targetContext
	if len(e.opts.Models) > 0 {
		tctx = e.targetWiringContextFor(e.opts.Models[0])
	}

	grpcServerFile := filepath.Join(e.targetServicePath(), "internal", "server", "grpc_server.go")
	if pkg.IsFileExists(grpcServerFile) {
		if err := e.addServiceToGrpcServer(grpcServerFile, tctx); err != nil {
			return fmt.Errorf("update grpc server: %w", err)
		}
	}

	restServerFile := filepath.Join(e.targetServicePath(), "internal", "server", "rest_server.go")
	if pkg.IsFileExists(restServerFile) {
		if err := e.addServiceToRestServer(restServerFile, tctx); err != nil {
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

	// 手写装配形态的源端清理:移除装配文件中该模块的全部登记行。
	if wiringFiles, globErr := filepath.Glob(filepath.Join(e.sourceServicePath(), "cmd", "server", "wiring*.go")); globErr == nil {
		for _, wf := range wiringFiles {
			if rmErr := generators.RemoveWiringModuleLines(wf, model); rmErr != nil {
				return rmErr
			}
		}
	}

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
	if err = os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("create target dir %s: %w", dstDir, err)
	}

	if err = os.WriteFile(dstFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write target file %s: %w", dstFile, err)
	}

	return nil
}

func (e *Extractor) upsertProvider(filePath string, functionCall string) error {
	if !pkg.IsFileExists(filePath) {
		return fmt.Errorf("provider file not found: %s", filePath)
	}
	return e.goGen.UpsertProviderSetFunction(filePath, functionCall)
}

func (e *Extractor) removeProvider(filePath string, functionCall string) error {
	if !pkg.IsFileExists(filePath) {
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
// 锚点注入(server 文件形参/路由 + 装配文件实参)
// ==============================

// injectServerFileAnchors 向带登记锚点的 server 文件注入各模块的形参与路由注册行,
// 并向装配文件注入对应的服务实参行。注入按 skipIf 幂等。
// 迁移语义与既有标记注入一致:路由域与别名取目标服务自身的域。
func (e *Extractor) injectServerFileAnchors(serverFile string, serverKind string, tctx targetContext) error {
	domain := strings.ToLower(e.opts.TargetService)
	for _, model := range e.opts.Models {
		paramLine, paramSkip := generators.BuildServerParamLine(model)
		routeLine, routeSkip := generators.BuildRouteLine(serverKind, domain, "v1", model)
		if err := generators.ApplyAnchorPatches(
			generators.AnchorPatch{
				Path:    serverFile,
				Anchors: []string{generators.AnchorParam},
				Lines:   []string{paramLine},
				SkipIf:  paramSkip,
			},
			generators.AnchorPatch{
				Path:    serverFile,
				Anchors: []string{generators.AnchorRoute},
				Lines:   []string{routeLine},
				SkipIf:  routeSkip,
			},
		); err != nil {
			return err
		}

		if err := e.goGen.EnsureAliasedImport(serverFile,
			generators.ApiPackageAlias(domain, "v1"),
			generators.ApiImportPath(serverKind, e.opts.ModulePath, domain, "v1")); err != nil {
			return err
		}
		if err := e.goGen.EnsureImport(serverFile,
			fmt.Sprintf("%s/app/%s/service/internal/service", e.opts.ModulePath, domain)); err != nil {
			return err
		}

		if tctx.wiringFile != "" {
			argLine, argSkip := generators.BuildServerArgLine(model)
			if err := generators.ApplyAnchorPatches(generators.AnchorPatch{
				Path:    tctx.wiringFile,
				Anchors: generators.WiringAnchorCandidates("arg-" + serverKind),
				Lines:   []string{argLine},
				SkipIf:  argSkip,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// ==============================
// grpc server 注入
// ==============================

func (e *Extractor) addServiceToGrpcServer(filePath string, tctx targetContext) error {
	// 手写装配形态:锚点注入形参与路由。
	if tctx.wiringFile != "" && generators.FileHasTrimmedLine(filePath, generators.AnchorParam) {
		return e.injectServerFileAnchors(filePath, "grpc", tctx)
	}

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

func (e *Extractor) addServiceToRestServer(filePath string, tctx targetContext) error {
	// 手写装配形态:锚点注入形参与路由。
	if tctx.wiringFile != "" && generators.FileHasTrimmedLine(filePath, generators.AnchorParam) {
		return e.injectServerFileAnchors(filePath, "rest", tctx)
	}

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
	if pkg.IsFileExists(grpcServerFile) {
		if err := e.removeServiceFromServer(grpcServerFile, "Server"); err != nil {
			return err
		}
	}

	restServerFile := filepath.Join(e.sourceServicePath(), "internal", "server", "rest_server.go")
	if pkg.IsFileExists(restServerFile) {
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

// isDirExists 检查路径是否为已存在的目录（两态语义：stat 出错一律
// false，且必须确为目录才 true）。用于 ORM schema 目录探测，与
// internal/pkg.IsDirExists 的三态语义不同，不可互换。
func isDirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}
