package generators

import (
	"fmt"
	"strings"

	"github.com/tx7do/go-utils/stringcase"
)

// ==============================
// 登记行构造器
//
// 注入路径(锚点注入)与全新渲染路径(wiring.tpl)共用这些构造器,
// 保证两条路径产出的行逐字符一致;skipIf 为幂等探测子串。
// ==============================

// BuildRepoWiringLine 构造仓储层登记行(仓储型服务的 data 层构造)。
func BuildRepoWiringLine(model string, ormClientVar string) (line string, skipIf string) {
	camel := stringcase.LowerCamelCase(model)
	pascal := stringcase.UpperCamelCase(model)
	return fmt.Sprintf("\t%sRepo := data.New%sRepo(ctx, %s)", camel, pascal, ormClientVar),
		fmt.Sprintf("data.New%sRepo(", pascal)
}

// BuildServiceWiringLine 构造服务层登记行。useClient 为真时数据源为服务客户端变量(BFF),
// 否则为仓储变量。
func BuildServiceWiringLine(model string, useClient bool) (line string, skipIf string) {
	camel := stringcase.LowerCamelCase(model)
	pascal := stringcase.UpperCamelCase(model)
	ds := camel + "Repo"
	if useClient {
		ds = camel + "ServiceClient"
	}
	return fmt.Sprintf("\t%sService := service.New%sService(ctx, %s)", camel, pascal, ds),
		fmt.Sprintf("service.New%sService(", pascal)
}

// BuildServerArgLine 构造传输层服务实参行(注入到 NewRestServer/NewGrpcServer 调用实参列表)。
func BuildServerArgLine(model string) (line string, skipIf string) {
	camel := stringcase.LowerCamelCase(model)
	return "\t\t" + camel + "Service,", "\t\t" + camel + "Service,"
}

// BuildServerParamLine 构造 server 文件的服务形参行。
func BuildServerParamLine(model string) (line string, skipIf string) {
	camel := stringcase.LowerCamelCase(model)
	pascal := stringcase.UpperCamelCase(model)
	return fmt.Sprintf("\t%sService *service.%sService,", camel, pascal),
		fmt.Sprintf("%sService *service.%sService,", camel, pascal)
}

// BuildRouteLine 构造 server 文件的路由注册行。serverKind 为 "rest"(HTTP 网关)或
// "grpc"(后端服务);domain 为路由所属 api 域模块名,version 为 api 包版本。
func BuildRouteLine(serverKind string, domain string, version string, model string) (line string, skipIf string) {
	camel := stringcase.LowerCamelCase(model)
	pascal := stringcase.UpperCamelCase(model)
	svcVar := camel + "Service"
	alias := ApiPackageAlias(domain, version)
	if serverKind == "rest" {
		return fmt.Sprintf("\t%s.Register%sServiceHTTPServer(srv, %s)", alias, pascal, svcVar),
			fmt.Sprintf("Register%sServiceHTTPServer(srv, %s)", pascal, svcVar)
	}
	return fmt.Sprintf("\t%s.Register%sServiceServer(srv, %s)", alias, pascal, svcVar),
		fmt.Sprintf("Register%sServiceServer(srv, %s)", pascal, svcVar)
}

// ApiPackageAlias 生成 api 包的 import 别名,与模板函数 apiPackageAlias 一致。
func ApiPackageAlias(name string, version string) string {
	return stringcase.LowerCamelCase(strings.ToLower(name)) + stringcase.UpperCamelCase(version)
}

// ApiImportPath 构造 api 包的 import 路径,与 server 模板中的 import 语句逐字一致
// (rest 模板的 module 段不转小写,grpc 模板整体转小写)。
func ApiImportPath(serverKind string, module string, domain string, version string) string {
	domain = strings.ToLower(domain)
	version = strings.ToLower(version)
	if serverKind != "rest" {
		module = strings.ToLower(module)
	}
	return module + "/api/gen/go/" + domain + "/service/" + version
}

// ==============================
// 全新渲染的装配文件分节块
// ==============================

// WiringInfraClient 描述基础设施小节中的一个客户端构造。
type WiringInfraClient struct {
	Var         string
	CleanupVar  string
	Constructor string
	HasCleanup  bool
}

// WiringInfraClientFor 把 dbClient 名称解析为基础设施描述符;
// 仅支持生成了对应 client 模板的类型(ent/gorm/redis),其余忽略。
func WiringInfraClientFor(dbClient string) (WiringInfraClient, bool) {
	switch strings.TrimSpace(strings.ToLower(dbClient)) {
	case "ent", "entgo":
		return WiringInfraClient{
			Var:         "entClient",
			CleanupVar:  "cleanupEnt",
			Constructor: "client.NewEntClient",
			HasCleanup:  true,
		}, true
	case "gorm":
		return WiringInfraClient{
			Var:         "gormClient",
			CleanupVar:  "",
			Constructor: "client.NewGormClient",
			HasCleanup:  false,
		}, true
	case "redis":
		return WiringInfraClient{
			Var:         "redisClient",
			CleanupVar:  "cleanupRedis",
			Constructor: "client.NewRedisClient",
			HasCleanup:  true,
		}, true
	}
	return WiringInfraClient{}, false
}

// WiringBlocks 为 wiring.tpl 的全部渲染变量。
type WiringBlocks struct {
	HeaderComment string

	InfraBlock     string
	RepoBlock      string
	ServiceBlock   string
	TransportBlock string
	NewAppArgs     string

	ImportData       bool
	ImportClient     bool
	ImportServer     bool
	ImportServicePkg bool
}

const wiringSectionBanner = "\t// ═══════════════════════ %s ═══════════════════════\n"

// BuildWiringBlocks 构造全新装配文件的分节块。
// repoModels 为本次生成产生了仓储文件的模型,serviceModels 为本次产生了服务文件的模型;
// 两者为空(纯脚手架)时各分节仅含锚点。orm 为仓储型服务的 ORM 类型(决定客户端变量命名),
// BFF 传空。dbClients 为要构造的基础设施客户端列表。
func BuildWiringBlocks(
	servers []string,
	useRepo bool,
	orm string,
	dbClients []string,
	repoModels []string,
	serviceModels []string,
) WiringBlocks {
	hasTemplatedServer := false
	for _, s := range servers {
		switch strings.TrimSpace(strings.ToLower(s)) {
		case "grpc", "rest", "websocket":
			hasTemplatedServer = true
		}
	}
	useGrpc := false
	for _, s := range servers {
		if strings.TrimSpace(strings.ToLower(s)) == "grpc" {
			useGrpc = true
			break
		}
	}
	// BFF:纯 rest 且仓储关闭,数据层为服务客户端;其余形态数据层为仓储。
	useClient := !useGrpc && !useRepo
	// 客户端分节仅存在于 BFF 形态;仓储分节存在于其余形态。
	clientSection := useClient
	repoSection := !useClient

	var blocks WiringBlocks
	blocks.HeaderComment = buildWiringHeaderComment(useClient)

	// ── 一、基础设施 ──
	var infra strings.Builder
	infra.WriteString(fmt.Sprintf(wiringSectionBanner, "一、基础设施"))
	for _, dbClient := range dbClients {
		desc, ok := WiringInfraClientFor(dbClient)
		if !ok {
			continue
		}
		infra.WriteString("\n")
		if desc.HasCleanup {
			infra.WriteString(fmt.Sprintf(
				"\t%s, %s, err := %s(ctx)\n"+
					"\tif err != nil {\n"+
					"\t\trollback()\n"+
					"\t\treturn nil, nil, err\n"+
					"\t}\n"+
					"\tcleanups = append(cleanups, %s)\n",
				desc.Var, desc.CleanupVar, desc.Constructor, desc.CleanupVar))
		} else {
			infra.WriteString(fmt.Sprintf(
				"\t%s, err := %s(ctx)\n"+
					"\tif err != nil {\n"+
					"\t\trollback()\n"+
					"\t\treturn nil, nil, err\n"+
					"\t}\n",
				desc.Var, desc.Constructor))
		}
	}
	blocks.InfraBlock = infra.String()

	// ── 二、仓储层 / 服务客户端 ──
	var repo strings.Builder
	if clientSection {
		repo.WriteString(fmt.Sprintf(wiringSectionBanner, "二、服务客户端(internal/data)"))
		repo.WriteString("\t" + AnchorClient + "\n")
	} else if repoSection {
		repo.WriteString(fmt.Sprintf(wiringSectionBanner, "二、仓储层(internal/data)"))
		if orm != "" {
			ormVar := stringcase.LowerCamelCase(orm) + "Client"
			for _, model := range repoModels {
				line, _ := BuildRepoWiringLine(model, ormVar)
				repo.WriteString(line + "\n")
			}
		}
		repo.WriteString("\t" + AnchorRepo + "\n")
	}
	blocks.RepoBlock = repo.String()

	// ── 三、服务层 ──
	var svc strings.Builder
	svc.WriteString(fmt.Sprintf(wiringSectionBanner, "三、服务层(internal/service)"))
	for _, model := range serviceModels {
		line, _ := BuildServiceWiringLine(model, useClient)
		svc.WriteString(line + "\n")
	}
	svc.WriteString("\t" + AnchorService + "\n")
	blocks.ServiceBlock = svc.String()

	// ── 四、传输层 ──
	var transport strings.Builder
	transport.WriteString(fmt.Sprintf(wiringSectionBanner, "四、传输层(internal/server)"))
	newAppArgs := ""
	for _, s := range servers {
		kind := strings.TrimSpace(strings.ToLower(s))
		switch kind {
		case "rest":
			transport.WriteString("\n\trestMiddlewares := server.NewRestMiddleware(ctx)\n")
			transport.WriteString("\n\thttpServer, err := server.NewRestServer(ctx, restMiddlewares,\n")
			for _, model := range serviceModels {
				line, _ := BuildServerArgLine(model)
				transport.WriteString(line + "\n")
			}
			transport.WriteString("\t\t" + AnchorRestArg + "\n")
			transport.WriteString("\t)\n")
			transport.WriteString("\tif err != nil {\n\t\trollback()\n\t\treturn nil, nil, err\n\t}\n")
			newAppArgs += "\thttpServer,\n"
		case "grpc":
			transport.WriteString("\n\tgrpcMiddlewares := server.NewGrpcMiddleware(ctx)\n")
			transport.WriteString("\n\tgrpcServer, err := server.NewGrpcServer(ctx, grpcMiddlewares,\n")
			for _, model := range serviceModels {
				line, _ := BuildServerArgLine(model)
				transport.WriteString(line + "\n")
			}
			transport.WriteString("\t\t" + AnchorGrpcArg + "\n")
			transport.WriteString("\t)\n")
			transport.WriteString("\tif err != nil {\n\t\trollback()\n\t\treturn nil, nil, err\n\t}\n")
			newAppArgs += "\tgrpcServer,\n"
		case "websocket":
			// WebSocket 以消息类型驱动,不登记 proto 服务;serviceModels 在此不产生注册行。
			transport.WriteString("\n\twsMiddlewares := server.NewWebsocketMiddleware(ctx)\n")
			transport.WriteString("\n\twsServer, err := server.NewWebsocketServer(ctx, wsMiddlewares)\n")
			transport.WriteString("\tif err != nil {\n\t\trollback()\n\t\treturn nil, nil, err\n\t}\n")
			newAppArgs += "\twsServer,\n"
		default:
			transport.WriteString(fmt.Sprintf(
				"\n\t// TODO: 传输层 %s 未提供生成模板,请在此构造实例并接入 newApp。\n", kind))
			newAppArgs += fmt.Sprintf("\tnil, // TODO: 构造 %s 实例\n", kind)
		}
	}
	blocks.TransportBlock = transport.String()
	blocks.NewAppArgs = newAppArgs

	// import 需求:仅当对应构造行实际存在于分节中时才引入包。
	blocks.ImportClient = strings.Contains(blocks.InfraBlock, "client.")
	blocks.ImportData = strings.Contains(blocks.RepoBlock, "data.New")
	blocks.ImportServicePkg = strings.Contains(blocks.ServiceBlock, "service.New")
	blocks.ImportServer = hasTemplatedServer

	return blocks
}

// buildWiringHeaderComment 构造装配文件的函数文档注释。
func buildWiringHeaderComment(isBff bool) string {
	var b strings.Builder
	b.WriteString("// initApp 手写装配整个应用,是本服务的依赖注入点。\n")
	b.WriteString("// initApp assembles the whole application by hand — the dependency injection point.\n")
	b.WriteString("//\n")
	b.WriteString("// 装配严格单向分层,自上而下的阅读顺序即依赖方向:\n")
	b.WriteString("// The wiring is strictly layered; reading top-down follows the dependency direction:\n")
	b.WriteString("//\n")
	if isBff {
		b.WriteString("//\t基础设施 → 服务客户端(data) → 服务层(service) → 传输层(server)\n")
		b.WriteString("//\n")
		b.WriteString("// 本服务是 BFF 网关:不直连数据库,数据层由对后端服务的 gRPC 服务客户端替代。\n")
		b.WriteString("// 服务客户端的构造行由项目按自身约定登记(如经服务发现),CRUD 生成器不代填;\n")
		b.WriteString("// 客户端就位之前,引用它的服务构造行无法通过编译。\n")
	} else {
		b.WriteString("//\t基础设施 → 仓储层(data) → 服务层(service) → 传输层(server)\n")
	}
	b.WriteString("//\n")
	b.WriteString("// 约定 / Conventions:\n")
	b.WriteString("//   - 新增模块:在对应分层小节的锚点注释后追加构造行,或由 CRUD 生成器自动注入,\n")
	b.WriteString("//     并传给下游消费者;漏接由编译器在调用处报错。\n")
	b.WriteString("//     To add a module: append a constructor line after the anchor comment of its layer\n")
	b.WriteString("//     section, or let the CRUD generator inject it, then pass it to downstream\n")
	b.WriteString("//     consumers; a missing connection is a compile error at the call site.\n")
	b.WriteString("//   - 持有 cleanup 的资源创建成功后立即注册进 cleanups;任何一步失败,rollback 逆序执行已注册\n")
	b.WriteString("//     的清理(shutdown 与中途失败共用同一条 LIFO 路径)。\n")
	b.WriteString("//     Resources owning a cleanup register it immediately; rollback runs them LIFO both on\n")
	b.WriteString("//     mid-way failure and on shutdown.\n")
	b.WriteString("//   - 本文件只做构造与传参,不写业务逻辑。\n")
	b.WriteString("//     Construction and parameter passing only; no business logic in this file.\n")

	return b.String()
}
