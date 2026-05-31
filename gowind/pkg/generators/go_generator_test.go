package generators

import (
	"testing"

	"github.com/tx7do/go-utils/code_generator"
)

func TestGoGenerator_Template_Main(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service":                  "user",
			"ServerImports":            []string{"github.com/example/myproject/server"},
			"ServerFormalParameters":   []string{"hs http.Server"},
			"ServerTransferParameters": []string{"hs"},
		},
	}

	if _, err := g.GenerateMain(t.Context(), opts); err != nil {
		t.Fatalf("Generate main.go failed: %v", err)
	}
}

func TestGoGenerator_Template_Wire(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service": "user",
		},
	}

	if _, err := g.GenerateWire(t.Context(), opts); err != nil {
		t.Fatalf("Generate wire.go failed: %v", err)
	}
}

func TestGoGenerator_Template_WireSet(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service":      "user",
			"Package":      "data",
			"NewFunctions": []string{"NewUserRepo", "NewOrderRepo"},
		},
	}

	if _, err := g.GenerateWireSet(t.Context(), opts); err != nil {
		t.Fatalf("Generate init.go failed: %v", err)
	}
}

func TestGoGenerator_Template_EntClient(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service": "user",
		},
	}

	if _, err := g.GenerateEntClient(t.Context(), opts); err != nil {
		t.Fatalf("Generate ent_client.go failed: %v", err)
	}
}

func TestGoGenerator_Template_EntRepo(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service":    "user",
			"ApiPackage": "userV1",
			"Model":      "user",
		},
	}

	if _, err := g.GenerateEntRepo(t.Context(), opts); err != nil {
		t.Fatalf("Generate ent_repo.go failed: %v", err)
	}
}

func TestGoGenerator_Template_GormClient(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service": "user",
		},
	}

	if _, err := g.GenerateGormClient(t.Context(), opts); err != nil {
		t.Fatalf("Generate gorm_client.go failed: %v", err)
	}
}

func TestGoGenerator_Template_GormInit(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir:     "./output",
		OutputName: "gorm_init.go",
		Module:     "github.com/example/myproject",
		Vars: map[string]any{
			"Service": "user",
		},
	}

	if _, err := g.GenerateGormInit(t.Context(), opts); err != nil {
		t.Fatalf("Generate gorm_init.go failed: %v", err)
	}
}

func TestGoGenerator_Template_GormRepo(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service":    "user",
			"ApiPackage": "userV1",
			"Model":      "user",
		},
	}

	if _, err := g.GenerateGormRepo(t.Context(), opts); err != nil {
		t.Fatalf("Generate gorm_repo.go failed: %v", err)
	}
}

func TestGoGenerator_Template_GrpcServer(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service":  "user",
			"Packages": []string{"user"},
			"Services": map[string]string{"user": "userV1", "role": "userV1"},
		},
	}

	if _, err := g.GenerateGrpcServer(t.Context(), opts); err != nil {
		t.Fatalf("Generate grpc_server.go failed: %v", err)
	}
}

func TestGoGenerator_Template_RedisClient(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service": "user",
		},
	}

	if _, err := g.GenerateRedisClient(t.Context(), opts); err != nil {
		t.Fatalf("Generate redis_client.go failed: %v", err)
	}
}

func TestGoGenerator_Template_RestServer(t *testing.T) {
	g := NewGoGenerator()

	opts := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"Service":  "admin",
			"Services": map[string]string{"user": "userV1", "role": "userV1"},
		},
	}

	if _, err := g.GenerateRestServer(t.Context(), opts); err != nil {
		t.Fatalf("Generate rest_server.go failed: %v", err)
	}
}

func TestGoGenerator_Template_Service(t *testing.T) {
	g := NewGoGenerator()

	opts1 := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"TargetApiPackageName":    "user",
			"TargetApiPackageVersion": "v1",

			"SourceApiPackageName":    "user",
			"SourceApiPackageVersion": "v1",

			"Service": "user",
			"Model":   "user",
			"IsGrpc":  true,
		},
	}

	if _, err := g.GenerateService(t.Context(), opts1); err != nil {
		t.Fatalf("Generate service.go failed: %v", err)
	}

	opts2 := code_generator.Options{
		OutDir: "./output",
		Module: "github.com/example/myproject",
		Vars: map[string]any{
			"TargetApiPackageName":    "admin",
			"TargetApiPackageVersion": "v1",

			"SourceApiPackageName":    "user",
			"SourceApiPackageVersion": "v1",

			"Service": "admin",
			"Model":   "role",
			"IsGrpc":  false,
		},
	}

	if _, err := g.GenerateService(t.Context(), opts2); err != nil {
		t.Fatalf("Generate service.go failed: %v", err)
	}
}
