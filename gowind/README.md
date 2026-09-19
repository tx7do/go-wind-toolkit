# GoWind CLI (gow)

GoWind CLI (gow) 是 GoWind Toolkit 的核心命令行入口，提供项目脚手架、微服务管理、代码生成、一键运行等全流程能力，覆盖从项目创建到开发运维的完整生命周期。

[English](./README.en-US.md) | **中文**

## 安装

```shell
go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest
```

验证安装：

```shell
gow version
gow help
```

## 快速开始

### 1. 创建新项目

```shell
# 基础创建
gow new myproject
cd myproject
go mod tidy
```

```shell
# 指定模块名
gow new myproject -m github.com/yourusername/myproject
cd myproject
go mod tidy
```

### 2. 添加微服务

```shell
# 添加基础服务
gow add service admin
gow add service user
go mod tidy
```

#### 高级选项

```shell
# gRPC 服务
gow add service order -s grpc

# REST 服务
gow add service admin -s rest

# 同时支持 gRPC + REST
gow add service admin -s rest -s grpc

# 指定 ORM（gorm/ent）+ gRPC
gow add svc payment -d gorm -s grpc

# 多数据源 + 多协议
gow add service admin -s rest -s grpc -d gorm -d redis
```

### 3. 运行微服务

```shell
# 当前目录直接运行（需在 app/xxx/service 下）
gow run
```

```shell
# 指定服务运行
gow run admin
```

```shell
# 当前目录不在任一服务内时,一并编译并运行模块内全部服务
# (各服务输出带名称前缀,Ctrl+C 一并停止全部)
gow run
```

### 4. 数据库驱动代码生成

从现有数据库生成完整的 CRUD 微服务代码（proto、ORM、service、server、装配、config）。装配按目标服务形态分流：手写装配服务注入 wiring.go 锚点，旧式 wire 服务更新 provider 集：

```shell
# 交互式（会提示输入 DSN 和 service name）
gow generate

# 完整命令行
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" --service user

# 指定 ORM 和表
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" \
  --service user --orm ent --servers grpc --tables users,roles

# 仅生成 proto 文件
gow generate --dsn "postgres://user:pass@localhost:5432/dbname" --service admin --proto-only

# 生成 REST 服务（从 gRPC 服务代理）
gow generate --dsn "mysql://..." --service user-admin \
  --servers rest --source-module user --skip-orm

# 使用别名
gow gen --dsn "..." --service user
```

#### 从 Go 源码 schema 直接生成（无需数据库）

已有 ent schema 或 gorm model 目录时，可跳过数据库直接把 DSN 写成源码源，生成同一套下游 CRUD 代码：

```shell
# ent schema 目录（服务约定路径 internal/data/ent/schema）
gow generate --dsn "ent://internal/data/ent/schema" --service user --orm ent

# gorm model 目录：解析模型后在 daoPath 下经临时包回转生成 DAO（自动清理）
# 目标 models 目录只补缺失文件，与手写模型类型冲突时跳过，绝不覆盖
gow generate --dsn "gorm://internal/data/models" --service payment --orm gorm

# 先用 dry-run 校验源与表解析结果
gow generate --dsn "ent://..." --service user -n
```

约束与说明：
- `ent://` 要求 `--orm ent`，`gorm://` 要求 `--orm gorm`（CLI 与生成器两侧都会校验）。
- ent 源的 Mixin / GoType / SchemaType 覆写无法静态还原，会打印 `[WARN]` 提示缺失字段。
- 解析产物中的 m2m 中间表与数据库流程一致，按 join table 规则从 proto 中剔除。

### 5. 微服务演进（提取拆分）

从一个已有服务中提取业务模块到另一个服务，实现渐进式微服务拆分：

```shell
# 从 admin 服务提取 role 模块到 user 服务
# 目标服务不存在时会自动创建
# ORM 类型根据源服务目录结构自动侦测
gow extract admin user --obj role

# 提取多个实体（逗号分隔）
gow extract admin user -o role,permission

# 提取多个实体（重复 flag）
gow extract admin user --obj role --obj permission

# 保留源文件
# gow extract admin user -o role --keep-source
```

### 6. 工具代码生成


```shell
# 为所有服务生成 Ent
gow ent

# 为指定服务生成 Ent
gow ent admin
```

#### 依赖装配

新建服务默认采用手写装配：`cmd/server/wiring.go` 按分层小节手写构造（基础设施 → 仓储/服务客户端 → 服务层 → 传输层），带 cleanup 的资源注册进 LIFO 回滚表，各登记位以 `register:*` 锚点注释标记，新增 CRUD 模块由生成器自动注入，无 wire 依赖。
`gow add service --wire` 可退回旧式 wire 脚手架；`gow wire` 命令为既有 wire 形态服务保留（手写装配的服务自动跳过），向下兼容：

```shell
# 旧式 wire 脚手架（默认生成手写装配）
gow add service admin --wire

# 为所有服务生成 Wire（仅 wire 形态服务，手写装配服务自动跳过）
gow wire

# 为指定服务生成 Wire
gow wire admin
```

#### Protobuf / API 代码生成

```shell
# 为所有服务生成 Proto & API
gow api
```

## 完整命令参考

### `gow new` — 项目初始化

```shell
gow new <project-name> [flags]
# 或者
gow new project <project-name> [flags]

Flags:
  -m, --module string   Go module 名称（默认：项目名）
```

### `gow add` — 新增组件

```shell
gow add service <service-name> [flags]

Flags:
  -s, --server strings   服务类型：grpc / rest / websocket（可多选）
  -d, --dao strings      数据访问层：gorm / ent / redis（可多选）
  -o, --orm string       ORM 类型：gorm / ent（默认：ent）
```

> `websocket` 传输以消息类型驱动（`srv.RegisterMessageHandler`），不注册 proto 服务。
> 生成的 `internal/server/websocket_server.go` 仅在该服务 `configs/*.yaml` 含 `server.websocket`
> 段（`network`/`addr`/`path`/`codec`）时才启用，消息处理器在 `register:route` 锚点后手动登记。

### `gow generate` — 数据库驱动代码生成

从数据库 schema 生成完整的 Kratos 微服务代码（proto、ORM、service、server、装配、config）。
数据源除数据库 DSN / SQL 文件外，还支持 Go 源码 schema：`ent://<ent schema 目录>` 与 `gorm://<gorm model 目录>`（见上文「从 Go 源码 schema 直接生成」）。

```shell
gow generate [flags]
# 或者
gow gen [flags]

Flags:
      --dsn string              Data source: e.g. mysql://user:pass@tcp(localhost:3306)/dbname, or ent://<dir> / gorm://<dir>
      --driver string           Database driver: mysql, postgres (default "mysql")
      --service string          Service name (module name)
      --orm string              ORM type: ent, gorm (default "ent")
  -s, --servers strings         Server types: grpc, rest (default [grpc])
  -t, --tables strings          Tables to include (default: all)
      --exclude-tables strings  Tables to exclude
      --module-version string   API module version (default "v1")
      --proto-only              Only generate proto files
      --skip-orm                Skip ORM code generation
      --skip-config             Skip config file generation
      --skip-makefile           Skip Makefile generation
      --source-module string    Source module name for REST service
  -n, --dry-run                 Validate the data source, resolve tables and preview the plan without writing anything
```

### `gow extract` — 微服务模块提取

从一个已有服务中提取业务模块（schema、repo、service、wire、server）到目标服务，用于微服务的渐进式拆分与演进。目标服务不存在时会自动创建脚手架。ORM 类型根据源服务目录结构自动侦测。

```shell
gow extract <source-service> <target-service> -o <model> [-o <model>...] [flags]

Flags:
  -o, --obj stringArray   要提取的实体名称（支持逗号分隔或重复使用）
      --orm string        ORM 类型覆盖：ent, gorm（默认自动侦测）
      --keep-source       保留源文件（默认删除）
```

#### 示例

```shell
# 单个实体
gow extract admin user -o role

# 多个实体
gow extract admin user -o role,permission
gow extract admin user --obj role --obj permission

# 手动指定 ORM 类型
gow extract admin user -o role --orm gorm

# 保留源文件
# gow extract admin user -o role --keep-source
```

### `gow run` — 运行服务

```shell
# 运行指定服务;当前目录在某服务内时运行该服务;
# 否则一并编译并运行模块内全部服务,输出按服务名加前缀,Ctrl+C 一并停止。
gow run [service-name]
```

### `gow build` — 编译服务（支持交叉编译）

```shell
# 编译全部服务(默认输出到各服务 bin/ 目录)
gow build

# 编译指定服务
gow build admin

# 交叉编译:GOOS×GOARCH 全组合,产物带 _<goos>_<goarch> 后缀;--out 集中到单目录
gow build --os linux,windows --arch amd64 --out ./dist
```

### `gow ent` — Ent 代码生成

```shell
gow ent [service-name]
```

### `gow wire` — Wire 代码生成（仅旧式 wire 服务；手写装配服务自动跳过）

```shell
gow wire [service-name]
```

### `gow api` — Protobuf / API 代码生成

```shell
gow api
```

### gow version — 查看版本

```shell
gow version
```

### gow help — 帮助

```shell
gow help
gow help <command>
```

### 项目结构（生成后）

```shell
myproject/
├── app/
│   ├── admin/
│   │     └── service/
│   └── user/
│          └── service/
│   │            └── internal/
│   │                   └── data/
│   │                          └── ent/
├── api/
│   └── protos/
├── go.mod
└── go.sum
```

## 特性总结

- ✅ 一键创建 Kratos 标准项目
- ✅ 一键添加多协议微服务（gRPC + REST）
- ✅ 数据库驱动 CRUD 代码生成（proto、ORM、service、server、装配、config）
- ✅ 自动生成 Ent / GORM 模型
- ✅ 自动生成 Protobuf & API 定义
- ✅ 依赖装配：手写 wiring.go 锚点式登记（默认）或 Wire provider 集（旧式，向下兼容）
- ✅ 微服务渐进式拆分与演进（模块提取）
- ✅ 一键运行、热重载支持
- ✅ 统一 CLI 入口，降低学习成本
- ✅ 桌面 UI 可视化面板（Wails）
