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

### 4. 数据库驱动代码生成

从现有数据库生成完整的 CRUD 微服务代码（proto、ORM、service、server、wire、config）：

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

### 5. 工具代码生成


```shell
# 为所有服务生成 Ent
gow ent

# 为指定服务生成 Ent
gow ent admin
```

#### Wire 依赖注入生成

```shell
# 为所有服务生成 Wire
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
  -s, --server strings   服务类型：grpc / rest（可多选）
  -d, --dao strings      数据访问层：gorm / ent / redis（可多选）
  -o, --orm string       ORM 类型：gorm / ent（默认：ent）
```

### `gow generate` — 数据库驱动代码生成

从数据库 schema 生成完整的 Kratos 微服务代码（proto、ORM、service、server、wire、config）。

```shell
gow generate [flags]
# 或者
gow gen [flags]

Flags:
      --dsn string              Database source name, e.g. mysql://user:pass@tcp(localhost:3306)/dbname
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
```

### `gow run` — 运行服务

```shell
gow run [service-name]
```

### `gow ent` — Ent 代码生成

```shell
gow ent [service-name]
```

### `gow wire` — Wire 代码生成

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
- ✅ 数据库驱动 CRUD 代码生成（proto、ORM、service、server、wire、config）
- ✅ 自动生成 Ent / GORM 模型
- ✅ 自动生成 Protobuf & API 定义
- ✅ 自动生成 Wire 依赖注入
- ✅ 一键运行、热重载支持
- ✅ 统一 CLI 入口，降低学习成本
- ✅ 桌面 UI 可视化面板（Wails）
