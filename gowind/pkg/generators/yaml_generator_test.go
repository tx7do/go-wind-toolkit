package generators

import (
	"context"
	"testing"

	"github.com/tx7do/go-utils/code_generator"
)

func TestYamlGenerator_Template_ServerYaml(t *testing.T) {
	g := NewYamlGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
	}

	if _, err := g.GenerateServerYaml(context.Background(), opts); err != nil {
		t.Fatalf("Generate server.yaml failed: %v", err)
	}
}

func TestYamlGenerator_Template_ClientYaml(t *testing.T) {
	g := NewYamlGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
	}

	if _, err := g.GenerateClientYaml(context.Background(), opts); err != nil {
		t.Fatalf("Generate client.yaml failed: %v", err)
	}
}

func TestYamlGenerator_Template_LoggerYaml(t *testing.T) {
	g := NewYamlGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
	}

	if _, err := g.GenerateLoggerYaml(context.Background(), opts); err != nil {
		t.Fatalf("Generate logger.yaml failed: %v", err)
	}
}

func TestYamlGenerator_Template_DataYaml(t *testing.T) {
	g := NewYamlGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
	}

	if _, err := g.GenerateDataYaml(context.Background(), opts); err != nil {
		t.Fatalf("Generate data.yaml failed: %v", err)
	}
}
