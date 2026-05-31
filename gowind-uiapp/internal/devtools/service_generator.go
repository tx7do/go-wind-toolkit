package devtools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-utils/stringcase"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

// ServiceGenerator 服务脚手架生成器
type ServiceGenerator struct {
	projectModule string
	projectName   string
	serviceName   string
	outputPath    string
	servers       []string
	dbClients     []string

	goGenerator       *generators.GoGenerator
	yamlGenerator     *generators.YamlGenerator
	makefileGenerator *generators.MakefileGenerator
}

// NewServiceGenerator 创建服务生成器
func NewServiceGenerator(projectModule, projectName, serviceName, outputPath string, servers, dbClients []string) *ServiceGenerator {
	return &ServiceGenerator{
		projectModule:     projectModule,
		projectName:       projectName,
		serviceName:       serviceName,
		outputPath:        outputPath,
		servers:           servers,
		dbClients:         dbClients,
		goGenerator:       generators.NewGoGenerator(),
		yamlGenerator:     generators.NewYamlGenerator(),
		makefileGenerator: generators.NewMakefileGenerator(),
	}
}

// Generate 生成完整的服务脚手架
func (g *ServiceGenerator) Generate() error {
	var err error

	// 生成 server 层
	if err = g.generateServer(); err != nil {
		return err
	}

	// 生成 service 层
	if err = g.generateServiceLayer(); err != nil {
		return err
	}

	// 生成 data 层
	if err = g.generateData(); err != nil {
		return err
	}

	// 生成 main
	if err = g.generateMain(); err != nil {
		return err
	}

	// 生成 Makefile
	if err = g.generateMakefile(); err != nil {
		return err
	}

	// 生成 configs
	if err = g.generateConfigs(); err != nil {
		return err
	}

	// 追加服务名称常量
	if err = g.appendServiceName(); err != nil {
		return err
	}

	return nil
}

func (g *ServiceGenerator) generateServer() error {
	serverPath := filepath.Join(g.outputPath, "app", g.serviceName, "service", "internal", "server")

	for _, server := range g.servers {
		switch strings.ToLower(server) {
		case "grpc":
			o := code_generator.Options{
				OutDir: serverPath,
				Module: g.projectModule,
				Vars: map[string]any{
					"Service": g.serviceName,
				},
			}
			if _, err := g.goGenerator.GenerateGrpcServer(context.Background(), o); err != nil {
				return fmt.Errorf("生成 gRPC server 失败: %w", err)
			}
		case "rest":
			o := code_generator.Options{
				OutDir: serverPath,
				Module: g.projectModule,
				Vars: map[string]any{
					"Service": g.serviceName,
				},
			}
			if _, err := g.goGenerator.GenerateRestServer(context.Background(), o); err != nil {
				return fmt.Errorf("生成 REST server 失败: %w", err)
			}
		}
	}

	// 生成 wire_set
	var newFunctions []string
	for _, server := range g.servers {
		funcName := "New" + stringcase.ToPascalCase(server) + "Server"
		newFunctions = append(newFunctions, funcName)
	}
	opts := code_generator.Options{
		OutDir: filepath.Join(serverPath, "providers"),
		Module: g.projectModule,
		Vars: map[string]any{
			"Service":      g.serviceName,
			"Package":      "server",
			"NewFunctions": newFunctions,
		},
	}
	_, err := g.goGenerator.GenerateWireSet(context.Background(), opts)
	return err
}

func (g *ServiceGenerator) generateServiceLayer() error {
	servicePath := filepath.Join(g.outputPath, "app", g.serviceName, "service", "internal", "service")

	var newFunctions []string
	newFunctions = append(newFunctions, "New"+stringcase.ToPascalCase(g.serviceName)+"Service")

	opts := code_generator.Options{
		OutDir: filepath.Join(servicePath, "providers"),
		Module: g.projectModule,
		Vars: map[string]any{
			"Service":      g.serviceName,
			"Package":      "service",
			"NewFunctions": newFunctions,
		},
	}
	_, err := g.goGenerator.GenerateWireSet(context.Background(), opts)
	return err
}

func (g *ServiceGenerator) generateData() error {
	dataPath := filepath.Join(g.outputPath, "app", g.serviceName, "service", "internal", "data")
	o := code_generator.Options{
		OutDir: dataPath,
		Module: g.projectModule,
		Vars: map[string]any{
			"Service": g.serviceName,
		},
	}

	var functions []string
	for _, dbClient := range g.dbClients {
		switch strings.ToLower(dbClient) {
		case "redis":
			o.Vars["HasRedis"] = true
		case "gorm":
			o.Vars["HasGorm"] = true
		case "ent", "entgo":
			o.Vars["HasEnt"] = true
		}
		functions = append(functions, "New"+stringcase.UpperCamelCase(dbClient)+"Client")
	}

	opts := code_generator.Options{
		OutDir: filepath.Join(dataPath, "providers"),
		Module: g.projectModule,
		Vars: map[string]any{
			"Service":      g.serviceName,
			"Package":      "data",
			"NewFunctions": functions,
		},
	}
	_, err := g.goGenerator.GenerateWireSet(context.Background(), opts)
	return err
}

func (g *ServiceGenerator) generateMain() error {
	mainPath := filepath.Join(g.outputPath, "app", g.serviceName, "service", "cmd", "server")

	opts := code_generator.Options{
		OutDir: mainPath,
		Module: g.projectModule,
		Vars: map[string]any{
			"Service":                  g.serviceName,
			"ServerImports":            generators.ServerImportPaths(g.servers),
			"ServerFormalParameters":   generators.ServerFormalParameters(g.servers),
			"ServerTransferParameters": generators.ServerTransferParameters(g.servers),
		},
	}

	if _, err := g.goGenerator.GenerateMain(context.Background(), opts); err != nil {
		return fmt.Errorf("生成 main 失败: %w", err)
	}

	// 生成 wire
	wireOpts := code_generator.Options{
		OutDir: mainPath,
		Module: g.projectModule,
		Vars: map[string]any{
			"Service": g.serviceName,
		},
	}
	_, err := g.goGenerator.GenerateWire(context.Background(), wireOpts)
	return err
}

func (g *ServiceGenerator) generateMakefile() error {
	makefilePath := filepath.Join(g.outputPath, "app", g.serviceName, "service")
	if err := os.MkdirAll(makefilePath, os.ModePerm); err != nil {
		return err
	}

	_, err := g.makefileGenerator.GenerateAppMakefile(context.Background(), code_generator.Options{
		OutDir: makefilePath,
	})
	return err
}

func (g *ServiceGenerator) generateConfigs() error {
	configsPath := filepath.Join(g.outputPath, "app", g.serviceName, "service", "configs")
	ctx := context.Background()

	if _, err := g.yamlGenerator.GenerateServerYaml(ctx, code_generator.Options{OutDir: configsPath}); err != nil {
		return err
	}
	if _, err := g.yamlGenerator.GenerateClientYaml(ctx, code_generator.Options{OutDir: configsPath}); err != nil {
		return err
	}
	if _, err := g.yamlGenerator.GenerateDataYaml(ctx, code_generator.Options{OutDir: configsPath}); err != nil {
		return err
	}
	if _, err := g.yamlGenerator.GenerateLoggerYaml(ctx, code_generator.Options{OutDir: configsPath}); err != nil {
		return err
	}
	return nil
}

func (g *ServiceGenerator) appendServiceName() error {
	serviceIdPath := filepath.Join(g.outputPath, "pkg", "serviceid")
	if err := os.MkdirAll(serviceIdPath, os.ModePerm); err != nil {
		return fmt.Errorf("创建 pkg/serviceid 目录失败: %w", err)
	}

	constName := fmt.Sprintf("%sService", stringcase.UpperCamelCase(g.serviceName))
	constValue := fmt.Sprintf("%s-%s-service", stringcase.LowerCamelCase(g.projectName), strings.ToLower(g.serviceName))
	fieldLine := fmt.Sprintf("    %s = %q", constName, constValue)

	serviceNamePath := filepath.Join(serviceIdPath, "service_id.go")

	if _, err := os.Stat(serviceNamePath); os.IsNotExist(err) {
		content := fmt.Sprintf("package service\n\nconst (\n%s\n)\n", fieldLine)
		return os.WriteFile(serviceNamePath, []byte(content), 0644)
	}

	data, err := os.ReadFile(serviceNamePath)
	if err != nil {
		return err
	}
	text := string(data)
	if strings.Contains(text, constName) {
		return nil
	}

	constIdx := strings.Index(text, "const (")
	if constIdx >= 0 {
		closeIdx := strings.Index(text[constIdx:], ")")
		if closeIdx >= 0 {
			insertPos := constIdx + closeIdx
			newText := text[:insertPos] + "\n" + fieldLine + "\n" + text[insertPos:]
			return os.WriteFile(serviceNamePath, []byte(newText), 0644)
		}
	}

	appendContent := fmt.Sprintf("\nconst (\n%s\n)\n", fieldLine)
	return os.WriteFile(serviceNamePath, []byte(text+appendContent), 0644)
}
