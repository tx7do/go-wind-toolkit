package generators

import (
	"context"
	"os"

	"github.com/tx7do/go-utils/code_generator"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators/templates/yaml"
)

// YamlGenerator 使用 TemplateEngine 渲染并将结果写入磁盘
type YamlGenerator struct {
	*code_generator.CodeGenerator
}

// NewYamlGenerator 创建生成器，engine 可为 nil（需要在调用前设置）
func NewYamlGenerator() *YamlGenerator {
	templateEngine, _ := code_generator.NewEmbeddedTemplateEngineFromMap(yaml.TemplateMap, funcMap)

	codeGenerator := code_generator.NewCodeGeneratorWithEngine(templateEngine)
	codeGenerator.FileExt = ".yaml"

	g := &YamlGenerator{
		CodeGenerator: codeGenerator,
	}

	return g
}

func (g *YamlGenerator) GenerateClientYaml(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	opts.OutputName = "client.yaml"

	return g.Generate(ctx, opts, "client_yaml.tpl")
}

func (g *YamlGenerator) GenerateServerYaml(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	opts.OutputName = "server.yaml"

	return g.Generate(ctx, opts, "server_yaml.tpl")
}

func (g *YamlGenerator) GenerateLoggerYaml(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	opts.OutputName = "logger.yaml"

	return g.Generate(ctx, opts, "logger_yaml.tpl")
}

func (g *YamlGenerator) GenerateDataYaml(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	opts.OutputName = "data.yaml"

	return g.Generate(ctx, opts, "data_yaml.tpl")
}
