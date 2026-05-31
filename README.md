# GoWind Toolkit

**A comprehensive all-in-one toolkit for Go-Kratos microservice development, including scaffolding, CRUD code generation,
dev tools, operation utilities, CLI and desktop UI.**

一个为 Go-Kratos 微服务生态打造的**一站式全能工具集**，包含脚手架、自动化代码生成、开发辅助、运维工具、命令行与可视化桌面客户端。

## Project Structure

```
go-wind-toolkit/
├── gowind/            # Module 1: CLI + Shared Libraries
│   ├── cmd/gow/       # CLI entry point (go install .../cmd/gow@latest)
│   ├── pkg/           # Exported libraries (shared by CLI and UI)
│   │   ├── generators/      # Code generation templates & engine
│   │   ├── sqlkratos/       # SQL → Full Kratos service generator
│   │   ├── sqlorm/          # SQL → ORM (ent/gorm) generator
│   │   ├── sqlproto/        # SQL → Protobuf/gRPC/REST converter
│   │   └── configexporter/  # Config exporter (Consul/Etcd/Nacos)
│   └── internal/     # CLI-specific code
├── gowind-uiapp/      # Module 2: Wails Desktop UI
│   ├── main.go
│   ├── frontend/     # Vue.js frontend
│   └── internal/     # UI-specific code
└── README.md
```

## Features

| Feature | Description |
|---|---|
| Project Scaffolding | `gow new` — One-click Kratos project creation |
| Service Management | `gow add service` — Add gRPC/REST microservices |
| Code Generation | `gow generate` — Generate CRUD code from database schema |
| Ent / GORM Models | Auto-generate ORM models from SQL database |
| Protobuf Generation | Auto-generate gRPC & REST proto definitions |
| Wire DI | Auto-generate Wire dependency injection |
| Config Exporter | Export configs to Consul / Etcd / Nacos |
| Desktop UI | Visual panel for development & operation |

## Install CLI

```shell
go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest
```

## Quick Start

### Create project

```shell
gow new myproject
cd myproject && go mod tidy
```

### Add service

```shell
gow add service admin -s grpc
```

### Generate CRUD code from database

```shell
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" --service user
```

## Feature Summary

- One-click creation of standard Kratos projects
- One-click addition of multi-protocol microservices (gRPC + REST)
- Database-driven CRUD code generation (proto, ORM, service, server, wire, config)
- Automatic generation of Ent / GORM models
- Automatic generation of Protobuf & API definitions
- Automatic generation of Wire dependency injection
- Config export to Consul / Etcd / Nacos
- Desktop UI (Wails-based visual panel)

