package gorm

import (
	"context"
	"errors"
	"os"

	"gorm.io/gen"
)

func Importer(_ context.Context, drv, dsn, schemaPath, daoPath *string, tables, _ []string) error {
	if schemaPath == nil {
		return errors.New("gormimport: schema path is nil")
	}
	if daoPath == nil {
		return errors.New("gormimport: dao path is nil")
	}
	if dsn == nil {
		return errors.New("gormimport: dsn is nil")
	}
	if drv == nil {
		return errors.New("gormimport: drv is nil")
	}

	_ = os.MkdirAll(*schemaPath, os.ModePerm)
	_ = os.MkdirAll(*daoPath, os.ModePerm)

	db := NewGormClient(*drv, *dsn)

	g := gen.NewGenerator(gen.Config{
		OutPath:      *daoPath,    // 生成的代码路径
		ModelPkgPath: *schemaPath, // 模型包路径

		Mode: gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // 不生成上下文

		WithUnitTest:      false, // 生成单元测试
		FieldNullable:     true,  // 指针类型字段允许为 nil
		FieldCoverable:    true,  // 生成可覆盖字段
		FieldSignable:     true,  // 有符号数和无符号数自动转换
		FieldWithIndexTag: true,  // 生成索引标签
		FieldWithTypeTag:  true,  // 生成类型标签
	})

	g.UseDB(db) // 设置数据库连接

	var models []any
	if len(tables) > 0 {
		for _, t := range tables {
			models = append(models, g.GenerateModel(t))
		}
	} else {
		models = g.GenerateAllTable()
	}

	// 生成所有表的模型和 DAO
	g.ApplyBasic(models...)

	// 生成代码
	g.Execute()

	return nil
}
