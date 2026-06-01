package service

import (
	"context"

	pkgService "github.com/tx7do/go-wind-toolkit/gowind/pkg/service"
)

// GeneratorOptions 服务生成器选项（委托给 pkg/service）
type GeneratorOptions = pkgService.GeneratorOptions

// Generate 生成服务脚手架代码
func Generate(ctx context.Context, opts GeneratorOptions) error {
	return pkgService.Generate(ctx, opts)
}

// NewGenerator 创建服务代码生成器
func NewGenerator() *pkgService.Generator {
	return pkgService.NewGenerator()
}
