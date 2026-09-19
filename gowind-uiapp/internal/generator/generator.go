package generator

import (
	"context"
	"fmt"
	"sync"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/devtools"
	sqlkratos "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlkratos"
)

// Logger 生成过程的日志接口（由调用方注入：GUI 注入 wails runtime 日志，CLI 注入标准输出）
type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}

type Generator struct {
	// mu 保护 options/logger/skipPostProcess:Wails 绑定调用各自在独立
	// goroutine 中执行,前端并发触发选项读写时存在数据竞争。
	mu             sync.Mutex
	options        GeneratorOptions
	logger         Logger
	// skipPostProcess 跳过生成后的 tidy/buf/ent/wire 后处理链
	skipPostProcess bool
}

func NewGenerator() *Generator {
	return &Generator{
		options: GeneratorOptions{},
		logger:  noopLogger{},
	}
}

// SetLogger 注入日志实现（默认丢弃日志）
func (g *Generator) SetLogger(logger Logger) {
	if logger == nil {
		logger = noopLogger{}
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.logger = logger
}

// SetSkipPostProcess 跳过 gRPC 生成后的 tidy/buf/ent/wire 后处理链
func (g *Generator) SetSkipPostProcess(skip bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.skipPostProcess = skip
}

// GetOptions 获取选项(副本)
func (g *Generator) GetOptions() GeneratorOptions {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make(GeneratorOptions, len(g.options))
	copy(out, g.options)
	return out
}

// SetOptions 设置选项（深拷贝）
func (g *Generator) SetOptions(options GeneratorOptions) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// 深拷贝：创建新切片并为每个 Option 创建新实例
	if len(options) == 0 {
		g.options = GeneratorOptions{}
		return
	}
	g.options = make(GeneratorOptions, len(options))
	for i, opt := range options {
		// 为每个 Option 创建新实例，避免共享指针
		newOpt := &Option{
			ID:           opt.ID,
			TableName:    opt.TableName,
			Service:      opt.Service,
			Exclude:      opt.Exclude,
			ProtoPackage: opt.ProtoPackage,
		}
		g.options[i] = newOpt
	}
}

// EditOption 编辑已有的选项
func (g *Generator) EditOption(o *Option) {
	if o == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()
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

	g.mu.Lock()
	defer g.mu.Unlock()
	o.ID = uint32(len(g.options) + 1)
	g.options = append(g.options, o)
}

// CleanOptions 清空所有选项
func (g *Generator) CleanOptions() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.options = GeneratorOptions{}
}

// ValidateOptions 验证选项的有效性，返回错误信息字符串，如果没有错误则返回空字符串
func (g *Generator) ValidateOptions() string {
	g.mu.Lock()
	defer g.mu.Unlock()
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

// GetValidateOptions 获取通过验证的选项列表(副本)
func (g *Generator) GetValidateOptions() GeneratorOptions {
	g.mu.Lock()
	defer g.mu.Unlock()
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
	protoPackageStrategy string,
	rootPath string,
	projectName string,
) error {
	opts := g.GetValidateOptions()
	if len(opts) == 0 {
		g.logger.Errorf("没有可用的表选项进行代码生成")
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

		g.logger.Infof("开始为服务生成代码: %s", serviceName)

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
				g.logger.Errorf("构建数据库连接字符串失败: %v", err)
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
		options.ProtoPackageStrategy = protoPackageStrategy

		options.OutputPath = rootPath

		for _, opt := range serviceOpts {
			options.IncludedTables = append(options.IncludedTables, opt.TableName)
			if opt.ProtoPackage != "" {
				if options.TableCustomPackages == nil {
					options.TableCustomPackages = make(map[string]string)
				}
				options.TableCustomPackages[opt.TableName] = opt.ProtoPackage
			}
		}

		if err := sqlkratos.Generate(ctx, options); err != nil {
			g.logger.Errorf("生成代码失败: %v", err)
			return err
		}
	}

	// === 后处理步骤（可通过 SkipPostProcess 跳过，如需由调用方自行控制各步骤） ===
	if g.skipPostProcess {
		g.logger.Infof("跳过后处理（tidy/buf/ent/wire）")
		return nil
	}

	// 1. go mod tidy
	g.logger.Infof("运行 go mod tidy...")
	if result := devtools.RunGoModTidy(rootPath); !result.Success {
		g.logger.Errorf("go mod tidy 失败: %s\n%s", result.Error, result.Output)
		return fmt.Errorf("go mod tidy 失败: %s\n%s", result.Error, result.Output)
	}

	// 2. buf generate（生成 protobuf 代码）
	g.logger.Infof("运行 buf generate...")
	if result := devtools.RunBufGenerate(rootPath); !result.Success {
		g.logger.Errorf("buf generate 失败: %s\n%s", result.Error, result.Output)
		return fmt.Errorf("buf generate 失败: %s\n%s", result.Error, result.Output)
	}

	// 3. 如果是 ent ORM，执行 ent generate
	if ormType == "ent" {
		for _, svcName := range serviceNames {
			g.logger.Infof("运行 ent generate: %s", svcName)
			if result := devtools.RunEntGenerate(rootPath, svcName); !result.Success {
				g.logger.Errorf("ent generate 失败 (%s): %s\n%s", svcName, result.Error, result.Output)
				return fmt.Errorf("ent generate 失败 (%s): %s\n%s", svcName, result.Error, result.Output)
			}
		}
	}

	// 4. wire generate（依赖注入代码生成）
	for _, svcName := range serviceNames {
		g.logger.Infof("运行 wire generate: %s", svcName)
		if result := devtools.RunWire(rootPath, svcName); !result.Success {
			g.logger.Errorf("wire 生成失败 (%s): %s\n%s", svcName, result.Error, result.Output)
			return fmt.Errorf("wire 生成失败 (%s): %s\n%s", svcName, result.Error, result.Output)
		}
	}

	return nil
}

func (g *Generator) GenerateRestCode(
	ctx context.Context,
	restServiceName string,
	ormType string,
	protoPackageStrategy string,
	dbConfig database.DBConfig,
	rootPath string,
	projectName string,
) error {
	opts := g.GetValidateOptions()
	if len(opts) == 0 {
		g.logger.Errorf("没有可用的表选项进行代码生成")
		return fmt.Errorf("没有可用的表选项进行代码生成")
	}

	mapOpts := make(map[string]GeneratorOptions)
	for _, opt := range opts {
		mapOpts[opt.Service] = append(mapOpts[opt.Service], opt)
	}

	for serviceName, serviceOpts := range mapOpts {
		var options sqlkratos.GeneratorOptions

		g.logger.Infof("开始为服务生成代码: %s", serviceName)

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
				g.logger.Errorf("构建数据库连接字符串失败: %v", err)
				return err
			}
			options.Source = dsn
		}

		options.UseRepo = false
		options.GenerateProto = true
		options.GenerateORM = false
		options.GenerateData = false
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
		options.ProtoPackageStrategy = protoPackageStrategy

		options.OutputPath = rootPath

		for _, opt := range serviceOpts {
			options.IncludedTables = append(options.IncludedTables, opt.TableName)
			if opt.ProtoPackage != "" {
				if options.TableCustomPackages == nil {
					options.TableCustomPackages = make(map[string]string)
				}
				options.TableCustomPackages[opt.TableName] = opt.ProtoPackage
			}
		}

		if err := sqlkratos.Generate(ctx, options); err != nil {
			g.logger.Errorf("生成代码失败: %v", err)
			return err
		}
	}

	return nil
}
