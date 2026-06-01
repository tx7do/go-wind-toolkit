package sqlkratos

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/jinzhu/inflection"
	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"

	sqlorm "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlorm"
	sqlproto "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto"
)

// ensureDSNScheme ensures the DSN has a valid scheme prefix based on the driver type.
// If the DSN already contains "://", it is returned as-is.
// For PostgreSQL key-value format DSN (e.g. "host=localhost port=5432 user=postgres ..."),
// it converts to URL format (e.g. "postgres://user:pass@host:port/dbname?sslmode=disable").
func ensureDSNScheme(dsn, driver string) string {
	if strings.Contains(dsn, "://") {
		return dsn
	}
	switch strings.ToLower(driver) {
	case "mysql":
		return "mysql://" + dsn
	case "postgresql", "postgres":
		// PostgreSQL key-value DSN: "host=localhost port=5432 user=postgres password=xxx dbname=mydb sslmode=disable"
		if isPostgresKeyValueDSN(dsn) {
			return convertPostgresKeyValueToURL(dsn)
		}
		return "postgres://" + dsn
	default:
		return dsn
	}
}

// isPostgresKeyValueDSN detects PostgreSQL key-value format DSN.
// Key-value DSN contains space-separated key=value pairs like "host=localhost port=5432 user=postgres".
func isPostgresKeyValueDSN(dsn string) bool {
	return strings.Contains(dsn, "=") && strings.Contains(dsn, " ")
}

// convertPostgresKeyValueToURL converts PostgreSQL key-value DSN to URL format.
// Input:  "host=localhost port=5432 user=postgres password=xxx dbname=mydb sslmode=disable"
// Output: "postgres://postgres:xxx@localhost:5432/mydb?sslmode=disable"
func convertPostgresKeyValueToURL(dsn string) string {
	parts := strings.Fields(dsn)
	vals := make(map[string]string)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			vals[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	host := vals["host"]
	if host == "" {
		host = "localhost"
	}
	port := vals["port"]
	if port == "" {
		port = "5432"
	}
	user := vals["user"]
	password := vals["password"]
	dbname := vals["dbname"]

	// Build URL: postgres://user:password@host:port/dbname?params
	result := "postgres://"
	if user != "" {
		result += url.QueryEscape(user)
		if password != "" {
			result += ":" + url.QueryEscape(password)
		}
		result += "@"
	}
	result += host + ":" + port
	if dbname != "" {
		result += "/" + url.QueryEscape(dbname)
	}

	// Collect remaining params as query string
	var params []string
	for k, v := range vals {
		switch k {
		case "host", "port", "user", "password", "dbname":
			// already handled
		default:
			params = append(params, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	if len(params) > 0 {
		result += "?" + strings.Join(params, "&")
	}

	return result
}

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
		// 每表独立包：proto module 为表名单数，grpc_server/rest_server 使用此包名做 import
		servicePackageMap[name] = strings.ToLower(name)
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
			tables,
			services,
			opts.ModuleVersion,
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
			opts.SourceModuleName, opts.ModuleVersion,
			opts.UseRepo, useGrpc,
			tables,
			services,
			servicePackageMap,
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

	// 确保 DSN 有正确的 scheme 前缀
	source := ensureDSNScheme(opts.Source, opts.Driver)

	for _, server := range opts.Servers {
		if server != "grpc" && server != "rest" {
			continue
		}

		if tables, err = sqlproto.Convert(
			ctx,
			&source,
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

	// 确保 DSN 有正确的 scheme 前缀
	source := ensureDSNScheme(opts.Source, opts.Driver)

	var schemaPath string
	var daoPath string
	switch opts.OrmType {
	case "ent":
		schemaPath = path.Join(serviceRootPath, "/data/ent/schema")
	case "gorm":
		schemaPath = path.Join(serviceRootPath, "/data/gorm/models")
		daoPath = path.Join(serviceRootPath, "/data/gorm/dao")
	}

	if err = sqlorm.Importer(
		ctx,
		opts.OrmType,
		&opts.Driver,
		&source,
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
	sourceModuleName, moduleVersion string,
	userRepo, isGrpcService bool,
	tables sqlproto.TableDataArray,
	services []string,
	servicePackageMap map[string]string,
) error {

	for _, table := range tables {
		if len(table.Fields) == 0 {
			continue
		}

		name := inflection.Singular(table.Name)
		// 每表独立包：targetModule 为表名单数
		targetModule := strings.ToLower(name)

		if err := g.WriteServicePackageCode(
			outputPath,
			projectName, serviceName,
			name,
			targetModule, sourceModuleName, moduleVersion,
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
	tables sqlproto.TableDataArray,
	services []string,
	moduleVersion string,
) error {
	if len(tables) == 0 {
		return nil
	}

	// 先生成 client 代码（只生成一次）
	switch orm {
	case "ent":
		if err := g.writeEntClientCode(outputPath, projectName, serviceName); err != nil {
			return err
		}
	case "gorm":
		if err := g.writeGormClientCode(outputPath, projectName, serviceName); err != nil {
			return err
		}
	}

	// 收集所有模型名
	var modelNames []string

	var dataFields []generators.DataField
	for _, table := range tables {
		if len(table.Fields) == 0 {
			continue
		}

		name := inflection.Singular(table.Name)
		modelNames = append(modelNames, name)

		// 每表独立包：proto module 为表名单数
		moduleName := strings.ToLower(name)

		dataFields = make([]generators.DataField, 0)
		for _, field := range table.Fields {
			if field.Type == "" {
				continue
			}

			dataField := generators.DataField{
				Name:         field.Name,
				Type:         field.Type,
				SqlType:      field.SqlType,
				Comment:      field.Comment,
				Null:         field.Null,
				IsPrimaryKey: field.IsPrimaryKey,
			}
			dataFields = append(dataFields, dataField)
		}

		// 生成 repo 代码
		switch orm {
		case "ent":
			if err := g.writeEntRepoCode(outputPath, projectName, serviceName, name, moduleName, moduleVersion, dataFields); err != nil {
				return err
			}
		case "gorm":
			if err := g.writeGormRepoCode(outputPath, projectName, serviceName, name, moduleName, moduleVersion, dataFields); err != nil {
				return err
			}
		}
	}

	// gorm_init 在所有模型收集完后生成
	if orm == "gorm" {
		if err := g.writeGormInitCode(outputPath, projectName, serviceName, modelNames); err != nil {
			return err
		}
	}

	// 生成 data 层 wire_set（client 用 client. 前缀，repo 用 data. 前缀）
	var clientFunctions []string
	switch orm {
	case "ent":
		clientFunctions = append(clientFunctions, "client.NewEntClient")
	case "gorm":
		clientFunctions = append(clientFunctions, "client.NewGormClient")
	}

	// 合并 client 和 repo 函数到一个 wire_set
	var allFunctions []string
	allFunctions = append(allFunctions, clientFunctions...)
	for _, svc := range services {
		allFunctions = append(allFunctions, fmt.Sprintf("data.New%sRepo", stringcase.UpperCamelCase(svc)))
	}

	return g.WriteDataWireSetCode(outputPath, projectName, serviceName, allFunctions)
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
