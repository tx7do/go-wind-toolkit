package generators

import (
	"context"
	"testing"

	"github.com/tx7do/go-utils/code_generator"
)

func TestMakefileGenerator_Template_AppMakefile(t *testing.T) {
	g := NewMakefileGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
	}

	if _, err := g.GenerateAppMakefile(context.Background(), opts); err != nil {
		t.Fatalf("Generate Makefile failed: %v", err)
	}
}
