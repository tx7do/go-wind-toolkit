# gowind-uiapp 命令行模式

`gowind-uiapp` 可执行文件是**双模式**的：不带参数启动图形界面；**带命令行参数**时进入无头 CLI 模式，将桌面应用的全部能力（项目探测、数据库、后端/前端代码生成、AI 助手、远程配置）以非交互方式暴露，供开发者与 AI Agent 调用。

> 独立的 `gowind-cli` 二进制已移除——CLI 能力内置于 `gowind-uiapp` 本体。
> 项目脚手架与开发工具不再在此重复，请使用 `gow` CLI（见下文「交给 gow 的能力」）。

- 结果输出到 **stdout**（JSON），日志输出到 **stderr**
- 默认输出缩进 JSON（人可读）；`--json` 输出单行紧凑 JSON（机器友好，适合 AI/脚本解析）
- 退出码：`0` 成功 / `1` 失败 / `2` 用法错误
- 零交互：所有输入通过 flag 或环境变量提供，缺参数直接报错
- Windows 发布版为 GUI 子系统构建，CLI 模式会自动附加父进程控制台；重定向输出（管道/文件）不受影响

构建：

```bash
cd gowind-uiapp && go build -o gowind-uiapp .
# 或使用 wails build 产物（同样支持命令行模式）
```

---

## 命令总览

| 命令 | 说明 |
|------|------|
| `gowind-uiapp project inspect` | 探测项目信息（模块路径、服务列表） |
| `gowind-uiapp db test/tables/columns` | 数据库连接测试与元数据 |
| `gowind-uiapp backend grpc/rest` | 后端 Kratos 微服务代码生成（`--servers` 选传输层） |
| `gowind-uiapp frontend gen/parse` | 前端 CRUD 代码生成（三框架）/ OpenAPI 服务解析预览 |
| `gowind-uiapp ai presets/test/ddl/partition/review/backend/find-openapi/config` | AI 助手（含按划分生成后端、配置查看/持久化） |
| `gowind-uiapp config types/services/export` | 远程配置中心导出 |

## 交给 gow 的能力

以下能力已从本工具移除，由 `gow` CLI 承担：

| 原命令 | 现用 gow 命令 |
|--------|--------------|
| `scaffold project` | `gow new <name>` |
| `scaffold service` | `gow add service <name>` |
| `dev buf` | `gow api` |
| `dev ent` | `gow ent [service]` |
| `dev wire` | `gow wire [service]` |
| `dev tidy` | `go mod tidy`（`gow generate`/`gow run`/`gow build` 等命令执行前自动执行） |

---

## project — 项目探测

```bash
gowind-uiapp project inspect --path D:/GoProject/go-wind-admin/backend
```

返回项目根目录、Go 模块路径、`app/` 下服务列表、是否有 `api/` 目录。

## db — 数据库

连接参数二选一：`--dsn`（支持环境变量 `GOWIND_DSN`）或离散参数（`--type --host --port --user --password --database`）。

```bash
# 连接测试
gowind-uiapp db test --dsn "mysql://user:pass@tcp(localhost:3306)/demo"

# 列出全部表
gowind-uiapp db tables --type postgresql --host localhost --port 5432 --user demo --password demo --database demo

# 列出某表的列
gowind-uiapp db columns --dsn "..." --table sys_user
```

> Oracle 连接暂不可用（上游驱动名错配），会明确报错。

## backend — 后端代码生成

数据源二选一：`--ddl <文件>`（本地 DDL，无需连库）或 `--dsn`。表到服务的映射用 `--mapping` JSON 文件或 `--tables` 简写。

```bash
# 表映射文件 tables.json:
# [{"table":"user","service":"identity"},
#  {"table":"role","service":"permission","protoPackage":"permission.service.v1"}]

# 生成 gRPC 全栈（proto+ent+service+server+装配+config），并自动执行
# go mod tidy -> buf generate -> ent generate（wire 仅对旧式 wire 服务执行，手写装配服务跳过）
gowind-uiapp backend grpc \
  --ddl schema.sql \
  --mapping tables.json \
  --orm ent --strategy per-table \
  --out /path/to/project

# 简写映射
gowind-uiapp backend grpc --dsn "$GOWIND_DSN" --tables user:identity,role:permission

# 只要生成产物、后处理自己控制
gowind-uiapp backend grpc --ddl schema.sql --mapping tables.json --skip-postprocess

# REST 网关（不生成 ORM/data，无后处理）
gowind-uiapp backend rest --ddl schema.sql --mapping tables.json \
  --service-name admin-portal --out /path/to/project

# 选择传输层（对应 gow generate -s）：默认 grpc，可多选
gowind-uiapp backend grpc --ddl schema.sql --mapping tables.json \
  --servers grpc,websocket --skip-postprocess --out /path/to/project
```

产物位于 `<out>/app/<服务名>/service/`，与 GUI 的 gRPC/REST 生成完全同源。

`--servers` 可选 `grpc` / `rest` / `websocket`（逗号分隔，缺省 `grpc`）。`websocket` 是消息驱动传输，只生成 `websocket_server.go`（不承载逐表 proto 服务），并在 `initApp` 装配里注入 `wsServer`/`wsMiddlewares`；它是**惰性**的——需在配置里补 `server.websocket` 段后才真正生效。GUI 后端向导的「传输层」多选框与此同源。

## frontend — 前端 CRUD 代码生成

```bash
# 预览将生成的文件清单（不写盘）
gowind-uiapp frontend gen \
  --openapi http://localhost:8000/q/openapi.yaml \
  --framework vue-vben \
  --module-path-map '{"role":"permission/role","api-audit-log":"log/api_audit_log"}' \
  --dry-run

# 写入前端项目（out 为前端 src 目录）
gowind-uiapp frontend gen \
  --openapi openapi.yaml \
  --framework vue-vben \
  --out D:/GoProject/go-wind-admin/frontend/admin/vue-vben/apps/admin/src

# 生成全部内容 JSON 到 stdout（供 AI 直接消费）
gowind-uiapp frontend gen --openapi openapi.yaml --framework react --stdout

# 仅解析 OpenAPI -> 服务清单（不生成，供 GUI/AI 预览有哪些 tag/操作）
gowind-uiapp frontend parse --openapi openapi.yaml
```

| 参数 | 说明 |
|------|------|
| `--openapi` | OpenAPI 3.0 YAML 文件路径或 http(s) URL（必填） |
| `--framework` | `vue-element` / `vue-vben` / `react`（必填） |
| `--tags` | 要生成的服务 tag（缺省全部） |
| `--types` | `composable,page,drawer,router,locale` 子集（缺省全部；react 的 composable 自动映射为 hooks） |
| `--module-path-map` | 文件名 -> 模块路径 JSON（或 `@file.json`），决定 `views/app/<module>/...` 布局 |
| `--router-modules` | 路由分组 JSON（或 `@file`），缺省 `auto` 按 basePath 自动检测 |
| `--out` | 前端项目 **src 目录**（与 `--dry-run`/`--stdout` 三选一） |

**写盘语义**：普通文件创建或覆盖（manifest 标注 `created`/`overwritten`）；vue-vben 的国际化是**合并式片段**——目标 `locales/langs/{lang}/page.json`/`menu.json` 已存在时按键合并写回（`merged`，键已存在则原位替换），不存在时新建独立片段文件（`merged-new`）。

生成器类型前缀映射（`permissionservicev1_` 等来自领域 proto 包）维护在 `gowind/pkg/frontendgen/openapi.go` 的 `serviceTypePrefixes`，黄金样本测试保证与 GUI/TS 版行为一致。

## ai — AI 助手

凭证可用 flag 或环境变量：`GOWIND_AI_PROVIDER` / `GOWIND_AI_BASE_URL` / `GOWIND_AI_API_KEY` / `GOWIND_AI_MODEL`。

```bash
gowind-uiapp ai presets                          # 服务商预设
gowind-uiapp ai test                             # 连通性测试
gowind-uiapp ai ddl --requirements req.md        # 需求文档 -> MySQL DDL
gowind-uiapp ai partition --ddl schema.sql       # DDL -> 微服务划分建议
gowind-uiapp ai review --files internal/user/service.go,internal/role/service.go

# 查找目录下 OpenAPI 文件
gowind-uiapp ai find-openapi --path /path/to/project

# AI 配置查看/持久化（默认脱敏，--show-secrets 显示完整 Key）
gowind-uiapp ai config show
gowind-uiapp ai config set --provider openai --base-url https://api.openai.com/v1 \
  --api-key sk-xxx --model gpt-4o

# 依据微服务划分 JSON 直接生成后端（partitions.json 来自 ai partition）
gowind-uiapp ai backend --ddl schema.sql --partitions @partitions.json --orm ent
```

典型的 AI 全链路组合：

```bash
gowind-uiapp ai ddl --requirements req.md --json | jq -r .content > schema.sql
gowind-uiapp ai partition --ddl schema.sql --json > partitions.json
# ai backend 直接读取 partitions.json 构建映射并生成，无需手工拼 mapping.json：
gowind-uiapp ai backend --ddl schema.sql --partitions @partitions.json --orm ent
```

## config — 远程配置导出

```bash
gowind-uiapp config types
gowind-uiapp config services --path /path/to/project
gowind-uiapp config export --type consul --endpoint http://localhost:8500 \
  --project demo --path /path/to/project --dry-run
gowind-uiapp config export --type nacos --endpoint http://localhost:8848 \
  --project demo --group DEFAULT_GROUP --service admin --path /path/to/project
```

Etcd 走 v3 gRPC 协议；endpoint 支持 `host:port` 或 `http(s)://host:port`，逗号分隔多节点。

---

## AI 调用约定

1. 优先 `--json` + stdout 解析；不要解析 stderr（那是日志）。
2. 退出码非 0 时，stdout（`--json`）会输出 `{"error":"..."}`。
3. 长流程（backend grpc 的后处理链）日志在 stderr 实时输出，最终结果一次性输出到 stdout。
4. 涉及凭证（数据库密码、AI API Key）建议用环境变量传入。
