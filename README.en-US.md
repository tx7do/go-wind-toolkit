# GoWind Toolkit

**English** | [中文](./README.md)

**A comprehensive all-in-one toolkit for Go-Kratos microservice development, including scaffolding, CRUD code generation,
dev tools, operation utilities, CLI and desktop UI.**

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
│   │   ├── service/         # Service scaffold generator
│   │   ├── extract/         # Microservice module extractor
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
| Microservice Evolution | `gow extract` — Gradual service splitting & module extraction |
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

### Microservice Evolution (Extract & Split)

```shell
# Extract role module from admin service to user service
# Target service is auto-created if it doesn't exist
# ORM type is auto-detected from source service
gow extract admin user -o role

# Extract multiple objects
gow extract admin user -o role,permission
```

## Feature Summary

- One-click creation of standard Kratos projects
- One-click addition of multi-protocol microservices (gRPC + REST)
- Database-driven CRUD code generation (proto, ORM, service, server, wire, config)
- Automatic generation of Ent / GORM models
- Automatic generation of Protobuf & API definitions
- Automatic generation of Wire dependency injection
- Gradual microservice splitting and evolution (module extraction)
- Config export to Consul / Etcd / Nacos
- Desktop UI (Wails-based visual panel)
