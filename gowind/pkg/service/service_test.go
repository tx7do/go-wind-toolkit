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

func TestHasBFFService(t *testing.T) {
	tests := []struct {
		name     string
		servers  []string
		expected bool
	}{
		{
			name:     "grpc only",
			servers:  []string{"grpc"},
			expected: false,
		},
		{
			name:     "rest only",
			servers:  []string{"rest"},
			expected: true,
		},
		{
			name:     "grpc and rest",
			servers:  []string{"grpc", "rest"},
			expected: true,
		},
		{
			name:     "empty",
			servers:  []string{},
			expected: false,
		},
		{
			name:     "grpc rest uppercase",
			servers:  []string{"gRPC", "REST"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := GeneratorOptions{Servers: tt.servers}
			assert.Equal(t, tt.expected, opts.HasBFFService())
		})
	}
}

func TestAppendServiceName(t *testing.T) {
	g := NewGenerator()

	err := g.appendServiceName("./test", "test", "user", false)
	assert.Nil(t, err)

	err = g.appendServiceName("./test", "test", "order", false)
	assert.Nil(t, err)

	err = g.appendServiceName("./test", "test", "admin", true)
	assert.Nil(t, err)

	err = g.appendServiceName("./test", "test", "front", true)
	assert.Nil(t, err)
}

func TestWriteMakefile(t *testing.T) {
	g := NewGenerator()

	err := g.writeMakefile("./test")
	assert.Nil(t, err)
}

func TestWriteConfigs(t *testing.T) {
	g := NewGenerator()

	err := g.writeConfigs("./test/configs")
	assert.Nil(t, err)
}
