package generators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators/templates/golang"
)

// GoGenerator 使用 TemplateEngine 渲染并将结果写入磁盘
type GoGenerator struct {
	*code_generator.CodeGenerator
}

// NewGoGenerator 创建生成器，engine 可为 nil（需要在调用前设置）
func NewGoGenerator() *GoGenerator {
	templateEngine, _ := code_generator.NewEmbeddedTemplateEngineFromMap(golang.TemplateMap, funcMap)

	codeGenerator := code_generator.NewCodeGeneratorWithEngine(templateEngine)

	g := &GoGenerator{
		CodeGenerator: codeGenerator,
	}

	return g
}

func (g *GoGenerator) GenerateMain(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}
	return g.Generate(ctx, opts, "main.tpl")
}

func (g *GoGenerator) GenerateWire(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}
	return g.Generate(ctx, opts, "wire.tpl")
}

func (g *GoGenerator) GenerateWireSet(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	var packageName string
	if v, ok := opts.Vars["Package"]; ok {
		packageName, _ = v.(string)
	}

	// 构建完整的函数调用列表
	var fullFunctionCalls []string
	if _, ok := opts.Vars["NewFunctions"]; ok && packageName != "" {
		newFunctions, _ := opts.Vars["NewFunctions"].([]string)
		for _, fn := range newFunctions {
			fullFunctionCalls = append(fullFunctionCalls, fmt.Sprintf("%s.%s", packageName, fn))
		}
	}

	// 构建输出路径
	outputPath = opts.OutDir
	if opts.OutputName != "" {
		outputPath = filepath.Join(opts.OutDir, opts.OutputName)
	} else {
		outputPath = filepath.Join(opts.OutDir, "wire_set.go")
	}

	// 检查文件是否存在
	if _, err = os.Stat(outputPath); err == nil {
		// 文件存在，使用 UpsertProviderSetFunctions 追加
		if len(fullFunctionCalls) > 0 {
			err = g.UpsertProviderSetFunctions(outputPath, fullFunctionCalls)
			if err != nil {
				return "", fmt.Errorf("failed to upsert provider set functions: %w", err)
			}
		}
		return outputPath, nil
	} else if !os.IsNotExist(err) {
		// 其他错误
		return "", fmt.Errorf("failed to check file existence: %w", err)
	}

	// 文件不存在，使用模板全新创建
	// 更新 opts.Vars 以供模板使用
	if len(fullFunctionCalls) > 0 {
		opts.Vars["NewFunctions"] = fullFunctionCalls
	}

	return g.Generate(ctx, opts, "wire_set.tpl")
}

func (g *GoGenerator) GenerateEntClient(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}
	return g.Generate(ctx, opts, "ent_client.tpl")
}

func (g *GoGenerator) GenerateEntRepo(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	var modelName string
	if v, ok := opts.Vars["Model"]; ok {
		modelName, _ = v.(string)
	}

	if _, ok := opts.Vars["ClassName"]; !ok {
		opts.Vars["ClassName"] = stringcase.ToPascalCase(modelName) + "Repo"
	}

	if _, ok := opts.Vars["ApiPackageVersion"]; !ok {
		opts.Vars["ApiPackageVersion"] = "v1"
	}

	if opts.OutputName == "" {
		opts.OutputName = stringcase.ToSnakeCase(modelName) + "_repo.go"
	}

	return g.Generate(ctx, opts, "ent_repo.tpl")
}

func (g *GoGenerator) GenerateGormClient(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}
	return g.Generate(ctx, opts, "gorm_client.tpl")
}

func (g *GoGenerator) GenerateGormInit(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}
	return g.Generate(ctx, opts, "gorm_init.tpl")
}

func (g *GoGenerator) GenerateGormRepo(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	var modelName string
	if v, ok := opts.Vars["Model"]; ok {
		modelName, _ = v.(string)
	}

	if _, ok := opts.Vars["ClassName"]; !ok {
		opts.Vars["ClassName"] = stringcase.ToPascalCase(modelName) + "Repo"
	}

	if _, ok := opts.Vars["ApiPackageVersion"]; !ok {
		opts.Vars["ApiPackageVersion"] = "v1"
	}

	if opts.OutputName == "" {
		opts.OutputName = stringcase.ToSnakeCase(modelName) + "_repo.go"
	}

	return g.Generate(ctx, opts, "gorm_repo.tpl")
}

func (g *GoGenerator) GenerateGrpcServer(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	if _, ok := opts.Vars["ApiPackageVersion"]; !ok {
		opts.Vars["ApiPackageVersion"] = "v1"
	}

	return g.Generate(ctx, opts, "grpc_server.tpl")
}

func (g *GoGenerator) GenerateRedisClient(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	return g.Generate(ctx, opts, "redis_client.tpl")
}

func (g *GoGenerator) GenerateRestServer(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	if _, ok := opts.Vars["ApiPackageVersion"]; !ok {
		opts.Vars["ApiPackageVersion"] = "v1"
	}

	return g.Generate(ctx, opts, "rest_server.tpl")
}

func (g *GoGenerator) GenerateService(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	var modelName string
	if v, ok := opts.Vars["Model"]; ok {
		modelName, _ = v.(string)
	}

	var isGrpcService bool
	if v, ok := opts.Vars["IsGrpc"]; ok {
		isGrpcService, _ = v.(bool)
	}

	if _, ok := opts.Vars["ClassName"]; !ok {

		opts.Vars["ClassName"] = stringcase.ToPascalCase(modelName) + "Service"
	}

	if _, ok := opts.Vars["TargetApiPackageVersion"]; !ok {
		opts.Vars["TargetApiPackageVersion"] = "v1"
	}
	if _, ok := opts.Vars["SourceApiPackageVersion"]; !ok {
		opts.Vars["SourceApiPackageVersion"] = "v1"
	}

	if _, ok := opts.Vars["SourceApiPackage"]; !ok {
		opts.Vars["SourceApiPackage"] = stringcase.LowerCamelCase(opts.Vars["SourceApiPackageName"].(string)) + stringcase.UpperCamelCase(opts.Vars["SourceApiPackageVersion"].(string))
	}
	if _, ok := opts.Vars["TargetApiPackage"]; !ok {
		opts.Vars["TargetApiPackage"] = stringcase.LowerCamelCase(opts.Vars["TargetApiPackageName"].(string)) + stringcase.UpperCamelCase(opts.Vars["TargetApiPackageVersion"].(string))
	}

	if _, ok := opts.Vars["ServiceInterface"]; !ok {
		if isGrpcService {
			opts.Vars["ServiceInterface"] = fmt.Sprintf("%s.Unimplemented%sServiceServer",
				opts.Vars["TargetApiPackage"].(string),
				stringcase.ToPascalCase(opts.Vars["Model"].(string)))
		} else {
			opts.Vars["ServiceInterface"] = fmt.Sprintf("%s.%sServiceHTTPServer",
				opts.Vars["TargetApiPackage"].(string),
				stringcase.ToPascalCase(opts.Vars["Model"].(string)))
		}
	}

	if _, ok := opts.Vars["DataSourceVar"]; !ok {
		if isGrpcService {
			opts.Vars["DataSourceVar"] = stringcase.LowerCamelCase(opts.Vars["Model"].(string)) + "Repo"
		} else {
			opts.Vars["DataSourceVar"] = stringcase.LowerCamelCase(opts.Vars["Model"].(string)) + "ServiceClient"
		}
	}
	if _, ok := opts.Vars["DataSourceType"]; !ok {
		if isGrpcService {
			opts.Vars["DataSourceType"] = "*data." + stringcase.UpperCamelCase(opts.Vars["Model"].(string)) + "Repo"
		} else {
			opts.Vars["DataSourceType"] = fmt.Sprintf("%s.%sServiceClient",
				opts.Vars["SourceApiPackage"].(string),
				stringcase.UpperCamelCase(opts.Vars["Model"].(string)))
		}
	}

	if _, ok := opts.Vars["IsSameApi"]; !ok {
		opts.Vars["IsSameApi"] = opts.Vars["SourceApiPackage"].(string) == opts.Vars["TargetApiPackage"].(string)
	}

	if _, ok := opts.Vars["UseRepo"]; !ok {
		if isGrpcService {
			opts.Vars["UseRepo"] = true
		}
	}

	if opts.OutputName == "" {
		opts.OutputName = stringcase.ToSnakeCase(modelName) + "_service.go"
	}

	return g.Generate(ctx, opts, "service.tpl")
}

func (g *GoGenerator) GenerateAssets(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	opts.OutputName = "assets.go"

	return g.Generate(ctx, opts, "assets.tpl")
}

// UpsertProviderSetFunction 向 ProviderSet 中添加函数，如果不存在则新增
// filePath: wire.go 文件路径
// functionCall: 要添加的函数调用，如 "server.NewRestServer"
func (g *GoGenerator) UpsertProviderSetFunction(filePath string, functionCall string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fileContent := string(content)

	// 查找 ProviderSet 定义
	// 匹配 var ProviderSet = wire.NewSet(...) 的模式
	providerSetPattern := regexp.MustCompile(`(var\s+ProviderSet\s*=\s*wire\.NewSet\s*\(\s*)((?:[^)]+|\n)+)(\s*\))`)

	matches := providerSetPattern.FindStringSubmatch(fileContent)
	if matches == nil {
		return fmt.Errorf("ProviderSet definition not found in file")
	}

	prefix := matches[1]        // "var ProviderSet = wire.NewSet("
	existingFuncs := matches[2] // 现有的函数列表
	suffix := matches[3]        // ")"

	// 检查函数是否已存在
	funcPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(functionCall) + `\b`)
	if funcPattern.MatchString(existingFuncs) {
		// 函数已存在，不需要添加
		return nil
	}

	// 准备新的函数列表
	// 先去除首尾空白，但保留内部结构
	trimmedFuncs := strings.TrimSpace(existingFuncs)

	// 添加新函数
	var newFuncs string
	if trimmedFuncs == "" {
		// 如果是空的 NewSet，直接添加
		newFuncs = "\n\t" + functionCall + ",\n"
	} else {
		// 确保现有函数列表以逗号结尾（如果不是空行）
		// 移除可能的尾部逗号和空白，然后统一添加
		trimmedFuncs = strings.TrimRight(trimmedFuncs, ", \t\n")
		// 在现有函数后添加，确保格式正确
		newFuncs = trimmedFuncs + ",\n\n\t" + functionCall + ",\n"
	}

	// 重新组合内容
	newProviderSet := prefix + newFuncs + suffix
	newContent := providerSetPattern.ReplaceAllString(fileContent, newProviderSet)

	// 写回文件
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// UpsertProviderSetFunctions 批量向 ProviderSet 中添加函数
func (g *GoGenerator) UpsertProviderSetFunctions(filePath string, functionCalls []string) error {
	for _, funcCall := range functionCalls {
		if err := g.UpsertProviderSetFunction(filePath, funcCall); err != nil {
			return err
		}
	}
	return nil
}

// CheckProviderSetFunctionExists 检查 ProviderSet 中是否存在指定函数
func (g *GoGenerator) CheckProviderSetFunctionExists(filePath string, functionCall string) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	fileContent := string(content)

	// 查找 ProviderSet 定义
	providerSetPattern := regexp.MustCompile(`var\s+ProviderSet\s*=\s*wire\.NewSet\s*\(\s*((?:[^)]+|\n)+)\s*\)`)
	matches := providerSetPattern.FindStringSubmatch(fileContent)
	if matches == nil {
		return false, fmt.Errorf("ProviderSet definition not found in file")
	}

	existingFuncs := matches[1]

	// 检查函数是否存在
	funcPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(functionCall) + `\b`)
	return funcPattern.MatchString(existingFuncs), nil
}
