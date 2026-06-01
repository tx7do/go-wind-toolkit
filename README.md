# GoWind Toolkit

[English](./README.en-US.md) | **中文**

一个为 Go-Kratos 微服务生态打造的**一站式全能工具集**，包含脚手架、自动化代码生成、开发辅助、运维工具、命令行与可视化桌面客户端。

## 项目结构

```
go-wind-toolkit/
├── gowind/            # 模块 1: CLI + 共享库
│   ├── cmd/gow/       # CLI 入口 (go install .../cmd/gow@latest)
│   ├── pkg/           # 导出库（CLI 和 UI 共用）
│   │   ├── generators/      # 代码生成模板与引擎
│   │   ├── sqlkratos/       # SQL → 完整 Kratos 服务生成器
│   │   ├── sqlorm/          # SQL → ORM (ent/gorm) 生成器
│   │   ├── sqlproto/        # SQL → Protobuf/gRPC/REST 转换器
│   │   ├── service/         # 服务脚手架生成器
│   │   ├── extract/         # 微服务模块提取器
│   │   └── configexporter/  # 配置导出 (Consul/Etcd/Nacos)
│   └── internal/     # CLI 专用代码
├── gowind-uiapp/      # 模块 2: Wails 桌面 UI
│   ├── main.go
│   ├── frontend/     # Vue.js 前端
│   └── internal/     # UI 专用代码
└── README.md
```

## 功能一览

| 功能 | CLI | 桌面 UI |
|---|:---:|:---:|
| 项目脚手架 (`gow new`) | ✅ | ✅ |
| 添加微服务 (`gow add service`) | ✅ | ✅ |
| 数据库驱动 CRUD 代码生成 | ✅ | ✅ |
| Ent / GORM 模型生成 | ✅ | ✅ |
| Protobuf gRPC & REST 定义生成 | ✅ | ✅ |
| Wire 依赖注入生成 | ✅ | ✅ |
| 微服务模块提取 (`gow extract`) | ✅ | — |
| 配置导出到 Consul / Etcd / Nacos | — | ✅ |
| 可视化表配置与服务分配 | — | ✅ |
| AI 辅助 DDL 生成与微服务划分 | — | ✅ |
| AI 代码审查 | — | ✅ |
| 服务启停管理 | — | ✅ |
| 开发工具 (buf/wire/ent) | — | ✅ |

## 桌面客户端 — gowind-uiapp

基于 [Wails](https://wails.io/) (Go + Vue 3) 构建的跨平台桌面应用，为 Go-Kratos 微服务开发提供可视化的全方位工具支持。

### 后端代码生成

通过向导式界面完成从数据库 Schema 到完整微服务代码的全流程生成。支持直连数据库（MySQL、PostgreSQL、SQLite、Oracle）、SQL 文件、远程 URL、在线编辑器四种方式导入 Schema；为每张表分配所属微服务、配置 Proto 包策略（每表独立包 / 按服务分包 / 自定义包名）；选择 ORM 类型（Ent / GORM），一键生成 gRPC 或 REST 服务代码，自动完成后处理（`go mod tidy` → `buf generate` → `ent generate` → `wire generate`）。

### 前端代码生成

基于 OpenAPI 定义自动生成完整的前端管理页面代码，支持三种主流前端框架：

- **Vue 3 + Element Plus**：生成 API 调用层、Vue Query Composable、列表页（表格 + 搜索 + 分页）、编辑抽屉、路由配置、i18n 国际化文件
- **Vue 3 + Vben Admin**（VxeGrid + useVbenDrawer）：生成 API 调用层、Composable、列表页、编辑抽屉、路由配置、i18n 国际化（页面 + 菜单）
- **React + Ant Design Pro**（ProTable + DrawerForm）：生成 API 调用层、React Query Hooks、列表页、编辑抽屉、路由配置、i18n 国际化文件

### 远程配置导出

将本地配置文件一键导出到 Consul、Etcd 或 Nacos，支持批量导出和服务级选择性导出。

### AI 助手

集成多种 LLM 提供商（OpenAI、DeepSeek、Ollama 等），支持从自然语言需求生成 DDL、AI 辅助微服务划分、代码审查，加速开发决策。

### 开发工具

内置 `buf generate`、`ent generate`、`wire generate`、`go mod tidy`、服务启停管理等常用开发命令，无需切换终端。

## 安装 CLI

```shell
go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest
```

## 快速开始

### 创建项目

```shell
gow new myproject
cd myproject && go mod tidy
```

### 添加服务

```shell
# 添加 gRPC 服务
gow add service admin -s grpc

# 添加 REST 服务
gow add service admin -s rest

# 同时支持 gRPC + REST
gow add service admin -s rest -s grpc

# 指定 ORM（gorm / ent）
gow add service admin -d gorm -s grpc
```

### 运行服务

```shell
# 在服务目录下直接运行
gow run

# 指定服务名运行
gow run admin
```

### 从数据库生成 CRUD 代码

```shell
# 交互式（提示输入 DSN 和服务名）
gow generate

# 完整命令行
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" --service user

# 指定 ORM 和表
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" \
  --service user --orm ent --servers grpc --tables users,roles

# 仅生成 proto 文件
gow generate --dsn "postgres://user:pass@localhost:5432/dbname" --service admin --proto-only

# 生成 REST 服务（代理自 gRPC 服务）
gow generate --dsn "mysql://..." --service user-admin \
  --servers rest --source-module user --skip-orm
```

### Ent 代码生成

```shell
# 为所有服务生成 Ent 代码
gow ent

# 为指定服务生成
gow ent admin
```

### Wire 依赖注入生成

```shell
# 为所有服务生成 Wire
gow wire

# 为指定服务生成
gow wire admin
```

### Protobuf / API 代码生成

```shell
# 为所有服务生成 Proto & API
gow api
```

### 微服务演进（模块提取）

```shell
# 从 admin 服务提取 role 模块到 user 服务
# 目标服务不存在时自动创建，ORM 类型自动侦测
gow extract admin user -o role

# 提取多个实体
gow extract admin user -o role,permission

# 手动指定 ORM 类型
gow extract admin user -o role --orm gorm

# 保留源文件（默认删除）
gow extract admin user -o role --keep-source
```

### 查看版本

```shell
gow version
```

## 特性总结

- 一键创建 Kratos 标准项目
- 一键添加多协议微服务（gRPC + REST）
- 数据库驱动 CRUD 代码生成（proto、ORM、service、server、wire、config）
- 自动生成 Ent / GORM 模型
- 自动生成 Protobuf & API 定义
- 自动生成 Wire 依赖注入
- 微服务渐进式拆分与演进（模块提取）
- 配置导出到 Consul / Etcd / Nacos
- 桌面 UI 可视化面板（Wails）

