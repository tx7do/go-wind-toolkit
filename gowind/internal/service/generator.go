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

func Generate(ctx context.Context, opts GeneratorOptions) error {
	g := NewGenerator()
	return g.Generate(ctx, opts)
}

type Generator struct {
	goGenerator       *generators.GoGenerator
	yamlGenerator     *generators.YamlGenerator
	makefileGenerator *generators.MakefileGenerator
}

func NewGenerator() *Generator {
	return &Generator{
		goGenerator:       generators.NewGoGenerator(),
		yamlGenerator:     generators.NewYamlGenerator(),
		makefileGenerator: generators.NewMakefileGenerator(),
	}
}

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
			opts.ProjectName,
			opts.ServiceName,
			opts.DbClients,
			[]string{},
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
		}
	}

	return g.writeWireSetCode(outputPath, projectModule, serviceName, "server", "Server", servers)
}

func (g *Generator) generateServicePackageCode(
	outputPath string,
	projectName string,
	serviceName string,
	services []string,
) error {
	return g.writeWireSetCode(outputPath, projectName, serviceName, "service", "Service", services)
}

func (g *Generator) generateDataPackageCode(
	outputPath string,
	projectModule string,
	projectName string,
	serviceName string,
	dbClients []string,
	repos []string,
) error {
	o := code_generator.Options{
		OutDir: outputPath,
		Module: projectModule,
		Vars: map[string]any{
			"Service": serviceName,
		},
	}

	for _, dbClient := range dbClients {
		switch strings.ToLower(dbClient) {
		case "redis":
			o.Vars["HasRedis"] = true
		case "gorm":
			o.Vars["HasGorm"] = true
		case "ent", "entgo":
			o.Vars["HasEnt"] = true
		}
	}

	var functions []string
	for _, repo := range repos {
		functions = append(functions, fmt.Sprintf("New%sRepo", stringcase.UpperCamelCase(repo)))
	}
	for _, dbClient := range dbClients {
		functions = append(functions, fmt.Sprintf("New%sClient", stringcase.UpperCamelCase(dbClient)))
	}
	//log.Printf("functions: %v\n", functions)
	return g.writeWireSetFunctionCode(outputPath, projectModule, serviceName, "data", functions)
}

func (g *Generator) generateMainPackageCode(
	outputPath string,
	moduleName string, serviceName string,
	servers []string,
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

	return g.writeWireCode(
		outputPath,
		moduleName, serviceName,
	)
}

// writeMakefile 生成默认的 Makefile 到指定目录。
func (g *Generator) writeMakefile(outputPath string) error {
	outputPath = filepath.Clean(outputPath)
	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
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
	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
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
	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
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
	services []string,
) error {
	var newFunctions []string
	for _, service := range services {
		funcName := "New" + stringcase.ToPascalCase(service) + postfix
		newFunctions = append(newFunctions, funcName)
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
