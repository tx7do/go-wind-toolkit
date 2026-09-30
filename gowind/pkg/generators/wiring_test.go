package generators

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tx7do/go-utils/code_generator"
)

// 全新渲染与锚点注入必须产出逐字符一致的登记行——两条路径共用
// Build*WiringLine/BuildServer*Line 构造器,本测试把这一不变量钉死。
func TestFreshWiringMatchesBuilderLines_RepoForm(t *testing.T) {
	out := t.TempDir()
	gen := NewGoGenerator()

	serverOut := filepath.Join(out, "server")
	if err := os.MkdirAll(serverOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.GenerateGrpcServer(context.Background(), code_generator.Options{
		OutDir: serverOut,
		Module: "example.com/demo",
		Vars: map[string]any{
			"Service":  "core",
			"Packages": map[string]string{"user": "user"},
			"Services": map[string]string{"user": "user"},
		},
	}); err != nil {
		t.Fatalf("GenerateGrpcServer: %v", err)
	}

	blocks := BuildWiringBlocks(
		[]string{"grpc"}, true, "ent", []string{"ent"}, []string{"user"}, []string{"user"},
	)
	mainOut := filepath.Join(out, "main")
	if err := os.MkdirAll(mainOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.GenerateWiring(context.Background(), code_generator.Options{
		OutDir: mainOut,
		Module: "example.com/demo",
		Vars: map[string]any{
			"Service": "core",
		},
	}, blocks); err != nil {
		t.Fatalf("GenerateWiring: %v", err)
	}

	// server 文件:形参与路由行与构造器一致,锚点就位。
	serverSrc := readTestFile(t, filepath.Join(serverOut, "grpc_server.go"))
	paramLine, _ := BuildServerParamLine("user")
	if !strings.Contains(serverSrc, "\n"+paramLine+"\n") {
		t.Fatalf("fresh grpc_server.go missing param line %q:\n%s", paramLine, serverSrc)
	}
	routeLine, _ := BuildRouteLine("grpc", "user", "v1", "user")
	if !strings.Contains(serverSrc, "\n"+routeLine+"\n") {
		t.Fatalf("fresh grpc_server.go missing route line %q:\n%s", routeLine, serverSrc)
	}
	if !strings.Contains(serverSrc, AnchorParam) || !strings.Contains(serverSrc, AnchorRoute) {
		t.Fatal("fresh grpc_server.go missing register anchors")
	}

	// 装配文件:仓储/服务/实参行与构造器一致,基础设施小节含 ent 客户端构造与 cleanup 登记。
	wiringSrc := readTestFile(t, filepath.Join(mainOut, "wiring.go"))
	repoLine, _ := BuildRepoWiringLine("user", "entClient")
	if !strings.Contains(wiringSrc, "\n"+repoLine+"\n") {
		t.Fatalf("fresh wiring.go missing repo line %q:\n%s", repoLine, wiringSrc)
	}
	serviceLine, _ := BuildServiceWiringLine("user", false)
	if !strings.Contains(wiringSrc, "\n"+serviceLine+"\n") {
		t.Fatalf("fresh wiring.go missing service line:\n%s", wiringSrc)
	}
	argLine, _ := BuildServerArgLine("user")
	if !strings.Contains(wiringSrc, "\n"+argLine+"\n") {
		t.Fatalf("fresh wiring.go missing server arg line:\n%s", wiringSrc)
	}
	for _, mustHave := range []string{
		AnchorRepo, AnchorService, AnchorGrpcArg,
		"client.NewEntClient(ctx)",
		"cleanups = append(cleanups, cleanupEnt)",
		"rollback()",
		"return newApp(",
	} {
		if !strings.Contains(wiringSrc, mustHave) {
			t.Fatalf("fresh wiring.go missing %q:\n%s", mustHave, wiringSrc)
		}
	}
	if strings.Contains(wiringSrc, AnchorClient) || strings.Contains(wiringSrc, AnchorRestArg) {
		t.Fatal("repo-form wiring must not carry BFF anchors")
	}
	if !blocks.ImportData || !blocks.ImportClient || !blocks.ImportServer || !blocks.ImportServicePkg {
		t.Fatal("repo-form blocks must request all four imports")
	}
}

func TestFreshWiringMatchesBuilderLines_BffForm(t *testing.T) {
	out := t.TempDir()
	gen := NewGoGenerator()

	serverOut := filepath.Join(out, "server")
	if err := os.MkdirAll(serverOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.GenerateRestServer(context.Background(), code_generator.Options{
		OutDir: serverOut,
		Module: "example.com/demo",
		Vars: map[string]any{
			"Service":  "admin",
			"Packages": map[string]string{"article": "article"},
			"Services": map[string]string{"article": "article"},
		},
	}); err != nil {
		t.Fatalf("GenerateRestServer: %v", err)
	}

	// BFF:无 ORM 客户端、无仓储行;服务行数据源为服务客户端变量。
	blocks := BuildWiringBlocks(
		[]string{"rest"}, false, "", nil, nil, []string{"article"},
	)
	mainOut := filepath.Join(out, "main")
	if err := os.MkdirAll(mainOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.GenerateWiring(context.Background(), code_generator.Options{
		OutDir: mainOut,
		Module: "example.com/demo",
		Vars: map[string]any{
			"Service": "admin",
		},
	}, blocks); err != nil {
		t.Fatalf("GenerateWiring: %v", err)
	}

	serverSrc := readTestFile(t, filepath.Join(serverOut, "rest_server.go"))
	paramLine, _ := BuildServerParamLine("article")
	if !strings.Contains(serverSrc, "\n"+paramLine+"\n") {
		t.Fatalf("fresh rest_server.go missing param line:\n%s", serverSrc)
	}
	routeLine, _ := BuildRouteLine("rest", "admin", "v1", "article")
	if !strings.Contains(serverSrc, "\n"+routeLine+"\n") {
		t.Fatalf("fresh rest_server.go missing route line %q:\n%s", routeLine, serverSrc)
	}
	if !strings.Contains(serverSrc, AnchorParam) || !strings.Contains(serverSrc, AnchorRoute) {
		t.Fatal("fresh rest_server.go missing register anchors")
	}

	wiringSrc := readTestFile(t, filepath.Join(mainOut, "wiring.go"))
	serviceLine, _ := BuildServiceWiringLine("article", true)
	if !strings.Contains(wiringSrc, "\n"+serviceLine+"\n") {
		t.Fatalf("fresh wiring.go missing BFF service line:\n%s", wiringSrc)
	}
	argLine, _ := BuildServerArgLine("article")
	if !strings.Contains(wiringSrc, "\n"+argLine+"\n") {
		t.Fatalf("fresh wiring.go missing rest arg line:\n%s", wiringSrc)
	}
	for _, mustHave := range []string{AnchorClient, AnchorService, AnchorRestArg, "return newApp("} {
		if !strings.Contains(wiringSrc, mustHave) {
			t.Fatalf("BFF wiring missing %q:\n%s", mustHave, wiringSrc)
		}
	}
	for _, banned := range []string{
		AnchorRepo, AnchorGrpcArg, "client.NewEntClient", "client.NewGormClient", "client.NewRedisClient",
	} {
		if strings.Contains(wiringSrc, banned) {
			t.Fatalf("BFF wiring must not contain %q:\n%s", banned, wiringSrc)
		}
	}
	if blocks.ImportData || blocks.ImportClient || !blocks.ImportServer || !blocks.ImportServicePkg {
		t.Fatal("BFF blocks must request only server/service imports")
	}
}

// WebSocket 传输脚手架:生成消息驱动的 server 骨架,wiring 构造 wsServer 且不落入 TODO 占位分支。
func TestFreshWiringWebsocketForm(t *testing.T) {
	out := t.TempDir()
	gen := NewGoGenerator()

	serverOut := filepath.Join(out, "server")
	if err := os.MkdirAll(serverOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.GenerateWebsocketServer(context.Background(), code_generator.Options{
		OutDir: serverOut,
		Module: "example.com/demo",
		Vars:   map[string]any{"Service": "chat"},
	}); err != nil {
		t.Fatalf("GenerateWebsocketServer: %v", err)
	}

	serverSrc := readTestFile(t, filepath.Join(serverOut, "websocket_server.go"))
	for _, mustHave := range []string{
		"func NewWebsocketServer(",
		"func NewWebsocketMiddleware(",
		"github.com/tx7do/kratos-transport/transport/websocket",
		"cfg.Server.Websocket",
		AnchorRoute,
	} {
		if !strings.Contains(serverSrc, mustHave) {
			t.Fatalf("websocket_server.go missing %q:\n%s", mustHave, serverSrc)
		}
	}

	blocks := BuildWiringBlocks([]string{"websocket"}, false, "", nil, nil, nil)
	mainOut := filepath.Join(out, "main")
	if err := os.MkdirAll(mainOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.GenerateWiring(context.Background(), code_generator.Options{
		OutDir: mainOut,
		Module: "example.com/demo",
		Vars:   map[string]any{"Service": "chat"},
	}, blocks); err != nil {
		t.Fatalf("GenerateWiring: %v", err)
	}

	wiringSrc := readTestFile(t, filepath.Join(mainOut, "wiring.go"))
	for _, mustHave := range []string{
		"wsMiddlewares := server.NewWebsocketMiddleware(ctx)",
		"wsServer, err := server.NewWebsocketServer(ctx, wsMiddlewares)",
		"wsServer,",
		"return newApp(",
	} {
		if !strings.Contains(wiringSrc, mustHave) {
			t.Fatalf("websocket wiring missing %q:\n%s", mustHave, wiringSrc)
		}
	}
	// 关键不变量:websocket 不再落入未实现占位分支。
	if strings.Contains(wiringSrc, "TODO: 传输层") {
		t.Fatalf("websocket wiring must not emit unimplemented-transport placeholder:\n%s", wiringSrc)
	}
	if !blocks.ImportServer {
		t.Fatal("websocket blocks must request the server import")
	}
}

// 全新空脚手架(无模块、无客户端):分节只有锚点,文件可读且结构完整。
func TestFreshWiringEmptyScaffold(t *testing.T) {
	blocks := BuildWiringBlocks([]string{"grpc"}, false, "", nil, nil, nil)
	out := t.TempDir()
	gen := NewGoGenerator()
	if _, err := gen.GenerateWiring(context.Background(), code_generator.Options{
		OutDir: out,
		Module: "example.com/demo",
		Vars:   map[string]any{"Service": "core"},
	}, blocks); err != nil {
		t.Fatalf("GenerateWiring: %v", err)
	}
	wiringSrc := readTestFile(t, filepath.Join(out, "wiring.go"))
	for _, mustHave := range []string{
		"func initApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {",
		AnchorRepo, AnchorService, AnchorGrpcArg, "return newApp(",
	} {
		if !strings.Contains(wiringSrc, mustHave) {
			t.Fatalf("empty scaffold missing %q:\n%s", mustHave, wiringSrc)
		}
	}
	for _, banned := range []string{"data.New", "service.New", "client.New"} {
		if strings.Contains(wiringSrc, banned) {
			t.Fatalf("empty scaffold must not contain %q:\n%s", banned, wiringSrc)
		}
	}
}

// 头部注释按形态区分:BFF 注明网关语义与客户端登记约定。
func TestWiringHeaderCommentVariants(t *testing.T) {
	bffBlocks := BuildWiringBlocks([]string{"rest"}, false, "", nil, nil, nil)
	repoBlocks := BuildWiringBlocks([]string{"grpc"}, true, "ent", []string{"ent"}, nil, nil)

	if !strings.Contains(bffBlocks.HeaderComment, "BFF") {
		t.Fatal("BFF header comment missing BFF note")
	}
	if strings.Contains(repoBlocks.HeaderComment, "BFF") {
		t.Fatal("repo-form header comment must not carry BFF note")
	}
}
