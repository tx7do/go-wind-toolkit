# GoWind CLI (gow)

GoWind CLI (gow) is the core command-line entry of GoWind Toolkit, providing full-lifecycle capabilities such as project scaffolding, microservice management, code generation, and one-click execution, covering the entire process from project creation to development and operation.

**English** | [中文](./README.md)

## Installation

```shell
go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest
```

Verify installation:

```shell
gow version
gow help
```

## Quick Start

### 1. Create a New Project

```shell
# Basic creation
gow new myproject
cd myproject
go mod tidy
```

```shell
# Specify module name
gow new myproject -m github.com/yourusername/myproject
cd myproject
go mod tidy
```

### 2. Add a New Microservice

```shell
# Add basic services
gow add service admin
gow add service user
go mod tidy
```

#### Advanced Options

```shell
# gRPC service
gow add service order -s grpc

# REST service
gow add service admin -s rest

# Support both gRPC + REST
gow add service admin -s rest -s grpc

# Specify ORM (gorm/ent) + gRPC
gow add svc payment -d gorm -s grpc

# Multiple data sources + multiple protocols
gow add service admin -s rest -s grpc -d gorm -d redis
```

### 3. Run the Microservice

```shell
# Run directly in the current directory (must be under app/xxx/service)
gow run
```

```shell
# Run a specified service
gow run admin
```

```shell
# When the working directory is not inside any service, build and run every
# service of the module at once (output is prefixed per service, Ctrl+C stops all)
gow run
```

### 4. Database-Driven Code Generation

Generate complete CRUD microservice code (proto, ORM, service, server, wiring, config) from an existing database. Wiring registration follows the target service: anchor injection into the hand-written wiring.go, or wire provider sets for legacy services:

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

# Using alias
gow gen --dsn "..." --service user
```

#### Generate directly from Go schema sources (no database required)

When the schema already exists as code — an ent schema dir or a gorm model dir — use it as the DSN to generate the same downstream CRUD pipeline:

```shell
# ent schema dir (convention path: internal/data/ent/schema)
gow generate --dsn "ent://internal/data/ent/schema" --service user --orm ent

# gorm model dir: models are parsed and round-tripped through a temp package to emit DAOs (auto-cleaned).
# The target models dir only gets missing files; entries conflicting with hand-written model types are skipped, never overwritten
gow generate --dsn "gorm://internal/data/models" --service payment --orm gorm

# Validate the source and preview resolved tables first
gow generate --dsn "ent://..." --service user -n
```

Notes:
- `ent://` requires `--orm ent`, `gorm://` requires `--orm gorm` (enforced by both the CLI and the generator).
- ent Mixin / GoType / SchemaType overrides cannot be resolved statically; missing fields are reported as `[WARN]`.
- Synthesized m2m join tables follow the same exclusion rules as the live-database pipeline.

### 5. Microservice Evolution (Extract & Split)

Extract business modules from an existing service to another, enabling gradual microservice splitting:

```shell
# Extract role module from admin service to user service
# Target service is auto-created if it doesn't exist
# ORM type is auto-detected from source service directory structure
gow extract admin user --obj role

# Extract multiple objects (comma-separated)
gow extract admin user -o role,permission

# Extract multiple objects (repeated flag)
gow extract admin user --obj role --obj permission

# Keep source files
# gow extract admin user -o role --keep-source
```

### 6. Tool Code Generation


```shell
# Generate Ent for all services
gow ent

# Generate Ent for a specified service
gow ent admin
```

#### Wire Dependency Injection Generation

```shell
# Generate Wire for all services
gow wire

# Generate Wire for a specified service
gow wire admin
```

#### Protobuf / API Code Generation

```shell
# Generate Proto & API for all services
gow api
```

## Full Command Reference

### `gow new` — Project Initialization

```shell
gow new <project-name> [flags]
# or
gow new project <project-name> [flags]

Flags:
  -m, --module string   Go module name (default: project name)
      --no-ci           Skip emitting the GitHub Actions CI workflow
```

> By default `gow new` emits `.github/workflows/ci.yml` at the repo root (setup-go reads the
> version from `go.mod`, chaining download → vet → build → test). An existing file is never
> overwritten; pass `--no-ci` to skip it entirely.

### `gow add` — Add Components

```shell
gow add service <service-name> [flags]

Flags:
  -s, --server strings   Service type: grpc / rest / websocket (multiple selectable)
  -d, --dao strings      Data access layer: gorm / ent / redis (multiple selectable)
  -o, --orm string       ORM type: gorm / ent (default: ent)
```

> The `websocket` transport is message-type driven (`srv.RegisterMessageHandler`) and does not
> register proto services. The generated `internal/server/websocket_server.go` only activates when
> the service's `configs/*.yaml` carries a `server.websocket` section (`network`/`addr`/`path`/`codec`);
> register message handlers after the `register:route` anchor.

### `gow generate` — Database-Driven Code Generation

Generate complete Kratos microservice code (proto, ORM, service, server, wiring, config) from database schema.
Besides live databases and SQL files, Go schema sources are supported as DSN: `ent://<ent schema dir>` and `gorm://<gorm model dir>` (see "Generate directly from Go schema sources" above).

```shell
gow generate [flags]
# or
gow gen [flags]

Flags:
      --dsn string              Data source: e.g. mysql://user:pass@tcp(localhost:3306)/dbname, or ent://<dir> / gorm://<dir>
      --driver string           Database driver: mysql, postgres (default "mysql")
      --service string          Service name (module name)
      --orm string              ORM type: ent, gorm (default "ent")
  -s, --servers strings         Server types: grpc, rest, websocket (default [grpc])
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

> `-s websocket` matches `gow add service`: websocket is a message-driven transport that only
> generates `internal/server/websocket_server.go` (no per-table proto registration) and injects
> `wsServer`/`wsMiddlewares` into `initApp`. It activates once a `server.websocket` section is added
> to the service's `configs/*.yaml`.

### `gow extract` — Microservice Module Extraction

Extract business modules (schema, repo, service, wiring, server) from a source service to a target service, for gradual microservice splitting and evolution. Target service scaffold is auto-created if it doesn't exist, inheriting the source's wiring form. ORM type is auto-detected from source service directory structure.

```shell
gow extract <source-service> <target-service> -o <model> [-o <model>...] [flags]

Flags:
  -o, --obj stringArray   Object/model names to extract (comma-separated or repeated)
      --orm string        ORM type override: ent, gorm (auto-detected by default)
      --keep-source       Keep source files (deleted by default)
```

#### Examples

```shell
# Single object
gow extract admin user -o role

# Multiple objects
gow extract admin user -o role,permission
gow extract admin user --obj role --obj permission

# Manually specify ORM type
gow extract admin user -o role --orm gorm

# Keep source files
# gow extract admin user -o role --keep-source
```

### `gow run` — Run Service

```shell
# Run the named service, or the service containing the working directory;
# otherwise build and run every service of the module at once, with per-service
# prefixed output; Ctrl+C stops all of them.
gow run [service-name]
```

### `gow build` — Build Services (cross-compilation supported)

```shell
# Build every service (output goes to each service's bin/ directory by default)
gow build

# Build selected services
gow build admin

# Cross compilation: every GOOS×GOARCH combination, binaries carry a
# _<goos>_<goarch> suffix; --out redirects all output into one directory
gow build --os linux,windows --arch amd64 --out ./dist
```

### `gow ent` — Ent Code Generation

```shell
gow ent [service-name]
```

### `gow wire` — Wire Code Generation (legacy services only; hand-wired services are skipped)

```shell
gow wire [service-name]
```

### `gow api` — Protobuf / API Code Generation

```shell
gow api
```

### gow version — Check Version

```shell
gow version
```

### gow help — Help

```shell
gow help
gow help <command>
```

### Project Structure \(After Generation\)

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

## Feature Summary

- ✅ One-click creation of standard Kratos projects
- ✅ One-click addition of multi-protocol microservices (gRPC + REST)
- ✅ Database-driven CRUD code generation (proto, ORM, service, server, wiring, config)
- ✅ Automatic generation of Ent / GORM models
- ✅ Automatic generation of Protobuf & API definitions
- ✅ Automatic generation of Wire dependency injection
- ✅ Gradual microservice splitting and evolution (module extraction)
- ✅ One-click execution and hot-reload support
- ✅ Unified CLI entry to reduce learning costs
- ✅ Desktop UI visual panel (Wails)
