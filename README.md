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

| 功能 | 说明 |
|---|---|
| 项目脚手架 | `gow new` — 一键创建 Kratos 项目 |
| 服务管理 | `gow add service` — 添加 gRPC/REST 微服务 |
| 代码生成 | `gow generate` — 从数据库 schema 生成 CRUD 代码 |
| Ent / GORM 模型 | 从 SQL 数据库自动生成 ORM 模型 |
| Protobuf 生成 | 自动生成 gRPC & REST proto 定义 |
| Wire 依赖注入 | 自动生成 Wire 依赖注入 |
| 微服务演进 | `gow extract` — 渐进式服务拆分与模块提取 |
| 配置导出 | 导出配置到 Consul / Etcd / Nacos |
| 桌面 UI | 可视化开发运维面板 |

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
gow add service admin -s grpc
```

### 从数据库生成 CRUD 代码

```shell
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" --service user
```

### 微服务演进（模块提取）

```shell
# 从 admin 服务提取 role 模块到 user 服务
# 目标服务不存在时自动创建，ORM 类型自动侦测
gow extract admin user -o role

# 提取多个实体
gow extract admin user -o role,permission
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

