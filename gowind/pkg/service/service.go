package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-utils/stringcase"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

// Generate 生成服务脚手架代码
func Generate(ctx context.Context, opts GeneratorOptions) error {
	g := NewGenerator()
	return g.Generate(ctx, opts)
}

// GeneratorOptions 保存代码生成器的选项。
type GeneratorOptions struct {
	OutputPath string

	ProjectModule string
	ProjectName   string
	ServiceName   string

	Servers   []string
	DbClients []string

	// UseWireDI 为真时生成旧式 wire 依赖注入(wire.go + 各层 providers/wire_set.go),
	// 与既有 wire 工程向下兼容;默认为假,生成手写装配文件(cmd/server/wiring.go)。
	UseWireDI bool

	GenerateServer   bool
	GenerateService  bool
	GenerateData     bool
	GenerateMain     bool
	GenerateMakefile bool
	GenerateConfigs  bool
}

// HasBFFService checks whether the BFF (Backend For Frontend) service is included in the generator options.
func (o GeneratorOptions) HasBFFService() bool {
	isBff := false
	for _, server := range o.Servers {
		if strings.ToLower(server) == "rest" {
			isBff = true
			break
		}
	}
	return isBff
}

// Generator 服务代码生成器
type Generator struct {
	goGenerator       *generators.GoGenerator
	yamlGenerator     *generators.YamlGenerator
	makefileGenerator *generators.MakefileGenerator
}

// NewGenerator 创建服务代码生成器
func NewGenerator() *Generator {
	return &Generator{
		goGenerator:       generators.NewGoGenerator(),
		yamlGenerator:     generators.NewYamlGenerator(),
		makefileGenerator: generators.NewMakefileGenerator(),
	}
}

// Generate 执行服务代码生成
func (g *Generator) Generate(_ context.Context, opts GeneratorOptions) error {
	var err error

	// 生成server层代码
	if opts.GenerateServer {
		serverPackagePath := filepath.Join(opts.OutputPath, "/app/", opts.ServiceName, "/service/internal/server")
		if err = g.generateServerPackageCode(
			serverPackagePath,
			opts.ProjectModule,
			opts.ServiceName,
			opts.Servers,
			opts.UseWireDI,
		); err != nil {
			return err
		}
	}

	// 生成service层代码
	if opts.GenerateService {
		servicePackagePath := filepath.Join(opts.OutputPath, "/app/", opts.ServiceName, "/service/internal/service")
		if err = g.generateServicePackageCode(
			servicePackagePath,
			opts.ProjectModule,
			opts.ServiceName,
			[]string{},
			opts.UseWireDI,
		); err != nil {
			return err
		}
	}

	// 生成data层代码
	if opts.GenerateData {
		dataPackagePath := filepath.Join(opts.OutputPath, "/app/", opts.ServiceName, "/service/internal/data")
		if err = g.generateDataPackageCode(
			dataPackagePath,
			opts.ProjectModule,
			opts.ServiceName,
			opts.DbClients,
			opts.UseWireDI,
		); err != nil {
			return err
		}
	}

	// 生成main包代码
	if opts.GenerateMain {
		mainPackagePath := filepath.Join(opts.OutputPath, "/app/", opts.ServiceName, "/service/cmd/server")
		if err = g.generateMainPackageCode(
			mainPackagePath,
			opts.ProjectModule,
			opts.ServiceName,
			opts.Servers,
			opts.UseWireDI,
			opts.DbClients,
		); err != nil {
			return err
		}
	}

	// 生成Makefile
	if opts.GenerateMakefile {
		makefilePath := filepath.Join(opts.OutputPath, "/app/", opts.ServiceName, "/service")
		if err = g.writeMakefile(makefilePath); err != nil {
			return err
		}
	}

	// 生成configs
	if opts.GenerateConfigs {
		configsPath := filepath.Join(opts.OutputPath, "/app/", opts.ServiceName, "/service/configs")
		if err = g.writeConfigs(configsPath); err != nil {
			return err
		}
	}

	// 追加服务名称常量定义
	{
		servicePackagePath := filepath.Join(opts.OutputPath, "/pkg/serviceid")
		if err = g.appendServiceName(
			servicePackagePath,
			opts.ProjectName,
			opts.ServiceName,
			opts.HasBFFService(),
		); err != nil {
			return err
		}
	}

	// 生成assets包代码
	if opts.HasBFFService() {
		assetsPath := filepath.Join(opts.OutputPath, "/app/", opts.ServiceName, "/service/cmd/server/assets")
		if err = g.writeAssets(assetsPath); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) generateServerPackageCode(
	outputPath string,
	projectModule string,
	serviceName string,
	servers []string,
	useWireDI bool,
) error {
	for _, server := range servers {
		switch strings.ToLower(server) {
		case "grpc":
			o := code_generator.Options{
				OutDir: outputPath,
				Module: projectModule,
				Vars: map[string]any{
					"Service": serviceName,
				},
			}
			if _, err := g.goGenerator.GenerateGrpcServer(context.Background(), o); err != nil {
				return err
			}
		case "rest":
			o := code_generator.Options{
				OutDir: outputPath,
				Module: projectModule,
				Vars: map[string]any{
					"Service": serviceName,
				},
			}
			if _, err := g.goGenerator.GenerateRestServer(context.Background(), o); err != nil {
				return err
			}
		case "websocket":
			o := code_generator.Options{
				OutDir: outputPath,
				Module: projectModule,
				Vars: map[string]any{
					"Service": serviceName,
				},
			}
			if _, err := g.goGenerator.GenerateWebsocketServer(context.Background(), o); err != nil {
				return err
			}
		}
	}

	if !useWireDI {
		return nil
	}

	return g.writeWireSetCode(outputPath, projectModule, serviceName, "server", "Server", servers)
}

func (g *Generator) generateServicePackageCode(
	outputPath string,
	projectName string,
	serviceName string,
	services []string,
	useWireDI bool,
) error {
	if !useWireDI {
		return nil
	}
	return g.writeWireSetCode(outputPath, projectName, serviceName, "service", "Service", services)
}

func (g *Generator) generateDataPackageCode(
	outputPath string,
	projectModule string,
	serviceName string,
	dbClients []string,
	useWireDI bool,
) error {
	// 生成数据层客户端文件(internal/data/client/*.go)。
	clientPath := filepath.Join(outputPath, "client")
	for _, dbClient := range dbClients {
		o := code_generator.Options{
			OutDir: clientPath,
			Module: projectModule,
			Vars: map[string]any{
				"Service": serviceName,
			},
		}
		switch strings.ToLower(dbClient) {
		case "redis":
			if _, err := g.goGenerator.GenerateRedisClient(context.Background(), o); err != nil {
				return err
			}
		case "gorm":
			if _, err := g.goGenerator.GenerateGormClient(context.Background(), o); err != nil {
				return err
			}
		case "ent", "entgo":
			if _, err := g.goGenerator.GenerateEntClient(context.Background(), o); err != nil {
				return err
			}
		}
	}

	if !useWireDI {
		return nil
	}

	var functions []string
	for _, dbClient := range dbClients {
		functions = append(functions, fmt.Sprintf("client.New%sClient", stringcase.UpperCamelCase(dbClient)))
	}
	return g.writeWireSetFunctionCode(outputPath, projectModule, serviceName, "data", functions)
}

func (g *Generator) generateMainPackageCode(
	outputPath string,
	moduleName string, serviceName string,
	servers []string,
	useWireDI bool,
	dbClients []string,
) error {
	opts := code_generator.Options{
		OutDir: outputPath,
		Module: moduleName,
		Vars: map[string]any{
			"Service":                  serviceName,
			"ServerImports":            generators.ServerImportPaths(servers),
			"ServerFormalParameters":   generators.ServerFormalParameters(servers),
			"ServerTransferParameters": generators.ServerTransferParameters(servers),
		},
	}

	_, err := g.goGenerator.GenerateMain(context.Background(), opts)
	if err != nil {
		return err
	}

	if useWireDI {
		return g.writeWireCode(
			outputPath,
			moduleName, serviceName,
		)
	}

	// 手写装配骨架:分层小节、cleanup 注册表与登记锚点,初始不含任何模块登记行。
	blocks := generators.BuildWiringBlocks(servers, false, "", dbClients, nil, nil)
	_, err = g.goGenerator.GenerateWiring(context.Background(), code_generator.Options{
		OutDir: outputPath,
		Module: moduleName,
		Vars: map[string]any{
			"Service": serviceName,
		},
	}, blocks)
	return err
}

// writeMakefile 生成默认的 Makefile 到指定目录。
func (g *Generator) writeMakefile(outputPath string) error {
	outputPath = filepath.Clean(outputPath)
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return err
	}

	if _, err := g.makefileGenerator.GenerateAppMakefile(context.Background(), code_generator.Options{
		OutDir: outputPath,
	}); err != nil {
		return err
	}

	return nil
}

// writeConfigs 生成默认的配置文件到指定目录。
func (g *Generator) writeConfigs(outputPath string) error {
	ctx := context.Background()
	var err error

	if _, err = g.yamlGenerator.GenerateServerYaml(ctx, code_generator.Options{
		OutDir: outputPath,
	}); err != nil {
		return err
	}

	if _, err = g.yamlGenerator.GenerateClientYaml(ctx, code_generator.Options{
		OutDir: outputPath,
	}); err != nil {
		return err
	}

	if _, err = g.yamlGenerator.GenerateDataYaml(ctx, code_generator.Options{
		OutDir: outputPath,
	}); err != nil {
		return err
	}

	if _, err = g.yamlGenerator.GenerateLoggerYaml(ctx, code_generator.Options{
		OutDir: outputPath,
	}); err != nil {
		return err
	}

	return nil
}

// appendServiceName 向 pkg/serviceid/service_id.go 文件追加服务名称常量定义。
func (g *Generator) appendServiceName(outputPath string, projectName, serviceName string, isBff bool) error {
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("create pkg/serviceid dir: %w", err)
	}

	servicePostfix := "service"
	if isBff {
		servicePostfix = "bff"
	}

	// 常量名与值
	constName := fmt.Sprintf("%sService", stringcase.UpperCamelCase(serviceName))
	constValue := fmt.Sprintf("%s-%s-%s", stringcase.LowerCamelCase(projectName), strings.ToLower(serviceName), servicePostfix)

	// 行格式，带缩进
	fieldLine := fmt.Sprintf("    %s = %q", constName, constValue)

	serviceNamePath := filepath.Join(outputPath, "service_id.go")

	// 文件不存在：创建包含 const 块的初始文件
	if !pkg.IsFileExists(serviceNamePath) {
		content := fmt.Sprintf("package service\n\nconst (\n%s\n)\n", fieldLine)
		if err := os.WriteFile(serviceNamePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write service name file: %w", err)
		}
		return nil
	}

	// 文件存在：读取并检查是否已包含常量名
	data, err := os.ReadFile(serviceNamePath)
	if err != nil {
		return fmt.Errorf("read service name file: %w", err)
	}
	text := string(data)
	if strings.Contains(text, constName) {
		// 已包含，不需插入
		return nil
	}

	// 找到第一个 const ( ... ) 块并在闭合 ) 之前插入新行
	constIdx := strings.Index(text, "const (")
	if constIdx >= 0 {
		// 在 const ( 后寻找对应的第一个 )（简单实现，适用于代码生成的文件）
		closeIdx := strings.Index(text[constIdx:], ")")
		if closeIdx >= 0 {
			insertPos := constIdx + closeIdx
			newText := text[:insertPos] + "\n" + fieldLine + "\n" + text[insertPos:]
			if err = os.WriteFile(serviceNamePath, []byte(newText), 0644); err != nil {
				return fmt.Errorf("write service name file after insert: %w", err)
			}
			return nil
		}
	}

	// 未找到 const 块，直接在文件末尾追加一个新的 const 块
	appendContent := fmt.Sprintf("\nconst (\n%s\n)\n", fieldLine)
	newText := text + appendContent
	if err = os.WriteFile(serviceNamePath, []byte(newText), 0644); err != nil {
		return fmt.Errorf("append service name file: %w", err)
	}

	return nil
}

// writeAssets 生成 assets 包代码到指定目录。
func (g *Generator) writeAssets(outputPath string) error {
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("create assets dir: %w", err)
	}

	if _, err := g.goGenerator.GenerateAssets(context.Background(), code_generator.Options{
		OutDir: outputPath,
	}); err != nil {
		return err
	}

	return nil
}

func (g *Generator) writeWireSetCode(
	outputPath string,
	projectModule string,
	serviceName string,
	packageName string,
	postfix string,
	servers []string,
) error {
	var newFunctions []string
	for _, server := range servers {
		funcName := "New" + stringcase.ToPascalCase(server) + postfix
		newFunctions = append(newFunctions, funcName)

		// server 层的中间件构造器与 server 构造器同属 provider 集
		// (server 模板签名含 middlewares 形参;仅当文件中确实定义了对应构造器时登记)。
		if packageName == "server" {
			serverFile := filepath.Join(outputPath, strings.ToLower(server)+"_server.go")
			if raw, err := os.ReadFile(serverFile); err == nil &&
				strings.Contains(string(raw), "func New"+stringcase.ToPascalCase(server)+"Middleware(") {
				newFunctions = append(newFunctions, "server.New"+stringcase.ToPascalCase(server)+"Middleware")
			}
		}
	}

	opts := code_generator.Options{
		OutDir: filepath.Join(outputPath, "providers"),
		Module: projectModule,
		Vars: map[string]any{
			"Service":      serviceName,
			"Package":      packageName,
			"NewFunctions": newFunctions,
		},
	}
	_, err := g.goGenerator.GenerateWireSet(context.Background(), opts)
	return err
}

func (g *Generator) writeWireSetFunctionCode(
	outputPath string,
	projectModule string,
	serviceName string,
	packageName string,
	functions []string,
) error {
	opts := code_generator.Options{
		OutDir: filepath.Join(outputPath, "providers"),
		Module: projectModule,
		Vars: map[string]any{
			"Service":      serviceName,
			"Package":      packageName,
			"NewFunctions": functions,
		},
	}
	_, err := g.goGenerator.GenerateWireSet(context.Background(), opts)
	return err
}

func (g *Generator) writeWireCode(
	outputPath string,
	projectName string,
	serviceName string,
) error {
	opts := code_generator.Options{
		OutDir: outputPath,
		Module: projectName,
		Vars: map[string]any{
			"Service": serviceName,
		},
	}
	_, err := g.goGenerator.GenerateWire(context.Background(), opts)
	return err
}
