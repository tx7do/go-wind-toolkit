package render

import (
	"strings"

	"github.com/jinzhu/inflection"
	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

type ProtoFieldArray []generators.ProtoField

type GrpcProtoTemplateData struct {
	Name    string // Proto文件名
	Comment string // Proto文件注释
	Version string // Proto版本号

	Module string // 模块名

	Fields ProtoFieldArray // 字段列表
}

func (d GrpcProtoTemplateData) PascalName() string {
	return stringcase.ToPascalCase(d.Name)
}
func (d GrpcProtoTemplateData) SnakeName() string {
	return stringcase.SnakeCase(d.Name)
}

func (d GrpcProtoTemplateData) Package() string {
	return strings.ToLower(d.Module) + ".service." + d.Version
}

type RestProtoTemplateData struct {
	Name    string
	Comment string
	Version string

	SourceModule string
	TargetModule string
}

func (d RestProtoTemplateData) PascalName() string {
	return stringcase.ToPascalCase(d.Name)
}

func (d RestProtoTemplateData) Path() string {
	// 比如：/admin/v1/users
	return "/" + strings.ToLower(d.TargetModule) + "/" + d.Version + "/" + stringcase.KebabCase(inflection.Plural(d.Name))
}

func (d RestProtoTemplateData) SourceProto() string {
	// 比如：user/service/v1/user.proto
	return strings.ToLower(d.SourceModule) + "/service/" + d.Version + "/" + strings.ToLower(d.Name) + ".proto"
}

func (d RestProtoTemplateData) SourcePackage() string {
	// 比如：user.service.v1
	return strings.ToLower(d.SourceModule) + ".service." + d.Version
}

func (d RestProtoTemplateData) TargetPackage() string {
	return strings.ToLower(d.TargetModule) + ".service." + d.Version
}
