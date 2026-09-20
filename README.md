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

通过向导式界面完成从数据库 Schema 到完整微服务代码的全流程生成。支持直连数据库（MySQL、PostgreSQL）、SQL 文件、远程 URL、在线编辑器四种方式导入 Schema；为每张表分配所属微服务、配置 Proto 包策略（每表独立包 / 按服务分包 / 自定义包名）；选择 ORM 类型（Ent / GORM），一键生成 gRPC 或 REST 服务代码，自动完成后处理（`go mod tidy` → `buf generate` → `ent generate` → `wire generate`）。

> 数据源支持 MySQL / PostgreSQL 连接串、DDL 文本（SQL 文件 / 远程 URL / 在线编辑器）与 `ent://<目录>` / `gorm://<目录>` Go 源码目录。SQLite、Oracle 连接不能作为生成数据源——SQLite 可先用 `sqlite3 <文件> .schema` 导出 DDL，再走「SQL 文件」导入。

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

# 预览将创建的服务布局，不做任何变更
gow add service admin -s grpc --dry-run
```

### 运行服务

```shell
# 在服务目录下直接运行
gow run

# 指定服务名运行
gow run admin

# 不指定服务名时运行模块内全部服务
gow run
```

### 热重载运行（--watch）

```shell
# watch 模式:保存文件自动重建并重启受影响的服务
gow run admin --watch

# 全部服务一起 watch(输出按服务名加前缀)
gow run --watch

# 也可在服务目录下运行
cd app/admin/service && gow run -w
```

watch 模式说明：

- 递归监听模块根目录（跳过 `.git`、`vendor`、`node_modules`、`bin`、隐藏目录等），`.go`/`.yaml`/`.yml`/`.json`/`.toml`/`.properties`/`.proto` 变更触发重建重启，`*_test.go` 与隐藏文件忽略
- 只重建重启**受影响的服务**：变更位于某服务目录内只重启该服务；模块级共享代码变更重启全部
- 500ms 防抖合并连续保存；**编译失败保持旧进程运行**，修复后下次保存自动重试
- 停止时先 SIGTERM 优雅退出（5s 宽限后强杀）；`.proto` 变更只触发重启，需先 `gow api` 重新生成

### 编译服务

```shell
# 编译全部服务到各服务 bin/ 目录
gow build

# 编译指定服务
gow build admin user

# 交叉编译(GOOS×GOARCH 全组合,产物带 _<goos>_<goarch> 后缀)
gow build --os linux,windows --arch amd64,arm64

# 注入版本号并产出精简二进制
gow build --version v1.2.3 --strip --trimpath

# 自定义 ldflags 与输出目录
gow build -o ./dist --ldflags "-X main.commit=$(git rev-parse --short HEAD)"
```

`--version` 通过 `-ldflags "-X main.version=..."` 注入到各服务 `main` 包的 `version` 变量（脚手架模板默认生成该变量）。

### 查看版本

```shell
gow version

# 发布构建时通过 ldflags 注入(发布流程已内置):
# go build -ldflags "-X main.version=v1.2.3 -X main.commit=abc1234 -X main.date=..."
gow version
# gow version v1.2.3 (commit: abc1234, built: 2026-09-12T08:00:00Z)
```

`go install` 安装时自动回落显示模块版本；本地源码构建显示 `dev`。

### 从数据库生成 CRUD 代码

```shell
# 交互式（提示输入 DSN 和服务名）
gow generate

# 校验数据源、解析表清单并预览计划，不写入任何文件
# （支持数据库 DSN 或内联 DDL 文本）
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" --service user --dry-run

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

# 显式子命令形式（等价）
gow ent generate admin

# 为服务新增 schema 并自动重新生成 Ent 代码
gow ent add admin Role,Permission
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

### ent schema 逆向导出 DDL（migrate）

```shell
# 为所有含 ent schema 的服务导出 DDL(默认 mysql 方言)
gow migrate

# 指定服务与方言(mysql / postgres / sqlite)
gow migrate admin --dialect postgres

# 集中输出到单目录(文件名为 <service>.<dialect>.sql)
gow migrate --dialect sqlite -o ./dist/sql

# 指定数据库版本以生成方言特定语法
gow migrate admin --dialect mysql --db-version 5.7
```

`gow migrate` 把 `app/<服务>/service/internal/data/ent` 的 ent schema 逆向为 `CREATE TABLE` 脚本（通过 ent 的 `schema.DDL` 离线规划，**无需连接数据库**），默认写入各服务的 `migrations/schema.<方言>.sql`。缺失的 ent 代码生成会自动补齐（同 `gow ent`）；非 ent（gorm）服务自动跳过。产物可直接作为 Atlas 基线迁移或用于审查 schema 状态。

### 微服务演进（模块提取）

```shell
# 从 admin 服务提取 role 模块到 user 服务
# 目标服务不存在时自动创建，ORM 类型自动侦测
gow extract admin user -o role

# 提取多个实体
gow extract admin user -o role,permission

# 手动指定 ORM 类型
gow extract admin user -o role --orm gorm

# 预览全部文件动作（复制/修改/删除），不做任何变更
gow extract admin user -o role --dry-run

# 保留源文件（默认删除）
gow extract admin user -o role --keep-source

# 脚本/CI 场景跳过删除确认
gow extract admin user -o role --yes
```

提取**默认删除源端文件**，因此：执行前会先打印完整计划（复制、就地修改、删除清单），删除前要求交互确认（默认拒绝）；`--dry-run` 只预览零变更，`--keep-source` 非破坏性免确认，`--yes` 供脚本跳过确认。

## 特性总结

- 一键创建 Kratos 标准项目
- 一键添加多协议微服务（gRPC + REST）
- 服务热重载运行与交叉编译（`gow run --watch` / `gow build`）
- 数据库驱动 CRUD 代码生成（proto、ORM、service、server、wire、config）
- 自动生成 Ent / GORM 模型
- ent schema 逆向导出 DDL（`gow migrate`，离线、免连库）
- 自动生成 Protobuf & API 定义
- 自动生成 Wire 依赖注入
- 微服务渐进式拆分与演进（模块提取）
- 配置导出到 Consul / Etcd / Nacos
- 桌面 UI 可视化面板（Wails）

