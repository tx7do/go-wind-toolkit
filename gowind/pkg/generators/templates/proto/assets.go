package proto

import _ "embed"

var TemplateMap = map[string][]byte{
	"grpc_proto.tpl": GrpcProtoTemplateData,
	"rest_proto.tpl": RestProtoTemplateData,
}

//go:embed grpc_proto.tpl
var GrpcProtoTemplateData []byte

//go:embed rest_proto.tpl
var RestProtoTemplateData []byte
