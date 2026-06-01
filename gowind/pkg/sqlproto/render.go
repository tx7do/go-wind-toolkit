package sqlproto

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jinzhu/inflection"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto/internal/render"
)

type ProtoField generators.ProtoField
type ProtoFieldArray []generators.ProtoField

func WriteServiceProto(
	outputPath string,
	serviceType string,
	strategy string,
	targetModuleName, sourceModuleName, moduleVersion string,
	tableName string,
	tableComment string,
	protoFields ProtoFieldArray,
	customPkg string,
) error {
	modelName := inflection.Singular(tableName)

	// 根据 strategy 决定 gRPC proto module 名称（REST 的 SourceModule 也必须用同一个）
	var protoModule string
	switch strategy {
	case "by-service":
		// 按服务分包：所有表共用服务名作为 proto module
		protoModule = strings.ToLower(sourceModuleName)
	case "custom":
		// 自定义：使用调用方传入的自定义包名，未指定则回退到表名单数
		if customPkg != "" {
			protoModule = strings.ToLower(customPkg)
		} else {
			protoModule = strings.ToLower(modelName)
		}
	default:
		// per-table（默认）：每表独立包，用表名单数
		protoModule = strings.ToLower(modelName)
	}

	switch strings.TrimSpace(strings.ToLower(serviceType)) {
	case "grpc":
		data := render.GrpcProtoTemplateData{
			Module:  protoModule,
			Version: moduleVersion,

			Name:    modelName,
			Comment: RemoveTableCommentSuffix(tableComment),
			Fields:  render.ProtoFieldArray(protoFields),
		}
		return render.WriteGrpcServiceProto(outputPath, data)

	case "rest":
		data := render.RestProtoTemplateData{
			SourceModule: protoModule,
			TargetModule: targetModuleName,
			Version:      moduleVersion,

			Name:    modelName,
			Comment: RemoveTableCommentSuffix(tableComment),
		}
		return render.WriteRestServiceProto(outputPath, data)

	default:
		return errors.New("sqlproto: unsupported service type: " + serviceType)
	}
}

func WriteServicesProto(
	outputPath string,
	serviceType string,
	strategy string,
	targetModuleName, sourceModuleName, moduleVersion string,
	tables TableDataArray,
	customPackages map[string]string,
) error {
	var protoFields ProtoFieldArray

	for i := 0; i < len(tables); i++ {
		table := tables[i]

		protoFields = make(ProtoFieldArray, 0, len(table.Fields))
		for n := 0; n < len(table.Fields); n++ {
			field := table.Fields[n]
			protoFields = append(protoFields, generators.ProtoField{
				Number:  n + 1,
				Name:    field.Name,
				Comment: field.Comment,
				Type:    field.Type,
			})
		}

		// 获取该表的自定义包名
		var customPkg string
		if customPackages != nil {
			customPkg = customPackages[table.Name]
		}

		if err := WriteServiceProto(
			outputPath,
			serviceType,
			strategy,
			targetModuleName, sourceModuleName, moduleVersion,
			table.Name, table.Comment,
			protoFields,
			customPkg,
		); err != nil {
			return fmt.Errorf("failed to write proto for table %s: %w", table.Name, err)
		}
	}

	return nil
}

func RemoveTableCommentSuffix(input string) string {
	re := regexp.MustCompile(`(表|table)$`)
	return re.ReplaceAllString(input, "")
}
