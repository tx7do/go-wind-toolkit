package frontendgen

import "fmt"

// Generate 按选项生成前端代码文件
func Generate(opts Options) ([]GeneratedFile, error) {
	if opts.Spec == nil {
		return nil, fmt.Errorf("OpenAPI 规格不能为空")
	}

	var files []GeneratedFile
	switch opts.Framework {
	case FrameworkVueVben:
		files = generateVben(opts)
	case FrameworkVueElement:
		files = generateElement(opts)
	case FrameworkReactAntd:
		files = generateReact(opts)
	default:
		return nil, fmt.Errorf("不支持的前端框架: %q（可选 vue-element / vue-vben / react）", string(opts.Framework))
	}

	files = append(files, generateErrorI18N(opts)...)
	return files, nil
}
