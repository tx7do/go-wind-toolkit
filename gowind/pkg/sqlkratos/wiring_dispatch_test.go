package sqlkratos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

// 全新服务同时请求 grpc 与 websocket:grpc 逐表注册 proto 服务,websocket 作为
// 消息驱动传输仅产 server 骨架、不做逐表 proto 注册;二者都应无错误地落盘。
func TestGenerateServerPackageCodeGrpcAndWebsocket(t *testing.T) {
	root := t.TempDir()
	serverDir := filepath.Join(root, "app", "user", "service", "internal", "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 无既有装配文件 → useWiringDI 形态,但 server 文件不存在,走整体渲染。
	wctx := newWiringContext(root, "user", "ent", true, true)

	g := NewGenerator()
	servicePackageMap := map[string]string{"user": "user"}
	err := g.generateServerPackageCode(serverDir, "proj", "user", servicePackageMap, []string{"grpc", "websocket"}, "v1", []string{"user"}, wctx)
	if err != nil {
		t.Fatalf("generateServerPackageCode: %v", err)
	}

	grpcSrc := readNormalized(t, filepath.Join(serverDir, "grpc_server.go"))
	if !strings.Contains(grpcSrc, "RegisterUserServiceServer") {
		t.Fatalf("grpc_server.go missing per-table registration:\n%s", grpcSrc)
	}

	wsSrc := readNormalized(t, filepath.Join(serverDir, "websocket_server.go"))
	if !strings.Contains(wsSrc, "func NewWebsocketServer(") {
		t.Fatalf("websocket_server.go missing server constructor:\n%s", wsSrc)
	}
	if strings.Contains(wsSrc, "RegisterUserServiceServer") {
		t.Fatal("websocket_server.go must not register proto services")
	}
}

// 既有带锚点的 server 文件:只注入新模块形参/路由,不整体重渲——
// 项目手工维护的中间件与路由内容必须原样保留;装配文件同步注入实参行;全程幂等。
func TestGenerateServerPackageCodeInjectsIntoAnchoredFiles(t *testing.T) {
	root := t.TempDir()
	serverDir := filepath.Join(root, "app", "admin", "service", "internal", "server")
	cmdServerDir := filepath.Join(root, "app", "admin", "service", "cmd", "server")
	for _, d := range []string{serverDir, cmdServerDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// 手工维护形态的 rest_server.go:带锚点与人工内容。
	restFixture := "package server\n\nimport (\n\t\"github.com/tx7do/kratos-bootstrap/bootstrap\"\n)\n\n// HAND_MAINTAINED_MARKER\n\nfunc NewRestServer(\n\tctx *bootstrap.Context,\n\n\tmiddlewares RestMiddlewares,\n\n\t" + generators.AnchorParam + "\n) (*http.Server, error) {\n\t" + generators.AnchorRoute + "\n\treturn nil, nil\n}\n"
	restFile := filepath.Join(serverDir, "rest_server.go")
	if err := os.WriteFile(restFile, []byte(restFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	// 手工维护形态的 wiring.go:带各登记锚点与 ORM 客户端构造(供形态探测)。
	wiringFixture := "package main\n\nimport (\n\t\"github.com/tx7do/kratos-bootstrap/bootstrap\"\n)\n\nfunc initApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {\n\tentClient, cleanupEnt, err := client.NewEntClient(ctx)\n\t" + generators.AnchorRepo + "\n\t" + generators.AnchorService + "\n\t" + generators.AnchorRestArg + "\n\treturn nil, nil, nil\n}\n"
	wiringFile := filepath.Join(cmdServerDir, "wiring.go")
	if err := os.WriteFile(wiringFile, []byte(wiringFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	// 形态探测:应识别为手写装配(带锚点 wiring.go 存在)。
	wctx := newWiringContext(root, "admin", "ent", false, true)
	if !wctx.useWiringDI {
		t.Fatal("expected wiring mode for anchored wiring.go")
	}
	if wctx.wiringFile != wiringFile {
		t.Fatalf("expected wiring file %q, got %q", wiringFile, wctx.wiringFile)
	}
	if wctx.ormClientVar != "entClient" {
		t.Fatalf("expected detected orm client var entClient, got %q", wctx.ormClientVar)
	}

	g := NewGenerator()
	servicePackageMap := map[string]string{"widget": "admin"}
	err := g.generateServerPackageCode(serverDir, "proj", "admin", servicePackageMap, []string{"rest", "grpc"}, "v1", []string{"widget"}, wctx)
	if err != nil {
		t.Fatalf("generateServerPackageCode: %v", err)
	}

	// rest_server.go:注入了新模块形参与路由,人工内容保留。
	restSrc := readNormalized(t, restFile)
	paramLine, _ := generators.BuildServerParamLine("widget")
	if !strings.Contains(restSrc, "\n"+paramLine+"\n") {
		t.Fatalf("rest_server.go missing injected param line:\n%s", restSrc)
	}
	routeLine, _ := generators.BuildRouteLine("rest", "admin", "v1", "widget")
	if !strings.Contains(restSrc, "\n"+routeLine+"\n") {
		t.Fatalf("rest_server.go missing injected route line:\n%s", restSrc)
	}
	if !strings.Contains(restSrc, "HAND_MAINTAINED_MARKER") {
		t.Fatal("hand-maintained rest_server.go content was clobbered by re-render")
	}
	if !strings.Contains(restSrc, "return nil, nil") {
		t.Fatal("hand-maintained rest_server.go body was clobbered")
	}

	// 装配文件:注入了服务实参行,其余内容保留。
	wiringSrc := readNormalized(t, wiringFile)
	argLine, _ := generators.BuildServerArgLine("widget")
	if !strings.Contains(wiringSrc, "\n"+argLine+"\n") {
		t.Fatalf("wiring.go missing injected server arg line:\n%s", wiringSrc)
	}
	if !strings.Contains(wiringSrc, "return nil, nil, nil") {
		t.Fatal("hand-maintained wiring.go body was clobbered")
	}

	// grpc_server.go 不存在:全新渲染(带锚点与该模块的形参/路由)。
	grpcFile := filepath.Join(serverDir, "grpc_server.go")
	grpcSrc := readNormalized(t, grpcFile)
	if !strings.Contains(grpcSrc, generators.AnchorParam) || !strings.Contains(grpcSrc, generators.AnchorRoute) {
		t.Fatal("fresh grpc_server.go missing anchors")
	}
	grpcParamLine, _ := generators.BuildServerParamLine("widget")
	if !strings.Contains(grpcSrc, "\n"+grpcParamLine+"\n") {
		t.Fatalf("fresh grpc_server.go missing param line:\n%s", grpcSrc)
	}
	grpcRouteLine, _ := generators.BuildRouteLine("grpc", "admin", "v1", "widget")
	if !strings.Contains(grpcSrc, "\n"+grpcRouteLine+"\n") {
		t.Fatalf("fresh grpc_server.go missing route line:\n%s", grpcSrc)
	}

	// 幂等:重复执行不改内容。
	beforeRest := readNormalized(t, restFile)
	beforeWiring := readNormalized(t, wiringFile)
	beforeGrpc := readNormalized(t, grpcFile)
	if err := g.generateServerPackageCode(serverDir, "proj", "admin", servicePackageMap, []string{"rest", "grpc"}, "v1", []string{"widget"}, wctx); err != nil {
		t.Fatalf("second generateServerPackageCode: %v", err)
	}
	if got := readNormalized(t, restFile); got != beforeRest {
		t.Fatal("second run mutated rest_server.go")
	}
	if got := readNormalized(t, wiringFile); got != beforeWiring {
		t.Fatal("second run mutated wiring.go")
	}
	if got := readNormalized(t, grpcFile); got != beforeGrpc {
		t.Fatal("second run mutated grpc_server.go")
	}
}

// 无 wiring 锚点、有 wire provider 集:沿用旧式 wire 形态(server 重渲 + provider 集更新)。
func TestGenerateServerPackageCodeWireMode(t *testing.T) {
	root := t.TempDir()
	serverDir := filepath.Join(root, "app", "user", "service", "internal", "server")
	providersDir := filepath.Join(serverDir, "providers")
	for _, d := range []string{providersDir, filepath.Join(root, "app", "user", "service", "cmd", "server")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// 旧式 provider 集(存在即 wire 形态;含真实文件形态的 import 块)。
	wireSet := "package providers\n\nimport (\n\t\"github.com/google/wire\"\n)\n\nvar ProviderSet = wire.NewSet(\n)\n"
	if err := os.WriteFile(filepath.Join(providersDir, "wire_set.go"), []byte(wireSet), 0o644); err != nil {
		t.Fatal(err)
	}

	wctx := newWiringContext(root, "user", "ent", true, true)
	if wctx.useWiringDI {
		t.Fatal("expected wire mode for provider-set service")
	}
	if wctx.wiringFile != "" {
		t.Fatal("wire mode must not carry a wiring file")
	}

	g := NewGenerator()
	servicePackageMap := map[string]string{"widget": "user"}
	if err := g.generateServerPackageCode(serverDir, "proj", "user", servicePackageMap, []string{"grpc"}, "v1", []string{"widget"}, wctx); err != nil {
		t.Fatalf("generateServerPackageCode: %v", err)
	}

	// 旧式路径:server 全新渲染,provider 集追加 NewGrpcServer/NewGrpcMiddleware 条目。
	grpcSrc := readNormalized(t, filepath.Join(serverDir, "grpc_server.go"))
	if !strings.Contains(grpcSrc, generators.AnchorParam) {
		t.Fatal("fresh render should still carry anchors")
	}
	setSrc := readNormalized(t, filepath.Join(providersDir, "wire_set.go"))
	for _, fn := range []string{"server.NewGrpcServer", "server.NewGrpcMiddleware"} {
		if !strings.Contains(setSrc, fn) {
			t.Fatalf("provider set missing %q:\n%s", fn, setSrc)
		}
	}
}

func readNormalized(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}
