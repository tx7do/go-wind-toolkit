package generators

import (
	"context"
	"os"

	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-utils/stringcase"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators/templates/proto"
)

// ProtoGenerator 使用 TemplateEngine 渲染并将结果写入磁盘
type ProtoGenerator struct {
	*code_generator.CodeGenerator
}

// NewProtoGenerator 创建生成器，engine 可为 nil（需要在调用前设置）
func NewProtoGenerator() *ProtoGenerator {
	templateEngine, _ := code_generator.NewEmbeddedTemplateEngineFromMap(proto.TemplateMap, funcMap)

	codeGenerator := code_generator.NewCodeGeneratorWithEngine(templateEngine)
	codeGenerator.FileExt = ".proto"

	g := &ProtoGenerator{
		CodeGenerator: codeGenerator,
	}

	return g
}

func (g *ProtoGenerator) GenerateGrpcServiceProto(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	var modelName string
	if v, ok := opts.Vars["Model"]; ok {
		modelName, _ = v.(string)
	}

	if opts.OutputName == "" {
		opts.OutputName = stringcase.ToSnakeCase(modelName) + ProtoFilePostfix
	}

	return g.Generate(ctx, opts, "grpc_proto.tpl")
}

func (g *ProtoGenerator) GenerateRestServiceProto(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	var modelName string
	if v, ok := opts.Vars["Model"]; ok {
		modelName, _ = v.(string)
	}

	if opts.OutputName == "" {
		opts.OutputName = RestProtoFilePrefix + stringcase.ToSnakeCase(modelName) + ProtoFilePostfix
	}

	return g.Generate(ctx, opts, "rest_proto.tpl")
}
