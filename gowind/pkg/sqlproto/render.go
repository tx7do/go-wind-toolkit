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
	targetModuleName, sourceModuleName, moduleVersion string,
	tableName string,
	tableComment string,
	protoFields ProtoFieldArray,
) error {
	modelName := inflection.Singular(tableName)

	switch strings.TrimSpace(strings.ToLower(serviceType)) {
	case "grpc":
		// 每张表使用自己的模型名作为 proto module，生成独立的 proto package
		// 例如表 sys_users → module=user → package=user.service.v1
		protoModule := strings.ToLower(modelName)
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
			SourceModule: sourceModuleName,
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
	targetModuleName, sourceModuleName, moduleVersion string,
	tables TableDataArray,
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

		//log.Printf("Generating proto for table: [%s] [%s]", table.Name, table.Comment)

		if err := WriteServiceProto(
			outputPath,
			serviceType,
			targetModuleName, sourceModuleName, moduleVersion,
			table.Name, table.Comment,
			protoFields,
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
