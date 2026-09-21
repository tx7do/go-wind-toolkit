package sqlkratos

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jinzhu/inflection"
	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"

	sqlorm "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlorm"
	sqlproto "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto"
)

// ensureDSNScheme ensures the DSN has a valid scheme prefix based on the driver type.
// If it already carries a scheme (or is DDL text, or driver is empty), it is returned as-is:
// such sources describe their own type, so the driver must not gate them.
// For PostgreSQL key-value format DSN (e.g. "host=localhost port=5432 user=postgres ..."),
// it converts to URL format (e.g. "postgres://user:pass@host:port/dbname?sslmode=disable").
// A driver we have no provider for is rejected here: passing it through would let the
// source be reinterpreted as text:// and "succeed" while generating nothing.
func ensureDSNScheme(dsn, driver string) (string, error) {
	if strings.Contains(dsn, "://") {
		return dsn, nil
	}
	if isDDLText(dsn) {
		return dsn, nil
	}
	switch strings.ToLower(driver) {
	case "":
		return dsn, nil
	case "mysql":
		return "mysql://" + dsn, nil
	case "postgresql", "postgres":
		// PostgreSQL key-value DSN: "host=localhost port=5432 user=postgres password=xxx dbname=mydb sslmode=disable"
		if isPostgresKeyValueDSN(dsn) {
			return convertPostgresKeyValueToURL(dsn), nil
		}
		return "postgres://" + dsn, nil
	default:
		return "", fmt.Errorf("sqlkratos: unsupported driver: %q (supported: mysql, postgres, postgresql; leave it empty when the source carries its own scheme)", driver)
	}
}

// isPostgresKeyValueDSN detects PostgreSQL key-value format DSN.
// Key-value DSN contains space-separated key=value pairs like "host=localhost port=5432 user=postgres".
func isPostgresKeyValueDSN(dsn string) bool {
	return strings.Contains(dsn, "=") && strings.Contains(dsn, " ")
}

// isDDLText detects DDL text content (CREATE TABLE statements) passed as Source
// instead of a DSN; such content must be handed to sqlproto's text:// handling as-is.
func isDDLText(dsn string) bool {
	return strings.Contains(strings.ToUpper(dsn), "CREATE TABLE")
}

// convertPostgresKeyValueToURL converts PostgreSQL key-value DSN to URL format.
// Input:  "host=localhost port=5432 user=postgres password=xxx dbname=mydb sslmode=disable"
// Output: "postgres://postgres:xxx@localhost:5432/mydb?sslmode=disable"
//
// 编码交给 url.URL:手写 url.QueryEscape 走的是表单规则,空格编成 "+",而 userinfo
// 与 path 里的 "+" 是字面量,含空格或 "+" 的口令会被原样送到服务端。
func convertPostgresKeyValueToURL(dsn string) string {
	parts := strings.Fields(dsn)
	vals := make(map[string]string)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			vals[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	host := vals["host"]
	if host == "" {
		host = "localhost"
	}
	port := vals["port"]
	if port == "" {
		port = "5432"
	}
	user := vals["user"]
	password := vals["password"]
	dbname := vals["dbname"]

	u := &url.URL{
		Scheme: "postgres",
		Host:   host + ":" + port,
	}
	switch {
	case user != "" && password != "":
		u.User = url.UserPassword(user, password)
	case user != "":
		// 无口令时不能留 "user:" 的空冒号:那会被解成一个空口令。
		u.User = url.User(user)
	}
	if dbname != "" {
		u.Path = "/" + dbname
	}

	// Values.Encode 按键排序,同一份 DSN 每次得到同一个字符串。
	params := url.Values{}
	for k, v := range vals {
		switch k {
		case "host", "port", "user", "password", "dbname":
			// already handled
		default:
			params.Set(k, v)
		}
	}
	u.RawQuery = params.Encode()

	return u.String()
}

func Generate(ctx context.Context, opts GeneratorOptions) error {
	g := NewGenerator()
	return g.Generate(ctx, opts)
}

// PlanTables 只读解析数据源中将被处理的表(已应用 include/exclude 过滤),
// 不导出 proto、不生成任何文件。供 generate --dry-run 预览与校验数据源。
func PlanTables(ctx context.Context, opts GeneratorOptions) (sqlproto.TableDataArray, error) {
	probe := opts
	probe.GenerateProto = false

	// Convert 内部会无条件创建 proto 输出目录;预览时若该目录(及其 api 父目录)
	// 原本不存在,结束后将其清理,保证预览零副作用。
	protoPath := path.Join(opts.OutputPath, "/api/protos/")
	apiDir := path.Dir(protoPath)
	protoExisted := dirExists(protoPath)
	apiExisted := dirExists(apiDir)

	tables, err := NewGenerator().generateProtobufCode(ctx, probe)

	if !protoExisted {
		_ = os.Remove(protoPath)
	}
	if !apiExisted {
		_ = os.Remove(apiDir)
	}
	return tables, err
}

func dirExists(dir string) bool {
	fi, err := os.Stat(dir)
	return err == nil && fi.IsDir()
}

type Generator struct {
	goGenerator       *generators.GoGenerator
	yamlGenerator     *generators.YamlGenerator
	makefileGenerator *generators.MakefileGenerator
	protoGenerator    *generators.ProtoGenerator
}

func NewGenerator() *Generator {
	return &Generator{
		goGenerator:       generators.NewGoGenerator(),
		yamlGenerator:     generators.NewYamlGenerator(),
		makefileGenerator: generators.NewMakefileGenerator(),
		protoGenerator:    generators.NewProtoGenerator(),
	}
}

func (g *Generator) Generate(ctx context.Context, opts GeneratorOptions) error {
	var err error

	var tables sqlproto.TableDataArray

	// 生成 Protobuf schema
	if tables, err = g.generateProtobufCode(ctx, opts); err != nil {
		return err
	}

	services := make([]string, 0)
	servicePackageMap := make(map[string]string)
	for _, table := range tables {
		if len(table.Fields) == 0 {
			continue
		}

		name := inflection.Singular(table.Name)

		services = append(services, name)

		// 根据 ProtoPackageStrategy 决定 proto module 名称
		var moduleName string
		switch opts.ProtoPackageStrategy {
		case "by-service":
			// 按服务分包：所有表共用服务名
			moduleName = strings.ToLower(opts.ModuleName)
		case "custom":
			// 自定义包名：从 TableCustomPackages 获取用户指定的包名
			if pkg, ok := opts.TableCustomPackages[table.Name]; ok && pkg != "" {
				moduleName = strings.ToLower(pkg)
			} else {
				moduleName = strings.ToLower(name)
			}
		default:
			// per-table（默认）：每表独立包
			moduleName = strings.ToLower(name)
		}
		servicePackageMap[name] = moduleName
	}

	var useGrpc bool
	for _, server := range opts.Servers {
		if server == "grpc" {
			useGrpc = true
			break
		}
	}

	// 目标服务的依赖装配形态判定:
	//   - 既有手写装配文件(含登记锚点) → 注入式登记(wiring 模式);
	//   - 既无装配文件又无 wire provider 集(全新服务) → 生成手写装配骨架(wiring 模式);
	//   - 仅有 wire provider 集(旧式服务) → 沿用 wire 生成(向下兼容)。
	// BFF(纯 rest 且不落仓储)的装配文件不含 ORM 客户端构造,探测时不再按 ORM 偏置。
	wctx := newWiringContext(opts.OutputPath, opts.ModuleName, opts.OrmType, useGrpc, opts.UseRepo)

	// 生成ORM代码
	if opts.GenerateORM {
		dataPackagePath := fmt.Sprintf("%s/app/%s/service/internal/", opts.OutputPath, opts.ModuleName)
		if err = g.generateOrmCode(ctx, opts, dataPackagePath); err != nil {
			return err
		}
	}

	// 生成data层代码
	if opts.GenerateData {
		dataPackagePath := fmt.Sprintf("%s/app/%s/service/internal/data", opts.OutputPath, opts.ModuleName)
		if err = g.generateDataPackageCode(
			dataPackagePath,
			opts.OrmType,
			opts.ProjectName,
			opts.ServiceName,
			tables,
			services,
			opts.ModuleVersion,
			servicePackageMap,
			wctx,
		); err != nil {
			return err
		}
	}

	// 生成service层代码
	if opts.GenerateService {
		servicePackagePath := fmt.Sprintf("%s/app/%s/service/internal/service/", opts.OutputPath, opts.ModuleName)
		servicePackagePath = path.Clean(servicePackagePath)
		log.Printf("Generating service package code at: %s", servicePackagePath)
		if err = g.generateServicePackageCode(
			servicePackagePath,
			opts.ProjectName,
			opts.ServiceName,
			opts.SourceModuleName, opts.ModuleVersion,
			opts.UseRepo, useGrpc,
			tables,
			services,
			servicePackageMap,
			wctx,
		); err != nil {
			return err
		}
	}

	// 生成server层代码
	if opts.GenerateServer {
		serverPackagePath := fmt.Sprintf("%s/app/%s/service/internal/server/", opts.OutputPath, opts.ModuleName)
		serverPackagePath = path.Clean(serverPackagePath)
		log.Printf("Generating server package code at: %s", serverPackagePath)
		if err = g.generateServerPackageCode(
			serverPackagePath,
			opts.ProjectName,
			opts.ServiceName,
			servicePackageMap,
			opts.Servers,
			opts.ModuleVersion,
			services,
			wctx,
		); err != nil {
			return err
		}
	}

	// rest 服务(BFF)随 server 层一并生成 swagger assets 包(rest_server 模板引用之)。
	if opts.GenerateServer {
		hasRestServer := false
		for _, s := range opts.Servers {
			if strings.EqualFold(strings.TrimSpace(s), "rest") {
				hasRestServer = true
				break
			}
		}
		if hasRestServer {
			assetsPath := filepath.Join(opts.OutputPath, "app", opts.ModuleName, "service", "cmd", "server", "assets")
			if _, err := g.goGenerator.GenerateAssets(ctx, code_generator.Options{
				OutDir: assetsPath,
				Module: opts.ProjectName,
			}); err != nil {
				return err
			}
		}
	}

	// 生成配置文件
	if opts.GenerateConfig {
		configPath := fmt.Sprintf("%s/app/%s/service/configs", opts.OutputPath, opts.ModuleName)
		if err = g.generateConfigCode(ctx, configPath, opts.Servers); err != nil {
			return err
		}
	}

	// 生成Makefile
	if opts.GenerateMakefile {
		makefilePath := fmt.Sprintf("%s/app/%s/service", opts.OutputPath, opts.ModuleName)
		if err = g.generateMakefileCode(ctx, makefilePath); err != nil {
			return err
		}
	}

	// 生成main包代码
	if opts.GenerateMain {
		mainPackagePath := fmt.Sprintf("%s/app/%s/service/cmd/server", opts.OutputPath, opts.ModuleName)
		if err = g.generateMainPackageCode(
			mainPackagePath,
			opts.ProjectName,
			opts.ServiceName,
			opts.Servers,
			services,
			wctx,
			opts,
		); err != nil {
			return err
		}
	}

	return nil
}

// generateProtobufCode generates the Protobuf code from the database schema.
func (g *Generator) generateProtobufCode(ctx context.Context, opts GeneratorOptions) (sqlproto.TableDataArray, error) {
	var err error
	var tables sqlproto.TableDataArray

	protoPath := path.Join(opts.OutputPath, "/api/protos/")

	// 确保 DSN 有正确的 scheme 前缀
	source, err := ensureDSNScheme(opts.Source, opts.Driver)
	if err != nil {
		return nil, err
	}

	for _, server := range opts.Servers {
		if server != "grpc" && server != "rest" {
			continue
		}

		if tables, err = sqlproto.Convert(
			ctx,
			&source,
			&protoPath,
			&opts.ModuleName,
			&opts.SourceModuleName,
			&opts.ModuleVersion,
			&server,
			opts.ProtoPackageStrategy,
			opts.TableCustomPackages,
			opts.IncludedTables,
			opts.ExcludedTables,
			opts.GenerateProto,
		); err != nil {
			return nil, err
		}
	}

	return tables, nil
}

// generateOrmCode generates the ORM code based on the specified ORM type.
//
// Go 源码数据源(ent://<dir>、gorm://<dir>)有特殊语义:
//   - ent:// 的 schema 本身就是输入,跳过导入(运行时代码由 gow ent 生成);
//   - gorm:// 经 sqlorm 路由到 DAO 回转生成,只补缺失模型、不覆盖用户模型。
func (g *Generator) generateOrmCode(
	ctx context.Context,
	opts GeneratorOptions,
	serviceRootPath string,
) error {
	var err error

	log.Println("Generating ORM code...")

	// 确保 DSN 有正确的 scheme 前缀
	source, err := ensureDSNScheme(opts.Source, opts.Driver)
	if err != nil {
		return err
	}

	var schemaPath string
	var daoPath string
	switch opts.OrmType {
	case "ent":
		schemaPath = path.Join(serviceRootPath, "/data/ent/schema")
	case "gorm":
		schemaPath = path.Join(serviceRootPath, "/data/gorm/models")
		daoPath = path.Join(serviceRootPath, "/data/gorm/dao")
	}

	switch {
	case strings.HasPrefix(source, "ent://"):
		if opts.OrmType != "ent" {
			return fmt.Errorf("sqlkratos: ent:// source requires --orm ent, got %q", opts.OrmType)
		}
		log.Println("Source is an ent schema dir; schema import skipped. Run `gow ent <service>` to (re)generate ent runtime code.")
		return nil

	case strings.HasPrefix(source, "gorm://"):
		if opts.OrmType != "gorm" {
			return fmt.Errorf("sqlkratos: gorm:// source requires --orm gorm, got %q", opts.OrmType)
		}
	}

	if err = sqlorm.Importer(
		ctx,
		opts.OrmType,
		&opts.Driver,
		&source,
		&schemaPath,
		&daoPath,
		opts.IncludedTables,
		opts.ExcludedTables,
	); err != nil {
		return err
	}

	log.Println("ORM code generation completed.")

	return nil
}

func (g *Generator) generateServerPackageCode(
	outputPath string,
	projectName string,
	serviceName string,
	servicePackageMap map[string]string,
	servers []string,
	moduleVersion string,
	services []string,
	wctx *wiringContext,
) error {
	for _, server := range servers {
		kind := strings.ToLower(server)
		serverFile := filepath.Join(outputPath, kind+"_server.go")

		// 既有带登记锚点的 server 文件:锚点注入新模块的形参与路由,不整体重渲,
		// 保留项目手工维护的中间件与路由内容。
		if wctx.useWiringDI {
			if _, statErr := os.Stat(serverFile); statErr == nil &&
				generators.FileHasTrimmedLine(serverFile, generators.AnchorParam) {
				if err := g.injectServerRegistrations(serverFile, kind, projectName, serviceName, servicePackageMap, moduleVersion, wctx, services); err != nil {
					return err
				}
				continue
			}
		}

		if err := g.WriteServerPackageCode(
			outputPath,
			projectName, server, serviceName,
			servicePackageMap,
		); err != nil {
			return err
		}
	}

	if !wctx.useWiringDI {
		return g.WriteWireSetCode(outputPath, projectName, serviceName, "server", "Server", servers)
	}
	return nil
}

// injectServerRegistrations 向既有的 server 文件锚点注入各模块的服务形参与路由注册,
// 并向装配文件注入对应的服务实参。注入按 skipIf 幂等:已登记的模块自动跳过。
func (g *Generator) injectServerRegistrations(
	serverFile string,
	serverKind string,
	projectName string,
	serviceName string,
	servicePackageMap map[string]string,
	moduleVersion string,
	wctx *wiringContext,
	models []string,
) error {
	for _, model := range models {
		// 路由所属 api 域:rest 恒为 BFF 自身域;grpc 为各模块的策略域。
		domain := strings.ToLower(serviceName)
		if serverKind == "grpc" {
			if m, ok := servicePackageMap[model]; ok && m != "" {
				domain = m
			} else {
				domain = strings.ToLower(model)
			}
		}

		paramLine, paramSkip := generators.BuildServerParamLine(model)
		routeLine, routeSkip := generators.BuildRouteLine(serverKind, domain, moduleVersion, model)

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

		// 路由引用的 api 包与 service 包 import 补齐(已存在则跳过)。
		if err := g.goGenerator.EnsureAliasedImport(serverFile,
			generators.ApiPackageAlias(domain, moduleVersion),
			generators.ApiImportPath(serverKind, projectName, domain, moduleVersion)); err != nil {
			return err
		}
		if err := g.goGenerator.EnsureImport(serverFile,
			fmt.Sprintf("%s/app/%s/service/internal/service", projectName, strings.ToLower(serviceName))); err != nil {
			return err
		}

		// 装配文件中的服务实参行。
		if wctx.wiringFile != "" {
			argLine, argSkip := generators.BuildServerArgLine(model)
			if err := generators.ApplyAnchorPatches(generators.AnchorPatch{
				Path:    wctx.wiringFile,
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

// injectWiringServiceLine 向装配文件注入一个模块的服务层构造行。
func (g *Generator) injectWiringServiceLine(
	wiringFile string,
	projectName string,
	serviceName string,
	model string,
	useClient bool,
) error {
	line, skipIf := generators.BuildServiceWiringLine(model, useClient)
	if err := generators.ApplyAnchorPatches(generators.AnchorPatch{
		Path:    wiringFile,
		Anchors: generators.WiringAnchorCandidates("service"),
		Lines:   []string{line},
		SkipIf:  skipIf,
	}); err != nil {
		return err
	}
	return g.goGenerator.EnsureImport(wiringFile,
		fmt.Sprintf("%s/app/%s/service/internal/service", projectName, strings.ToLower(serviceName)))
}

// injectWiringRepoLine 向装配文件注入一个模块的仓储层构造行。
func (g *Generator) injectWiringRepoLine(
	wiringFile string,
	projectName string,
	serviceName string,
	model string,
	ormClientVar string,
) error {
	line, skipIf := generators.BuildRepoWiringLine(model, ormClientVar)
	if err := generators.ApplyAnchorPatches(generators.AnchorPatch{
		Path:    wiringFile,
		Anchors: generators.WiringAnchorCandidates("repo"),
		Lines:   []string{line},
		SkipIf:  skipIf,
	}); err != nil {
		return err
	}
	return g.goGenerator.EnsureImport(wiringFile,
		fmt.Sprintf("%s/app/%s/service/internal/data", projectName, strings.ToLower(serviceName)))
}

func (g *Generator) generateServicePackageCode(
	outputPath string,
	projectName, serviceName string,
	sourceModuleName, moduleVersion string,
	userRepo, isGrpcService bool,
	tables sqlproto.TableDataArray,
	services []string,
	servicePackageMap map[string]string,
	wctx *wiringContext,
) error {

	for _, table := range tables {
		if len(table.Fields) == 0 {
			continue
		}

		name := inflection.Singular(table.Name)

		// 从 servicePackageMap 获取策略决定的 gRPC proto module 名
		sourceModule := servicePackageMap[name]
		if sourceModule == "" {
			sourceModule = strings.ToLower(name)
		}

		// gRPC 和 REST 的 Target/Source 语义不同：
		// gRPC: Target=源gRPC module(按策略), Source=同Target
		// REST: Target=BFF module(serviceName), Source=源gRPC module(按策略)
		var targetModule, effectiveSourceModule string
		if isGrpcService {
			targetModule = sourceModule
			effectiveSourceModule = sourceModule
		} else {
			targetModule = strings.ToLower(serviceName)
			effectiveSourceModule = sourceModule
		}

		if err := g.WriteServicePackageCode(
			outputPath,
			projectName, serviceName,
			name,
			targetModule, effectiveSourceModule, moduleVersion,
			userRepo, isGrpcService,
		); err != nil {
			return err
		}
	}

	if !wctx.useWiringDI {
		return g.WriteWireSetCode(outputPath, projectName, serviceName, "service", "Service", services)
	}
	if wctx.wiringFile != "" {
		for _, model := range services {
			if err := g.injectWiringServiceLine(wctx.wiringFile, projectName, serviceName, model, wctx.isBff); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Generator) generateDataPackageCode(
	outputPath string,
	orm string,
	projectName string, serviceName string,
	tables sqlproto.TableDataArray,
	services []string,
	moduleVersion string,
	servicePackageMap map[string]string,
	wctx *wiringContext,
) error {
	if len(tables) == 0 {
		return nil
	}

	// 先生成 client 代码（只生成一次）
	switch orm {
	case "ent":
		if err := g.writeEntClientCode(outputPath, projectName, serviceName); err != nil {
			return err
		}
	case "gorm":
		if err := g.writeGormClientCode(outputPath, projectName, serviceName); err != nil {
			return err
		}
	}

	// 收集所有模型名
	var modelNames []string

	var dataFields []generators.DataField
	for _, table := range tables {
		if len(table.Fields) == 0 {
			continue
		}

		name := inflection.Singular(table.Name)
		modelNames = append(modelNames, name)

		// 从 servicePackageMap 获取策略决定的 proto module 名
		moduleName := servicePackageMap[name]
		if moduleName == "" {
			moduleName = strings.ToLower(name)
		}

		dataFields = make([]generators.DataField, 0)
		for _, field := range table.Fields {
			if field.Type == "" {
				continue
			}

			dataField := generators.DataField{
				Name:         field.Name,
				Type:         field.Type,
				SqlType:      field.SqlType,
				Comment:      field.Comment,
				Null:         field.Null,
				IsPrimaryKey: field.IsPrimaryKey,
			}
			dataFields = append(dataFields, dataField)
		}

		// 生成 repo 代码
		switch orm {
		case "ent":
			if err := g.writeEntRepoCode(outputPath, projectName, serviceName, name, moduleName, moduleVersion, dataFields); err != nil {
				return err
			}
		case "gorm":
			if err := g.writeGormRepoCode(outputPath, projectName, serviceName, name, moduleName, moduleVersion, dataFields); err != nil {
				return err
			}
		}
	}

	// gorm_init 在所有模型收集完后生成
	if orm == "gorm" {
		if err := g.writeGormInitCode(outputPath, projectName, serviceName, modelNames); err != nil {
			return err
		}
	}

	// 装配登记:旧式 wire 服务写入 provider 集;手写装配服务向装配文件锚点注入仓储构造行。
	if !wctx.useWiringDI {
		var clientFunctions []string
		switch orm {
		case "ent":
			clientFunctions = append(clientFunctions, "client.NewEntClient")
		case "gorm":
			clientFunctions = append(clientFunctions, "client.NewGormClient")
		}

		var allFunctions []string
		allFunctions = append(allFunctions, clientFunctions...)
		for _, svc := range services {
			allFunctions = append(allFunctions, fmt.Sprintf("data.New%sRepo", stringcase.UpperCamelCase(svc)))
		}

		return g.WriteDataWireSetCode(outputPath, projectName, serviceName, allFunctions)
	}

	if wctx.wiringFile != "" && !wctx.isBff && orm != "" {
		for _, model := range services {
			if err := g.injectWiringRepoLine(wctx.wiringFile, projectName, serviceName, model, wctx.ormClientVar); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Generator) generateMainPackageCode(
	outputPath string,

	projectName string, serviceName string,

	servers []string,
	services []string,
	wctx *wiringContext,
	opts GeneratorOptions,
) error {
	if err := g.WriteMainCode(
		outputPath,
		projectName, serviceName,
		servers,
	); err != nil {
		return err
	}

	if !wctx.useWiringDI {
		return g.WriteWireCode(
			outputPath,
			projectName, serviceName,
		)
	}

	// 既有装配文件保持手工维护,不重渲;全新服务渲染装配骨架。
	if wctx.wiringFile != "" {
		return nil
	}

	var ormForBlocks string
	var dbClients []string
	var repoModels []string
	if !wctx.isBff && opts.OrmType != "" && opts.GenerateData {
		ormForBlocks = opts.OrmType
		dbClients = []string{opts.OrmType}
		repoModels = append(repoModels, services...)
	}
	var serviceModels []string
	if opts.GenerateService {
		serviceModels = append(serviceModels, services...)
	}

	blocks := generators.BuildWiringBlocks(
		servers,
		opts.UseRepo,
		ormForBlocks,
		dbClients,
		repoModels,
		serviceModels,
	)
	_, err := g.goGenerator.GenerateWiring(context.Background(), code_generator.Options{
		OutDir: outputPath,
		Module: projectName,
		Vars: map[string]any{
			"Service": serviceName,
		},
	}, blocks)
	return err
}

// generateConfigCode 生成配置文件 (client.yaml, server.yaml, logger.yaml, data.yaml)
func (g *Generator) generateConfigCode(ctx context.Context, configPath string, servers []string) error {
	log.Println("Generating config files...")

	if _, err := g.yamlGenerator.GenerateLoggerYaml(ctx, code_generator.Options{
		OutDir: configPath,
	}); err != nil {
		return err
	}

	if _, err := g.yamlGenerator.GenerateDataYaml(ctx, code_generator.Options{
		OutDir: configPath,
	}); err != nil {
		return err
	}

	if _, err := g.yamlGenerator.GenerateClientYaml(ctx, code_generator.Options{
		OutDir: configPath,
	}); err != nil {
		return err
	}

	if _, err := g.yamlGenerator.GenerateServerYaml(ctx, code_generator.Options{
		OutDir: configPath,
	}); err != nil {
		return err
	}

	log.Println("Config files generation completed.")
	return nil
}

// generateMakefileCode 生成 Makefile
func (g *Generator) generateMakefileCode(ctx context.Context, servicePath string) error {
	log.Println("Generating Makefile...")

	_, err := g.makefileGenerator.GenerateAppMakefile(ctx, code_generator.Options{
		OutDir: servicePath,
	})

	if err != nil {
		return err
	}

	log.Println("Makefile generation completed.")
	return nil
}
