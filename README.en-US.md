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

Complete the full-flow generation from database schema to complete microservice code through a wizard-style interface. Supports four schema import methods: direct database connection (MySQL, PostgreSQL), SQL file, remote URL, and online editor. Assign each table to its microservice, configure proto package strategy (per-table / by-service / custom), select ORM type (Ent / GORM), and generate gRPC or REST service code with one click, automatically running post-processing (`go mod tidy` → `buf generate` → `ent generate` → `wire generate`).

> Supported sources are MySQL / PostgreSQL connection strings, DDL text (SQL file / remote URL / online editor) and `ent://<dir>` / `gorm://<dir>` Go source directories. SQLite and Oracle connections cannot be used as generation sources — for SQLite, export DDL first with `sqlite3 <file> .schema` and import it as a SQL file.

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

# Preview the service layout without creating anything
gow add service admin -s grpc --dry-run
```

### Run a Service

```shell
# Run directly in the service directory
gow run

# Run a specified service
gow run admin

# Run every service in the module when no name is given
gow run
```

### Hot Reload (--watch)

```shell
# Watch mode: saving a file rebuilds and restarts only the affected services
gow run admin --watch

# Watch every service (output prefixed with service names)
gow run --watch

# Or run inside a service directory
cd app/admin/service && gow run -w
```

How watch mode works:

- Recursively watches the module root (skipping `.git`, `vendor`, `node_modules`, `bin`, hidden dirs); changes to `.go`/`.yaml`/`.yml`/`.json`/`.toml`/`.properties`/`.proto` trigger rebuild & restart, `*_test.go` and hidden files are ignored
- Only **affected services** restart: a change inside one service's directory restarts just that service; module-level shared code restarts all
- Consecutive saves are merged with a 500ms debounce; a **failed rebuild keeps the old process running**, and the next save retries automatically
- Shutdown sends SIGTERM first (killed after a 5s grace period); `.proto` changes only trigger a restart — run `gow api` first to regenerate code

### Build Services

```shell
# Build every service into each service's bin/ directory
gow build

# Build specific services
gow build admin user

# Cross compile (full GOOS×GOARCH matrix, binaries carry a _<goos>_<goarch> suffix)
gow build --os linux,windows --arch amd64,arm64

# Stamp a version and produce slim binaries
gow build --version v1.2.3 --strip --trimpath

# Custom ldflags and output directory
gow build -o ./dist --ldflags "-X main.commit=$(git rev-parse --short HEAD)"
```

`--version` injects the `version` variable of each service's `main` package via `-ldflags "-X main.version=..."` (the scaffold template generates it by default).

### Check Version

```shell
gow version

# Release builds inject it via ldflags (already wired into the release workflow):
# go build -ldflags "-X main.version=v1.2.3 -X main.commit=abc1234 -X main.date=..."
gow version
# gow version v1.2.3 (commit: abc1234, built: 2026-09-12T08:00:00Z)
```

Binaries installed via `go install` fall back to the module version; local source builds show `dev`.

### Generate CRUD Code from Database

```shell
# Interactive (prompts for DSN and service name)
gow generate

# Validate the data source, resolve tables and preview the plan (nothing written;
# works with a database DSN or inline DDL text)
gow generate --dsn "mysql://user:pass@tcp(localhost:3306)/dbname" --service user --dry-run

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

# Explicit subcommand form (equivalent)
gow ent generate admin

# Add schema(s) to a service and regenerate Ent code automatically
gow ent add admin Role,Permission
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

### Reverse: Dump ent Schema to DDL (migrate)

```shell
# Dump DDL for every service with an ent schema (mysql dialect by default)
gow migrate

# Pick a service and dialect (mysql / postgres / sqlite)
gow migrate admin --dialect postgres

# Collect all files into one directory (named <service>.<dialect>.sql)
gow migrate --dialect sqlite -o ./dist/sql

# Target a specific database version for dialect-specific syntax
gow migrate admin --dialect mysql --db-version 5.7
```

`gow migrate` renders the ent schema under `app/<service>/service/internal/data/ent`
into a `CREATE TABLE` script (planned offline via ent's `schema.DDL` — **no
database connection needed**), written to `migrations/schema.<dialect>.sql` of
each service by default. Missing ent codegen is filled in automatically (as
`gow ent` does); non-ent (gorm) services are skipped. The output works as an
Atlas baseline migration or for reviewing schema state.

### Microservice Evolution (Module Extraction)

```shell
# Extract role module from admin service to user service
# Target service is auto-created if it doesn't exist, ORM type auto-detected
gow extract admin user -o role

# Extract multiple entities
gow extract admin user -o role,permission

# Manually specify ORM type
gow extract admin user -o role --orm gorm

# Preview all file actions (copy/modify/delete) without touching anything
gow extract admin user -o role --dry-run

# Keep source files (deleted by default)
gow extract admin user -o role --keep-source

# Skip the deletion confirmation prompt (for scripts/CI)
gow extract admin user -o role --yes
```

Extraction **deletes source files by default**: the full plan (copies, in-place
modifications, deletions) is printed first, and the deletion requires an
interactive confirmation (defaults to no). `--dry-run` previews with zero
changes, `--keep-source` is non-destructive and skips the prompt, `--yes`
skips the prompt for scripts.

## Feature Summary

- One-click creation of standard Kratos projects
- One-click addition of multi-protocol microservices (gRPC + REST)
- Hot-reload service runner and cross compilation (`gow run --watch` / `gow build`)
- Database-driven CRUD code generation (proto, ORM, service, server, wire, config)
- Automatic Ent / GORM model generation
- Reverse: dump ent schema to DDL (`gow migrate`, offline, no database needed)
- Automatic Protobuf & API definition generation
- Automatic Wire dependency injection generation
- Gradual microservice splitting and evolution (module extraction)
- Config export to Consul / Etcd / Nacos
- Desktop UI visual panel (Wails)
