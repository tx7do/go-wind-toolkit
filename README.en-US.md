# GoWind Toolkit

**English** | [中文](./README.md)

An **all-in-one toolkit** for the Go-Kratos microservice ecosystem, featuring scaffolding, automated code generation, development aids, ops tools, a CLI, and a visual desktop client.

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
│   │   ├── service/         # Service scaffolding generator
│   │   ├── extract/         # Microservice module extractor
│   │   └── configexporter/  # Config exporter (Consul/Etcd/Nacos)
│   └── internal/     # CLI-specific code
├── gowind-uiapp/      # Module 2: Wails Desktop UI
│   ├── main.go
│   ├── frontend/     # Vue.js frontend
│   └── internal/     # UI-specific code
└── README.md
```

## Feature Overview

| Feature | CLI | Desktop UI |
|---|:---:|:---:|
| Project scaffolding (`gow new`) | ✅ | ✅ |
| Add microservice (`gow add service`) | ✅ | ✅ |
| Database-driven CRUD code generation | ✅ | ✅ |
| Ent / GORM model generation | ✅ | ✅ |
| Protobuf gRPC & REST definition generation | ✅ | ✅ |
| Wire dependency injection generation | ✅ | ✅ |
| Microservice module extraction (`gow extract`) | ✅ | — |
| Config export to Consul / Etcd / Nacos | — | ✅ |
| Visual table configuration & service assignment | — | ✅ |
| AI-assisted DDL generation & microservice partitioning | — | ✅ |
| AI code review | — | ✅ |
| Service start/stop management | — | ✅ |
| Dev tools (buf/wire/ent) | — | ✅ |

## Desktop Client — gowind-uiapp

A cross-platform desktop application built with [Wails](https://wails.io/) (Go + Vue 3), providing visual all-in-one tooling for Go-Kratos microservice development.

### Backend Code Generation

Complete the full-flow generation from database schema to complete microservice code through a wizard-style interface. Supports four schema import methods: direct database connection (MySQL, PostgreSQL, SQLite, Oracle), SQL file, remote URL, and online editor. Assign each table to its microservice, configure proto package strategy (per-table / by-service / custom), select ORM type (Ent / GORM), and generate gRPC or REST service code with one click, automatically running post-processing (`go mod tidy` → `buf generate` → `ent generate` → `wire generate`).

### Frontend Code Generation

Automatically generate complete frontend admin page code based on OpenAPI definitions, supporting three popular frontend frameworks:

- **Vue 3 + Element Plus**: API call layer, Vue Query Composable, list page (table + search + pagination), edit drawer, router config, i18n locale files
- **Vue 3 + Vben Admin** (VxeGrid + useVbenDrawer): API call layer, Composable, list page, edit drawer, router config, i18n locales (page + menu)
- **React + Ant Design Pro** (ProTable + DrawerForm): API call layer, React Query Hooks, list page, edit drawer, router config, i18n locale files

### Remote Config Export

One-click export of local configuration files to Consul, Etcd, or Nacos, with support for batch export and service-level selective export.

### AI Assistant

Integrates multiple LLM providers (OpenAI, DeepSeek, Ollama, etc.), supporting DDL generation from natural language requirements, AI-assisted microservice partitioning, and code review to accelerate development decisions.

### Dev Tools

Built-in `buf generate`, `ent generate`, `wire generate`, `go mod tidy`, service start/stop management, and other common development commands — no need to switch terminals.

## Install CLI

```shell
go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest
```

## Quick Start

### Create a Project

```shell
gow new myproject
cd myproject && go mod tidy
```

### Add a Service

```shell
# Add a gRPC service
gow add service admin -s grpc

# Add a REST service
gow add service admin -s rest

# Support both gRPC + REST
gow add service admin -s rest -s grpc

# Specify ORM (gorm / ent)
gow add service admin -d gorm -s grpc
```

### Run a Service

```shell
# Run directly in the service directory
gow run

# Run a specified service
gow run admin
```

### Generate CRUD Code from Database

```shell
# Interactive (prompts for DSN and service name)
gow generate

# Full command line
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" --service user

# Specify ORM and tables
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" \
  --service user --orm ent --servers grpc --tables users,roles

# Generate proto files only
gow generate --dsn "postgres://user:pass@localhost:5432/dbname" --service admin --proto-only

# Generate REST service (proxying from gRPC service)
gow generate --dsn "mysql://..." --service user-admin \
  --servers rest --source-module user --skip-orm
```

### Ent Code Generation

```shell
# Generate Ent for all services
gow ent

# Generate Ent for a specified service
gow ent admin
```

### Wire Dependency Injection Generation

```shell
# Generate Wire for all services
gow wire

# Generate Wire for a specified service
gow wire admin
```

### Protobuf / API Code Generation

```shell
# Generate Proto & API for all services
gow api
```

### Microservice Evolution (Module Extraction)

```shell
# Extract role module from admin service to user service
# Target service is auto-created if it doesn't exist, ORM type auto-detected
gow extract admin user -o role

# Extract multiple entities
gow extract admin user -o role,permission

# Manually specify ORM type
gow extract admin user -o role --orm gorm

# Keep source files (deleted by default)
gow extract admin user -o role --keep-source
```

### Check Version

```shell
gow version
```

## Feature Summary

- One-click creation of standard Kratos projects
- One-click addition of multi-protocol microservices (gRPC + REST)
- Database-driven CRUD code generation (proto, ORM, service, server, wire, config)
- Automatic Ent / GORM model generation
- Automatic Protobuf & API definition generation
- Automatic Wire dependency injection generation
- Gradual microservice splitting and evolution (module extraction)
- Config export to Consul / Etcd / Nacos
- Desktop UI visual panel (Wails)
