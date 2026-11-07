package sqlkratos_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlkratos"
)

// TestCompileE2E_GeneratedServiceCompilesAndTestsPass 是全链路编译门禁:
// 把 sqlkratos 的完整产物(含 --tests 单测骨架)生成进 scratch Go 模块,
// 走与真实项目一致的后处理链(tidy → buf generate → ent generate),
// 然后 go build ./... 并 go test ./... ——生成的 fake 仓储单测必须真实通过。
//
// 它回答"生成器只保证长得对,不保证编译得过"的问题:service 消费方接口、
// 测试骨架、wiring 装配,任何模板漂移在这里现形。
//
// 依赖:buf 与 protoc 插件在 PATH;Go 模块缓存温热(离线也可)。buf 依赖
// 解析需要网络;不可用时可用 GOWIND_COMPILE_E2E_BUF_LOCK 指向一份现成的
// buf.lock(依赖集合一致即可),否则跳过。
// compileVariant 门禁变体:同一份 DDL,不同 传输层×ORM 组合的生成物都要
// 编译通过——server/service 模板在 bLogger 迁移这类横切改动时最容易只有
// 一条路径被人工验证。
type compileVariant struct {
	name        string
	servers     []string
	orm         string
	useRepo     bool // false = BFF(service 消费 proto client 接口,无 data 层)
	entCodegen  bool // 是否需要 ent client 代码生成(entgen 模块流程)
	extraPlugins []string
}

func TestCompileE2E_GeneratedServiceCompilesAndTestsPass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile e2e in -short mode")
	}
	if os.Getenv("GOWIND_SKIP_COMPILE_E2E") == "1" {
		t.Skip("GOWIND_SKIP_COMPILE_E2E=1")
	}
	if _, err := exec.LookPath("buf"); err != nil {
		t.Skip("buf not found in PATH; skipping compile e2e")
	}

	variants := []compileVariant{
		{name: "grpc-ent", servers: []string{"grpc"}, orm: "ent", useRepo: true, entCodegen: true, extraPlugins: []string{"protoc-gen-go-grpc"}},
		{name: "rest-ent", servers: []string{"grpc", "rest"}, orm: "ent", useRepo: true, entCodegen: true, extraPlugins: []string{"protoc-gen-go-grpc", "protoc-gen-go-http"}},
		{name: "rest-bff", servers: []string{"rest"}, orm: "ent", useRepo: false, entCodegen: false, extraPlugins: []string{"protoc-gen-go-grpc", "protoc-gen-go-http"}},
		{name: "grpc-gorm", servers: []string{"grpc"}, orm: "gorm", useRepo: true, entCodegen: false, extraPlugins: []string{"protoc-gen-go-grpc"}},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			runCompileVariant(t, v)
		})
	}
}

func runCompileVariant(t *testing.T, v compileVariant) {
	const (
		module  = "scratch.local"
		service = "core"
	)

	for _, plugin := range append([]string{"protoc-gen-go"}, v.extraPlugins...) {
		if _, err := exec.LookPath(plugin); err != nil {
			t.Skipf("%s not found in PATH; skipping %s variant", plugin, v.name)
		}
	}


	ctx := context.Background()
	// 失败时保留现场供排查(GOWIND_COMPILE_E2E_KEEP=1),默认自动清理。
	scratch, err := os.MkdirTemp("", "gowind-compile-e2e-")
	require.NoError(t, err)
	if os.Getenv("GOWIND_COMPILE_E2E_KEEP") == "1" {
		t.Logf("保留现场: %s", scratch)
	} else {
		t.Cleanup(func() { _ = os.RemoveAll(scratch) })
	}

	// go.mod 先行:ent 变体预置 entgo require(ent 代码生成经 `go run
	// entgo.io/ent/cmd/ent`,无 require 时在 tidy 之前跑不了)。
	gomod := "module " + module + "\n\ngo 1.24\n"
	if v.entCodegen {
		gomod += "\nrequire entgo.io/ent v0.14.5\n"
	}
	writeFile(t, filepath.Join(scratch, "go.mod"), gomod)

	ddl := `CREATE TABLE users (
	id bigint(20) NOT NULL AUTO_INCREMENT,
	name varchar(100) NOT NULL COMMENT '用户名',
	created_by bigint unsigned NOT NULL DEFAULT 0 COMMENT '创建人',
	updated_by bigint unsigned NOT NULL DEFAULT 0 COMMENT '更新人',
	score int(10) unsigned NOT NULL DEFAULT 0,
	is_active tinyint(1) NOT NULL DEFAULT 1,
	created_at datetime,
	PRIMARY KEY (id)
);

CREATE TABLE orders (
	id bigint(20) NOT NULL AUTO_INCREMENT,
	total decimal(10,2) NOT NULL,
	created_by bigint unsigned NOT NULL DEFAULT 0,
	updated_by bigint unsigned NOT NULL DEFAULT 0,
	PRIMARY KEY (id)
);`

	require.NoError(t, sqlkratos.Generate(ctx, sqlkratos.GeneratorOptions{
		Source:           ddl,
		OrmType:          v.orm,
		UseRepo:          v.useRepo,
		// proto/service/server/单测骨架是可独立编译的产物(骨架按 useRepo 分
		// fake 仓储/fake client 两形态);main/config/makefile 依赖模板仓的
		// pkg/serviceid 与手写装配,scratch 里没有,剔除。
		GenerateProto:    true,
		GenerateORM:      v.useRepo,
		GenerateData:     v.useRepo,
		GenerateService:  true,
		GenerateServer:   true,
		GenerateTests:    true,
		GenerateMain:     false,
		GenerateConfig:   false,
		GenerateMakefile: false,
		Servers:          v.servers,
		ProjectName:      module,
		ServiceName:      service,
		ModuleName:       service,
		ModuleVersion:    "v1",
		OutputPath:       scratch,
	}))

	// 生成的测试骨架必须在场(门禁主对象之一)。
	testFile := filepath.Join(scratch, "app", service, "service", "internal", "service", "user_service_test.go")
	assert.FileExists(t, testFile, "service 单测骨架应已生成")

	// 顺序必须是 buf → ent → tidy:新鲜产物没有 gen/go 与 ent client 时,
	// tidy 会把 scratch.local/<缺失包> 当外部模块去网上解析,必挂。
	// (真实项目能 tidy 先行,是因为模板仓的 gen/go 是提交在库里的。)
	protoDir := filepath.Join(scratch, "api", "protos")
	prepareBufModule(t, protoDir)
	bufGenerate(t, protoDir, module, v.extraPlugins...)

	if v.entCodegen {
	// ent 代码生成:entc/load 需要**完全健康**的模块图才能加载 schema 包,
	// 而 scratch 此刻缺 ent client/sum 链,加载必残缺——把 schema 拷进一次性
	// 临时模块(依赖只有 entgo,tidy 可完整解析),在那里生成,再把产物回拷。
	workDir, werr := os.MkdirTemp("", "gowind-entgen-")
	require.NoError(t, werr)
	// 临时模块名必须等于 scratch 里 ent 目录的最终导入路径——entc 用它
	// 生成本模块导入(entgen/migrate → <路径>/migrate),回拷后才能对上。
	entModule := module + "/app/" + service + "/service/internal/data/ent"
	if os.Getenv("GOWIND_COMPILE_E2E_KEEP") == "1" {
		t.Logf("保留 entgen 现场: %s", workDir)
	} else {
		t.Cleanup(func() { _ = os.RemoveAll(workDir) })
	}
	// x/tools 需 replace 到新版:entgo v0.14.x 锁定的 v0.30 解不动 go 1.27
	// 的导出数据(entc/load 报 "package without types"),v0.52+ 可用。
	// replace 不会被 tidy 降级(direct require 会被 tidy 清回锁定版本)。
	writeFile(t, filepath.Join(workDir, "go.mod"), "module "+entModule+"\n\ngo 1.24\n\n"+
		"require entgo.io/ent v0.14.6\n\n"+
		"replace golang.org/x/tools => golang.org/x/tools v0.52.0\n")
	entDir := filepath.Join(scratch, "app", service, "service", "internal", "data", "ent")
	require.NoError(t, copyTree(filepath.Join(entDir, "schema"), filepath.Join(workDir, "schema")),
		"拷贝 schema 到临时模块")

	tidyWork := pkg.NewGoCmdWithTimeout(workDir, 5*time.Minute)
	tidyWork.Env = append(tidyWork.Env, "GOWORK=off")
	tidyWork.Stdout = os.Stderr
	tidyWork.Stderr = os.Stderr
	// 顺序:tidy(补 schema 图的 sum)→ get(补 ent cmd 图的 sum;若放在
	// tidy 之后会被 tidy 重写掉包级 sum,go run 必挂)→ run。
	require.NoError(t, tidyWork.Run(ctx, "mod", "tidy"), "临时模块 tidy")
	require.NoError(t, tidyWork.Run(ctx, "get", "entgo.io/ent/cmd/ent@v0.14.6"), "拉取 ent cmd")

	// feature 集与 internal/ent 的 generateEnt 保持一致。
	gen := pkg.NewGoCmdWithTimeout(workDir, 5*time.Minute)
	gen.Env = append(gen.Env, "GOWORK=off")
	gen.Stdout = os.Stderr
	gen.Stderr = os.Stderr
	require.NoError(t, gen.Run(ctx,
		"run", "entgo.io/ent/cmd/ent", "generate",
		"--feature", "privacy",
		"--feature", "entql",
		"--feature", "sql/modifier",
		"--feature", "sql/upsert",
		"--feature", "sql/lock",
		"--feature", "sql/versioned-migration",
		"./schema",
	), "ent generate")

	// v0.14.6 把生成产物写在 CWD(即 schema 的父目录)根,排除脚手架文件后回拷。
	skip := map[string]bool{"go.mod": true, "go.sum": true, "schema": true}
	require.NoError(t, filepath.WalkDir(workDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(workDir, path)
		if rerr != nil {
			return rerr
		}
		if rel == "." {
			return nil
		}
		if skip[filepath.ToSlash(rel)] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(entDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(target, data, 0o644)
	}), "回拷生成的 ent client")
	}

	// ent client 就位后 tidy:data/server 对它的导入才有 sum 可依。
	require.NoError(t, pkg.GoModTidy(ctx, scratch), "go mod tidy (post-ent)")

	// 编译门禁。
	goCmd := func(args ...string) {
		t.Logf("go %v", args)
		cmd := pkg.NewGoCmdWithTimeout(scratch, 15*time.Minute)
		cmd.Env = append(cmd.Env, "GOWORK=off")
		cmd.Stdout = os.Stderr // 过程日志进测试输出,失败时可读
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run(ctx, args...), "go %v", args)
	}
	goCmd("build", "./...")
	// 终极断言:生成的单测骨架(fake 仓储)必须真实通过。
	goCmd("test", "./app/"+service+"/service/internal/service/")
}

// prepareBufModule 写入 scratch 的 buf.yaml 并解析 BSR 依赖。
// 依赖解析需要网络;失败时回落到 GOWIND_COMPILE_E2E_BUF_LOCK 指向的
// 现成 buf.lock(依赖集合一致即可,离线走模块缓存),再不行就跳过。
func prepareBufModule(t *testing.T, protoDir string) {
	t.Helper()

	writeFile(t, filepath.Join(protoDir, "buf.yaml"), `version: v2
modules:
  - path: .
deps:
  - buf.build/googleapis/googleapis
  - buf.build/gnostic/gnostic
  - buf.build/tx7do/pagination
`)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "buf", "dep", "update")
	cmd.Dir = protoDir
	if out, err := cmd.CombinedOutput(); err == nil {
		return
	} else {
		t.Logf("buf dep update 失败(离线?),尝试回落 seed lock: %s", truncateLog(out))
	}

	seed := os.Getenv("GOWIND_COMPILE_E2E_BUF_LOCK")
	if seed == "" {
		t.Skip("buf 依赖解析不可用且未提供 GOWIND_COMPILE_E2E_BUF_LOCK;跳过编译门禁")
	}
	data, err := os.ReadFile(seed)
	require.NoError(t, err, "读取 seed buf.lock")
	writeFile(t, filepath.Join(protoDir, "buf.lock"), string(data))
}

// bufGenerate 用临时模板生成 pb 代码:managed 注入 go_package 前缀,
// paths=source_relative 落到 api/gen/go/<包路径>,与 service 模板的导入一致。
// extraPlugins 随传输层变化:grpc 追加 go-grpc,rest 追加 go-http。
func bufGenerate(t *testing.T, protoDir, module string, extraPlugins ...string) {
	t.Helper()

	tmpDir := t.TempDir()
	tmpl := filepath.Join(tmpDir, "buf.gen.yaml")
	out := filepath.Join(protoDir, "..", "gen", "go")
	writeFile(t, tmpl, compileBufTemplate(module, filepath.ToSlash(out), extraPlugins...))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "buf", "generate", "--template", tmpl)
	cmd.Dir = protoDir
	if out2, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("buf generate: %v\n%s", err, truncateLog(out2))
	}

	// 环境补齐:REST 产物的 assets.go embed openapi.yaml——真实项目该文件
	// 由 openapi 管线(gow openapi)产出,scratch 里写最小占位让 embed 通过。
	if serversHasRest(extraPlugins) {
		assetsDir := filepath.Join(protoDir, "..", "..",
			"app", "core", "service", "cmd", "server", "assets")
		_ = os.MkdirAll(assetsDir, 0o755)
		_ = os.WriteFile(filepath.Join(assetsDir, "openapi.yaml"),
			[]byte("openapi: 3.0.3\ninfo:\n  title: placeholder\n  version: \"0.0.1\"\npaths: {}\n"), 0o644)
	}

	// 环境补齐:REST service 引用模板仓的 pkg/middleware/auth(写操作人),
	// 写最小同形桩(真实项目由脚手架模板提供)。
	if serversHasRest(extraPlugins) {
		root := filepath.Join(out, "..", "..", "..")
		authDir := filepath.Join(root, "pkg", "middleware", "auth")
		_ = os.MkdirAll(authDir, 0o755)
		_ = os.WriteFile(filepath.Join(authDir, "auth.go"), []byte(
			"package auth\n\nimport \"context\"\n\n"+
				"type UserTokenPayload struct{ UserId uint64 }\n\n"+
				"func (p *UserTokenPayload) GetUserId() uint64 { if p == nil { return 0 }; return p.UserId }\n\n"+
				"func FromContext(_ context.Context) (*UserTokenPayload, error) { return &UserTokenPayload{}, nil }\n"), 0o644)
	}

	// 环境补齐:repo 模板调用包级 ErrorXxx 助手,真实项目由其错误码方案
	// (protoc-gen-go-errors / 中央错误包)提供,scratch 里写最小桩。
	helper := "package servicev1\n\nimport (\n\t\"google.golang.org/grpc/codes\"\n\t\"google.golang.org/grpc/status\"\n)\n\n" +
		"func ErrorBadRequest(msg string) error { return status.Error(codes.InvalidArgument, msg) }\n" +
		"func ErrorInternalServerError(msg string) error { return status.Error(codes.Internal, msg) }\n"
	_ = filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Ext(path) == ".go" && strings.HasSuffix(path, "_grpc.pb.go") {
			_ = os.WriteFile(filepath.Join(filepath.Dir(path), "errors.helper.go"), []byte(helper), 0o644)
		}
		return nil
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// copyTree 递归拷贝目录(常规文件),目标已存在的子目录合并、同名文件覆盖。
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func truncateLog(b []byte) string {
	const max = 2000
	if len(b) > max {
		return string(b[:max]) + "...(truncated)"
	}
	return string(b)
}

// compileBufTemplate 构造门禁的临时 buf.gen.yaml:
//   - managed 兜底无 go_package 的 proto(与真实项目一致);
//   - disable 清单与真实 buf 模板一致——依赖模块的 pb 走各自的 Go 模块
//     (分页 = github.com/tx7do/go-crud),不用本模块前缀重写;
//   - extraPlugins 按传输层追加(grpc→go-grpc,rest→go-http)。
func compileBufTemplate(module, out string, extraPlugins ...string) string {
	b := "version: v2\n" +
		"managed:\n" +
		"  enabled: true\n" +
		"  disable:\n" +
		"    - module: buf.build/googleapis/googleapis\n" +
		"    - module: buf.build/gnostic/gnostic\n" +
		"    - module: buf.build/tx7do/pagination\n" +
		"  override:\n" +
		"    - file_option: go_package_prefix\n" +
		"      value: " + module + "/api/gen/go\n" +
		"plugins:\n" +
		"  - local: protoc-gen-go\n" +
		"    out: " + out + "\n" +
		"    opt: paths=source_relative\n"
	for _, plug := range extraPlugins {
		b += "  - local: " + plug + "\n" +
			"    out: " + out + "\n" +
			"    opt: paths=source_relative\n"
	}
	return b
}

// serversHasRest 通过插件清单判断本次是否包含 REST 传输(go-http 插件)。
func serversHasRest(extraPlugins []string) bool {
	for _, p := range extraPlugins {
		if p == "protoc-gen-go-http" {
			return true
		}
	}
	return false
}
