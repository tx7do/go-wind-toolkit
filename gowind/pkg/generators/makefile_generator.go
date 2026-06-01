package generators

import (
	"context"
	"os"

	"github.com/tx7do/go-utils/code_generator"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators/templates/makefile"
)

// MakefileGenerator 使用 TemplateEngine 渲染并将结果写入磁盘
type MakefileGenerator struct {
	*code_generator.CodeGenerator
}

// NewMakefileGenerator 创建生成器，engine 可为 nil（需要在调用前设置）
func NewMakefileGenerator() *MakefileGenerator {
	templateEngine, _ := code_generator.NewEmbeddedTemplateEngineFromMap(makefile.TemplateMap, funcMap)

	codeGenerator := code_generator.NewCodeGeneratorWithEngine(templateEngine)
	codeGenerator.FileExt = ""

	g := &MakefileGenerator{
		CodeGenerator: codeGenerator,
	}

	return g
}

func (g *MakefileGenerator) GenerateAppMakefile(ctx context.Context, opts code_generator.Options) (outputPath string, err error) {
	if g.CodeGenerator == nil {
		return "", os.ErrInvalid
	}

	opts.OutputName = "Makefile"

	return g.Generate(ctx, opts, "app_makefile.tpl")
}
