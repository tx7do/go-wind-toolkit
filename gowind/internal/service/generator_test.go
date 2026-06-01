package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	err := Generate(context.Background(), GeneratorOptions{
		GenerateMain:     true,
		GenerateServer:   true,
		GenerateService:  true,
		GenerateData:     true,
		GenerateMakefile: true,
		GenerateConfigs:  true,

		ProjectModule: "github.com/gowind-example",
		ProjectName:   "gowind-example",
		ServiceName:   "user",

		Servers:   []string{"rest", "grpc"},
		DbClients: []string{"ent", "redis"},

		OutputPath: "./test",
	})
	assert.Nil(t, err)
}

func TestExtractProjectName(t *testing.T) {
	projectModule := "github.com/gowind-example"
	projectName := extractProjectName(projectModule)
	assert.Equal(t, "gowind-example", projectName)

	projectModule = "gowind-example"
	projectName = extractProjectName(projectModule)
	assert.Equal(t, "gowind-example", projectName)
}
