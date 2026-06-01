package sqlkratos

import (
	"context"
	"errors"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

func (g *Generator) WriteDataPackageCode(
	outputPath string,
	orm string,
	projectName string,
	serviceName string,
	name string,
	moduleName, moduleVersion string,
	protoFields []generators.DataField,
) error {
	var copyDataFields generators.DataFieldArray
	for _, field := range protoFields {
		if field.Type == "" {
			continue
		}

		copyDataField := generators.DataField{
			Name:         field.Name,
			Type:         field.Type,
			SqlType:      field.SqlType,
			Comment:      field.Comment,
			Null:         field.Null,
			IsPrimaryKey: field.IsPrimaryKey,
		}
		copyDataFields = append(copyDataFields, copyDataField)
	}

	switch strings.TrimSpace(strings.ToLower(orm)) {
	case "ent":
		if err := g.writeEntClientCode(outputPath, projectName, serviceName); err != nil {
			return err
		}
		return g.writeEntRepoCode(outputPath, projectName, serviceName, name, moduleName, moduleVersion, copyDataFields)

	case "gorm":
		if err := g.writeGormClientCode(outputPath, projectName, serviceName); err != nil {
			return err
		}
		return g.writeGormRepoCode(outputPath, projectName, serviceName, name, moduleName, moduleVersion, copyDataFields)

	default:
		return errors.New("sqlproto: unsupported orm: " + orm)
	}
}

func (g *Generator) writeEntClientCode(
	outputPath string,
	projectModule string,
	serviceName string,
) error {
	clientPath := path.Join(outputPath, "client")
	opts := code_generator.Options{
		OutDir: clientPath,
		Module: projectModule,
		Vars: map[string]any{
			"Service": serviceName,
		},
	}

	_, err := g.goGenerator.GenerateEntClient(context.Background(), opts)
	return err
}

func (g *Generator) writeGormClientCode(
	outputPath string,
	projectModule string,
	serviceName string,
) error {
	clientPath := path.Join(outputPath, "client")

	opts := code_generator.Options{
		OutDir: clientPath,
		Module: projectModule,
		Vars: map[string]any{
			"Service": serviceName,
		},
	}
	_, err := g.goGenerator.GenerateGormClient(context.Background(), opts)
	return err
}

func (g *Generator) writeGormInitCode(
	outputPath string,
	projectModule string,
	serviceName string,
	models []string,
) error {
	gormPath := path.Join(outputPath, "gorm")

	opts := code_generator.Options{
		OutDir: gormPath,
		Module: projectModule,
		Vars: map[string]any{
			"Service": serviceName,
			"Models":  models,
		},
	}
	_, err := g.goGenerator.GenerateGormInit(context.Background(), opts)
	return err
}

func (g *Generator) writeEntRepoCode(
	outputPath string,
	projectModule string,
	serviceName string,
	model string,
	apiPackageName string,
	apiPackageVersion string,
	fields generators.DataFieldArray,
) error {
	opts := code_generator.Options{
		OutDir: outputPath,
		Module: projectModule,
		Vars: map[string]any{
			"Service":                     serviceName,
			"ApiPackage":                  stringcase.LowerCamelCase(apiPackageName) + stringcase.UpperCamelCase(apiPackageVersion),
			"Model":                       model,
			"Fields":                      fields,
			"HasTimeConversionField":      fields.HasTimeConversionField(),
			"HasStringNumConversionField": fields.HasStringNumConversionField(),
		},
	}

	_, err := g.goGenerator.GenerateEntRepo(context.Background(), opts)
	return err
}

func (g *Generator) writeGormRepoCode(
	outputPath string,
	projectModule string,
	serviceName string,
	model string,
	apiPackageName string,
	apiPackageVersion string,
	fields generators.DataFieldArray,
) error {
	opts := code_generator.Options{
		OutDir: outputPath,
		Module: projectModule,
		Vars: map[string]any{
			"Service":    serviceName,
			"ApiPackage": stringcase.LowerCamelCase(apiPackageName) + stringcase.UpperCamelCase(apiPackageVersion),
			"Model":      model,
			"Fields":     fields,
		},
	}

	_, err := g.goGenerator.GenerateGormRepo(context.Background(), opts)
	return err
}

func (g *Generator) WriteServicePackageCode(
	outputPath string,
	projectName string,
	serviceName string,
	name string,
	targetModuleName, sourceModuleName, moduleVersion string,
	useRepo, isGrpcService bool,
) error {
	o := code_generator.Options{
		OutDir: outputPath,
		Module: projectName,
		Vars: map[string]any{
			"TargetApiPackageName":    targetModuleName,
			"TargetApiPackageVersion": moduleVersion,

			"SourceApiPackageName":    sourceModuleName,
			"SourceApiPackageVersion": moduleVersion,

			"Service": serviceName,
			"Model":   name,
			"IsGrpc":  isGrpcService,
			"UseRepo": useRepo,
		},
	}

	log.Printf("Generating service package code for model: %s %s", name, outputPath)

	if _, err := g.goGenerator.GenerateService(context.Background(), o); err != nil {
		return err
	}

	return nil
}

func (g *Generator) WriteServerPackageCode(
	outputPath string,
	projectName string,
	serviceType string,
	serviceName string,
	services map[string]string,
) error {
	switch strings.ToLower(serviceType) {
	case "grpc":
		// 从 Services 中提取去重的模块名，用于生成 import 语句
		packagesMap := make(map[string]string)
		for _, moduleName := range services {
			packagesMap[moduleName] = moduleName
		}
		o := code_generator.Options{
			OutDir: outputPath,
			Module: projectName,
			Vars: map[string]any{
				"Service":  serviceName,
				"Services": services,
				"Packages": packagesMap,
			},
		}
		if _, err := g.goGenerator.GenerateGrpcServer(context.Background(), o); err != nil {
			return err
		}

	case "rest":
		// 从 Services 中提取去重的模块名，用于生成 import 语句
		packagesMap := make(map[string]string)
		for _, moduleName := range services {
			packagesMap[moduleName] = moduleName
		}
		o := code_generator.Options{
			OutDir: outputPath,
			Module: projectName,
			Vars: map[string]any{
				"Service":  serviceName,
				"Services": services,
				"Packages": packagesMap,
			},
		}
		if _, err := g.goGenerator.GenerateRestServer(context.Background(), o); err != nil {
			return err
		}

	default:
		return errors.New("sqlproto: unsupported service type: " + serviceType)
	}

	return nil
}

func (g *Generator) WriteWireSetCode(
	outputPath string,
	projectModule string,
	serviceName string,
	packageName string,
	postfix string,
	services []string,
) error {
	var newFunctions []string
	for _, service := range services {
		newFunction := "New" + stringcase.UpperCamelCase(service) + postfix
		newFunctions = append(newFunctions, newFunction)
	}

	opts := code_generator.Options{
		OutDir: filepath.Join(outputPath, "providers"),
		Module: projectModule,
		Vars: map[string]any{
			"Service":      serviceName,
			"Package":      packageName,
			"NewFunctions": newFunctions,
		},
	}
	_, err := g.goGenerator.GenerateWireSet(context.Background(), opts)
	return err
}

// WriteDataWireSetCode 生成 data 层的 wire_set，支持 client 和 data 两个包的函数
func (g *Generator) WriteDataWireSetCode(
	outputPath string,
	projectModule string,
	serviceName string,
	allFunctions []string,
) error {
	providersPath := filepath.Join(outputPath, "providers")

	// 检查是否有 client. 前缀的函数
	var extraImports []string
	for _, fn := range allFunctions {
		if strings.HasPrefix(fn, "client.") {
			extraImports = append(extraImports, "data/client")
			break
		}
	}

	opts := code_generator.Options{
		OutDir: providersPath,
		Module: projectModule,
		Vars: map[string]any{
			"Service":      serviceName,
			"Package":      "data",
			"NewFunctions": allFunctions,
			"ExtraImports": extraImports,
		},
	}

	_, err := g.goGenerator.GenerateWireSet(context.Background(), opts)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) WriteWireCode(
	outputPath string,

	projectName string,
	serviceName string,
) error {
	opts := code_generator.Options{
		OutDir: outputPath,
		Module: projectName,
		Vars: map[string]any{
			"Service": serviceName,
		},
	}
	_, err := g.goGenerator.GenerateWire(context.Background(), opts)
	return err
}

func (g *Generator) WriteMainCode(
	outputPath string,

	projectName string,
	serviceName string,

	servers []string,
) error {
	opts := code_generator.Options{
		OutDir: outputPath,
		Module: projectName,
		Vars: map[string]any{
			"Service":                  serviceName,
			"ServerImports":            generators.ServerImportPaths(servers),
			"ServerFormalParameters":   generators.ServerFormalParameters(servers),
			"ServerTransferParameters": generators.ServerTransferParameters(servers),
		},
	}

	_, err := g.goGenerator.GenerateMain(context.Background(), opts)
	return err
}
