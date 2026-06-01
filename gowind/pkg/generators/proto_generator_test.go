package generators

import (
	"context"
	"testing"

	"github.com/tx7do/go-utils/code_generator"
)

func TestProtoGenerator_Template_GrpcServiceProto(t *testing.T) {
	g := NewProtoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Vars: map[string]any{
			"Package":   "user.service.v1",
			"Model":     "user",
			"ModelName": "用户",
			"Fields": []ProtoField{
				{Name: "id", Type: "int64", Comment: "用户ID", Number: 1},
				{Name: "name", Type: "string", Comment: "用户名", Number: 2},
				{Name: "email", Type: "string", Comment: "用户邮箱", Number: 3},
			},
		},
	}

	if _, err := g.GenerateGrpcServiceProto(context.Background(), opts); err != nil {
		t.Fatalf("Generate grpc_proto.go failed: %v", err)
	}
}

func TestProtoGenerator_Template_RestServiceProto(t *testing.T) {
	g := NewProtoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Vars: map[string]any{
			"TargetPackage": "admin.service.v1",
			"SourcePackage": "user.service.v1",
			"SourceProto":   "user/service/v1/user.proto",
			"Model":         "user",
			"Path":          "/admin/v1/users",
			"ModelName":     "用户",
		},
	}

	if _, err := g.GenerateRestServiceProto(context.Background(), opts); err != nil {
		t.Fatalf("Generate rest_proto.go failed: %v", err)
	}
}
