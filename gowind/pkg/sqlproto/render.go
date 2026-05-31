package sqlproto

import (
	"errors"
	"log"
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
	switch strings.TrimSpace(strings.ToLower(serviceType)) {
	case "grpc":
		data := render.GrpcProtoTemplateData{
			Module:  targetModuleName,
			Version: moduleVersion,

			Name:    inflection.Singular(tableName),
			Comment: RemoveTableCommentSuffix(tableComment),
			Fields:  render.ProtoFieldArray(protoFields),
		}
		return render.WriteGrpcServiceProto(outputPath, data)

	case "rest":
		data := render.RestProtoTemplateData{
			SourceModule: sourceModuleName,
			TargetModule: targetModuleName,
			Version:      moduleVersion,

			Name:    inflection.Singular(tableName),
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
			log.Fatal(err)
			return err
		}
	}

	return nil
}

func RemoveTableCommentSuffix(input string) string {
	re := regexp.MustCompile(`(表|table)$`)
	return re.ReplaceAllString(input, "")
}
