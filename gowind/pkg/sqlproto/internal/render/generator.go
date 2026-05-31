package render

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/tx7do/go-utils/code_generator"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

// WriteGrpcServiceProto write gRPC service proto file
func WriteGrpcServiceProto(outputPath string, data GrpcProtoTemplateData) error {
	outputPath = filepath.Join(outputPath, strings.ToLower(data.Module), "service", strings.ToLower(data.Version))
	outputPath = filepath.Clean(outputPath)

	//log.Printf("Writing gRPC service proto to: [%s] [%s] [%s] [%s]",
	//	outputPath, data.Module, data.Name, data.Comment)

	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
		return err
	}

	g := generators.NewProtoGenerator()

	opts := code_generator.Options{
		OutDir: outputPath,
		Vars: map[string]any{
			"Package":   data.Package(),
			"Model":     data.Name,
			"ModelName": data.Comment,
			"Fields":    data.Fields,
		},
	}

	_, err := g.GenerateGrpcServiceProto(context.Background(), opts)

	return err
}

// WriteRestServiceProto write REST service proto file
func WriteRestServiceProto(outputPath string, data RestProtoTemplateData) error {
	outputPath = filepath.Join(outputPath, strings.ToLower(data.TargetModule), "service", strings.ToLower(data.Version))
	outputPath = filepath.Clean(outputPath)

	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
		return err
	}

	g := generators.NewProtoGenerator()

	opts := code_generator.Options{
		OutDir: outputPath,
		Vars: map[string]any{
			"TargetPackage": data.TargetPackage(),
			"SourcePackage": data.SourcePackage(),
			"SourceProto":   data.SourceProto(),
			"ModelName":     data.Comment,
			"Path":          data.Path(),
			"Model":         data.Name,
		},
	}

	_, err := g.GenerateRestServiceProto(context.Background(), opts)

	return err
}
