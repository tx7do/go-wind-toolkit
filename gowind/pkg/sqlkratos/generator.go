package sqlkratos

import (
	"context"
	"fmt"
	"log"
	"path"

	"github.com/jinzhu/inflection"
	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"

	sqlorm "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlorm"
	sqlproto "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto"
)

func Generate(ctx context.Context, opts GeneratorOptions) error {
	g := NewGenerator()
	return g.Generate(ctx, opts)
}

type Generator struct {
	goGenerator       *generators.GoGenerator
	yamlGenerator     *generators.YamlGenerator
	makefileGenerator *generators.MakefileGenerator
	protoGenerator    *generators.ProtoGenerator
}

func NewGenerator() *Generator {
	return &Generator{
		goGenerator:       generators.NewGoGenerator(),
		yamlGenerator:     generators.NewYamlGenerator(),
		makefileGenerator: generators.NewMakefileGenerator(),
		protoGenerator:    generators.NewProtoGenerator(),
	}
}

func (g *Generator) Generate(ctx context.Context, opts GeneratorOptions) error {
	var err error

	var tables sqlproto.TableDataArray

	// 生成 Protobuf schema
	if tables, err = g.generateProtobufCode(ctx, opts); err != nil {
		return err
	}

	services := make([]string, 0)
	servicePackageMap := make(map[string]string)
	for _, table := range tables {
		if len(table.Fields) == 0 {
			continue
		}

		name := inflection.Singular(table.Name)

		services = append(services, name)
		servicePackageMap[name] = opts.ModuleName
	}

	var useGrpc bool
	for _, server := range opts.Servers {
		if server == "grpc" {
			useGrpc = true
			break
		}
	}

	// 生成ORM代码
	if opts.GenerateORM {
		dataPackagePath := fmt.Sprintf("%s/app/%s/service/internal/", opts.OutputPath, opts.ModuleName)
		if err = g.generateOrmCode(ctx, opts, dataPackagePath); err != nil {
			return err
		}
	}

	// 生成data层代码
	if opts.GenerateData {
		dataPackagePath := fmt.Sprintf("%s/app/%s/service/internal/data", opts.OutputPath, opts.ModuleName)
		if err = g.generateDataPackageCode(
			dataPackagePath,
			opts.OrmType,
			opts.ProjectName,
			opts.ServiceName,
			opts.ModuleName, opts.ModuleVersion,
			tables,
			services,
		); err != nil {
			return err
		}
	}

	// 生成service层代码
	if opts.GenerateService {
		servicePackagePath := fmt.Sprintf("%s/app/%s/service/internal/service/", opts.OutputPath, opts.ModuleName)
		servicePackagePath = path.Clean(servicePackagePath)
		log.Printf("Generating service package code at: %s", servicePackagePath)
		if err = g.generateServicePackageCode(
			servicePackagePath,
			opts.ProjectName,
			opts.ServiceName,
			opts.ModuleName, opts.SourceModuleName, opts.ModuleVersion,
			opts.UseRepo, useGrpc,
			tables,
			services,
		); err != nil {
			return err
		}
	}

	// 生成server层代码
	if opts.GenerateServer {
		serverPackagePath := fmt.Sprintf("%s/app/%s/service/internal/server/", opts.OutputPath, opts.ModuleName)
		serverPackagePath = path.Clean(serverPackagePath)
		log.Printf("Generating server package code at: %s", serverPackagePath)
		if err = g.generateServerPackageCode(
			serverPackagePath,
			opts.ProjectName,
			opts.ServiceName,
			servicePackageMap,
			opts.Servers,
		); err != nil {
			return err
		}
	}

	// 生成配置文件
	if opts.GenerateConfig {
		configPath := fmt.Sprintf("%s/app/%s/service/configs", opts.OutputPath, opts.ModuleName)
		if err = g.generateConfigCode(ctx, configPath, opts.Servers); err != nil {
			return err
		}
	}

	// 生成Makefile
	if opts.GenerateMakefile {
		makefilePath := fmt.Sprintf("%s/app/%s/service", opts.OutputPath, opts.ModuleName)
		if err = g.generateMakefileCode(ctx, makefilePath); err != nil {
			return err
		}
	}

	// 生成main包代码
	if opts.GenerateMain {
		mainPackagePath := fmt.Sprintf("%s/app/%s/service/cmd/server", opts.OutputPath, opts.ModuleName)
		if err = g.generateMainPackageCode(
			mainPackagePath,
			opts.ProjectName,
			opts.ServiceName,
			opts.Servers,
		); err != nil {
			return err
		}
	}

	return nil
}

// generateProtobufCode generates the Protobuf code from the database schema.
func (g *Generator) generateProtobufCode(ctx context.Context, opts GeneratorOptions) (sqlproto.TableDataArray, error) {
	var err error
	var tables sqlproto.TableDataArray

	protoPath := path.Join(opts.OutputPath, "/api/protos/")

	for _, server := range opts.Servers {
		if server != "grpc" && server != "rest" {
			continue
		}

		if tables, err = sqlproto.Convert(
			ctx,
			&opts.Source,
			&protoPath,
			&opts.ModuleName,
			&opts.SourceModuleName,
			&opts.ModuleVersion,
			&server,
			opts.IncludedTables,
			opts.ExcludedTables,
			opts.GenerateProto,
		); err != nil {
			return nil, err
		}
	}

	return tables, nil
}

// generateOrmCode generates the ORM code based on the specified ORM type.
func (g *Generator) generateOrmCode(
	ctx context.Context,
	opts GeneratorOptions,
	serviceRootPath string,
) error {
	var err error

	log.Println("Generating ORM code...")

	var schemaPath string
	var daoPath string
	switch opts.OrmType {
	case "ent":
		schemaPath = path.Join(serviceRootPath, "/data/ent/schema")
	case "gorm":
		schemaPath = path.Join(serviceRootPath, "/data/gorm/schema")
		daoPath = path.Join(serviceRootPath, "/data/gorm/dao")
	}

	if err = sqlorm.Importer(
		ctx,
		opts.OrmType,
		&opts.Driver,
		&opts.Source,
		&schemaPath,
		&daoPath,
		opts.IncludedTables,
		opts.ExcludedTables,
	); err != nil {
		return err
	}

	log.Println("ORM code generation completed.")

	return nil
}

func (g *Generator) generateServerPackageCode(
	outputPath string,
	projectName string,
	serviceName string,
	servicePackageMap map[string]string,
	servers []string,
) error {
	for _, server := range servers {
		if err := g.WriteServerPackageCode(
			outputPath,
			projectName, server, serviceName,
			servicePackageMap,
		); err != nil {
			return err
		}
	}

	return g.WriteWireSetCode(outputPath, projectName, serviceName, "server", "Server", servers)
}

func (g *Generator) generateServicePackageCode(
	outputPath string,
	projectName, serviceName string,
	targetModuleName, sourceModuleName, moduleVersion string,
	userRepo, isGrpcService bool,
	tables sqlproto.TableDataArray,
	services []string,
) error {

	for _, table := range tables {
		if len(table.Fields) == 0 {
			continue
		}

		name := inflection.Singular(table.Name)

		if err := g.WriteServicePackageCode(
			outputPath,
			projectName, serviceName,
			name,
			targetModuleName, sourceModuleName, moduleVersion,
			userRepo, isGrpcService,
		); err != nil {
			return err
		}
	}

	return g.WriteWireSetCode(outputPath, projectName, serviceName, "service", "Service", services)
}

func (g *Generator) generateDataPackageCode(
	outputPath string,
	orm string,
	projectName string, serviceName string,
	moduleName, moduleVersion string,
	tables sqlproto.TableDataArray,
	services []string,
) error {
	if len(tables) == 0 {
		return nil
	}

	var dataFields []generators.DataField
	for _, table := range tables {
		if len(table.Fields) == 0 {
			continue
		}

		name := inflection.Singular(table.Name)

		dataFields = make([]generators.DataField, 0)
		for _, field := range table.Fields {
			if field.Type == "" {
				continue
			}

			dataField := generators.DataField{
				Name: field.Name,
				Type: field.Type,
				//Null:    field.Null,
				Comment: field.Comment,
			}
			dataFields = append(dataFields, dataField)
		}

		if err := g.WriteDataPackageCode(
			outputPath,
			orm,
			projectName, serviceName, name,
			moduleName, moduleVersion,
			dataFields,
		); err != nil {
			return err
		}
	}

	switch orm {
	case "ent":
		_ = g.WriteWireSetCode(outputPath, projectName, serviceName, "data", "Client", []string{"Ent"})
	case "gorm":
		_ = g.WriteWireSetCode(outputPath, projectName, serviceName, "data", "Client", []string{"Gorm"})
	}

	return g.WriteWireSetCode(outputPath, projectName, serviceName, "data", "Repo", services)
}

func (g *Generator) generateMainPackageCode(
	outputPath string,

	projectName string, serviceName string,

	servers []string,
) error {
	if err := g.WriteMainCode(
		outputPath,
		projectName, serviceName,
		servers,
	); err != nil {
		return err
	}

	return g.WriteWireCode(
		outputPath,
		projectName, serviceName,
	)
}

// generateConfigCode 生成配置文件 (client.yaml, server.yaml, logger.yaml, data.yaml)
func (g *Generator) generateConfigCode(ctx context.Context, configPath string, servers []string) error {
	log.Println("Generating config files...")

	if _, err := g.yamlGenerator.GenerateLoggerYaml(ctx, code_generator.Options{
		OutDir: configPath,
	}); err != nil {
		return err
	}

	if _, err := g.yamlGenerator.GenerateDataYaml(ctx, code_generator.Options{
		OutDir: configPath,
	}); err != nil {
		return err
	}

	if _, err := g.yamlGenerator.GenerateClientYaml(ctx, code_generator.Options{
		OutDir: configPath,
	}); err != nil {
		return err
	}

	if _, err := g.yamlGenerator.GenerateServerYaml(ctx, code_generator.Options{
		OutDir: configPath,
	}); err != nil {
		return err
	}

	log.Println("Config files generation completed.")
	return nil
}

// generateMakefileCode 生成 Makefile
func (g *Generator) generateMakefileCode(ctx context.Context, servicePath string) error {
	log.Println("Generating Makefile...")

	_, err := g.makefileGenerator.GenerateAppMakefile(ctx, code_generator.Options{
		OutDir: servicePath,
	})

	if err != nil {
		return err
	}

	log.Println("Makefile generation completed.")
	return nil
}
