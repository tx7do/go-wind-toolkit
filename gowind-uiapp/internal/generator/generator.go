package generator

import (
	"context"
	"fmt"

	"github.com/labstack/gommon/log"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/devtools"
	sqlkratos "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlkratos"
)

type Generator struct {
	options GeneratorOptions
}

func NewGenerator() *Generator {
	return &Generator{
		options: GeneratorOptions{},
	}
}

// GetOptions 获取选项
func (g *Generator) GetOptions() GeneratorOptions {
	return g.options
}

// SetOptions 设置选项
func (g *Generator) SetOptions(options GeneratorOptions) {
	g.options = options
}

// EditOption 编辑已有的选项
func (g *Generator) EditOption(o *Option) {
	if o == nil {
		return
	}

	for i, opt := range g.options {
		if opt.TableName == o.TableName {
			g.options[i] = o
			return
		}
	}
}

// AddOption 添加新的选项
func (g *Generator) AddOption(o *Option) {
	if o == nil {
		return
	}

	if o.TableName == "" {
		return
	}

	o.ID = uint32(len(g.options) + 1)

	g.options = append(g.options, o)
}

// CleanOptions 清空所有选项
func (g *Generator) CleanOptions() {
	g.options = GeneratorOptions{}
}

// ValidateOptions 验证选项的有效性，返回错误信息字符串，如果没有错误则返回空字符串
func (g *Generator) ValidateOptions() string {
	if len(g.options) == 0 {
		return "no tables selected"
	}

	for _, opt := range g.options {
		if opt.TableName == "" {
			return "table name cannot be empty"
		}
		if opt.Service == "" {
			return "service name cannot be empty"
		}
	}

	return ""
}

// GetValidateOptions 获取通过验证的选项列表
func (g *Generator) GetValidateOptions() GeneratorOptions {
	var options GeneratorOptions
	for _, opt := range g.options {
		if opt.TableName != "" &&
			opt.Service != "" &&
			!opt.Exclude {
			options = append(options, opt)
		}
	}
	return options
}

// GenerateGrpcCode 生成代码
func (g *Generator) GenerateGrpcCode(
	ctx context.Context,
	dbConfig database.DBConfig,
	ormType string,
	rootPath string,
	projectName string,
) error {
	opts := g.GetValidateOptions()
	if len(opts) == 0 {
		runtime.LogErrorf(ctx, "没有可用的表选项进行代码生成")
		return fmt.Errorf("没有可用的表选项进行代码生成")
	}

	mapOpts := make(map[string]GeneratorOptions)
	for _, opt := range opts {
		mapOpts[opt.Service] = append(mapOpts[opt.Service], opt)
	}

	var serviceNames []string

	for serviceName, serviceOpts := range mapOpts {
		serviceNames = append(serviceNames, serviceName)

		var options sqlkratos.GeneratorOptions

		log.Info("开始为服务生成代码: ", serviceName)

		options.OrmType = ormType
		options.Driver = string(dbConfig.Type)

		if dbConfig.SQLContent != "" {
			options.Source = dbConfig.SQLContent
		} else if dbConfig.UseDSN {
			options.Source = dbConfig.DSN
		} else {
			// 构建 DSN
			dsn, err := database.BuildDSN(dbConfig)
			if err != nil {
				runtime.LogErrorf(ctx, "构建数据库连接字符串失败: %v", err)
				return err
			}
			options.Source = dsn
		}

		options.UseRepo = true
		options.GenerateProto = true
		options.GenerateORM = true
		options.GenerateData = true
		options.GenerateService = true
		options.GenerateServer = true
		options.GenerateMain = true
		options.GenerateConfig = true
		options.GenerateMakefile = true

		options.Servers = []string{"grpc"}

		options.ProjectName = projectName
		options.ServiceName = serviceName

		options.SourceModuleName = serviceName
		options.ModuleName = serviceName
		options.ModuleVersion = "v1"

		options.OutputPath = rootPath

		for _, opt := range serviceOpts {
			options.IncludedTables = append(options.IncludedTables, opt.TableName)
		}

		if err := sqlkratos.Generate(ctx, options); err != nil {
			runtime.LogErrorf(ctx, "生成代码失败: %v", err)
			return err
		}
	}

	// === 后处理步骤 ===

	// 1. go mod tidy
	log.Info("运行 go mod tidy...")
	if result := devtools.RunGoModTidy(rootPath); !result.Success {
		runtime.LogErrorf(ctx, "go mod tidy 失败: %s\n%s", result.Error, result.Output)
		return fmt.Errorf("go mod tidy 失败: %s\n%s", result.Error, result.Output)
	}

	// 2. buf generate（生成 protobuf 代码）
	log.Info("运行 buf generate...")
	if result := devtools.RunBufGenerate(rootPath); !result.Success {
		runtime.LogErrorf(ctx, "buf generate 失败: %s\n%s", result.Error, result.Output)
		return fmt.Errorf("buf generate 失败: %s\n%s", result.Error, result.Output)
	}

	// 3. 如果是 ent ORM，执行 ent generate
	if ormType == "ent" {
		for _, svcName := range serviceNames {
			log.Info("运行 ent generate: ", svcName)
			if result := devtools.RunEntGenerate(rootPath, svcName); !result.Success {
				runtime.LogErrorf(ctx, "ent generate 失败 (%s): %s\n%s", svcName, result.Error, result.Output)
				return fmt.Errorf("ent generate 失败 (%s): %s\n%s", svcName, result.Error, result.Output)
			}
		}
	}

	// 4. wire generate（依赖注入代码生成）
	for _, svcName := range serviceNames {
		log.Info("运行 wire generate: ", svcName)
		if result := devtools.RunWire(rootPath, svcName); !result.Success {
			runtime.LogErrorf(ctx, "wire 生成失败 (%s): %s\n%s", svcName, result.Error, result.Output)
			return fmt.Errorf("wire 生成失败 (%s): %s\n%s", svcName, result.Error, result.Output)
		}
	}

	return nil
}

func (g *Generator) GenerateRestCode(
	ctx context.Context,
	restServiceName string,
	ormType string,
	dbConfig database.DBConfig,
	rootPath string,
	projectName string,
) error {
	opts := g.GetValidateOptions()
	if len(opts) == 0 {
		runtime.LogErrorf(ctx, "没有可用的表选项进行代码生成")
		return fmt.Errorf("没有可用的表选项进行代码生成")
	}

	mapOpts := make(map[string]GeneratorOptions)
	for _, opt := range opts {
		mapOpts[opt.Service] = append(mapOpts[opt.Service], opt)
	}

	for serviceName, serviceOpts := range mapOpts {
		var options sqlkratos.GeneratorOptions

		log.Info("开始为服务生成代码: ", serviceName)

		options.Driver = string(dbConfig.Type)
		options.OrmType = ormType

		if dbConfig.SQLContent != "" {
			options.Source = dbConfig.SQLContent
		} else if dbConfig.UseDSN {
			options.Source = dbConfig.DSN
		} else {
			// 构建 DSN
			dsn, err := database.BuildDSN(dbConfig)
			if err != nil {
				runtime.LogErrorf(ctx, "构建数据库连接字符串失败: %v", err)
				return err
			}
			options.Source = dsn
		}

		options.UseRepo = true
		options.GenerateProto = true
		options.GenerateORM = false
		options.GenerateData = true
		options.GenerateService = true
		options.GenerateServer = true
		options.GenerateMain = true
		options.GenerateConfig = true
		options.GenerateMakefile = true

		options.Servers = []string{"rest"}

		options.ProjectName = projectName
		options.ServiceName = restServiceName

		options.SourceModuleName = serviceName
		options.ModuleName = restServiceName
		options.ModuleVersion = "v1"

		options.OutputPath = rootPath

		for _, opt := range serviceOpts {
			options.IncludedTables = append(options.IncludedTables, opt.TableName)
		}

		if err := sqlkratos.Generate(ctx, options); err != nil {
			runtime.LogErrorf(ctx, "生成代码失败: %v", err)
			return err
		}
	}

	return nil
}
