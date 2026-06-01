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

### 4. Database-Driven Code Generation

Generate complete CRUD microservice code (proto, ORM, service, server, wire, config) from an existing database:

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
```

### `gow add` — Add Components

```shell
gow add service <service-name> [flags]

Flags:
  -s, --server strings   Service type: grpc / rest (multiple selectable)
  -d, --dao strings      Data access layer: gorm / ent / redis (multiple selectable)
  -o, --orm string       ORM type: gorm / ent (default: ent)
```

### `gow generate` — Database-Driven Code Generation

Generate complete Kratos microservice code (proto, ORM, service, server, wire, config) from database schema.

```shell
gow generate [flags]
# or
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

### `gow extract` — Microservice Module Extraction

Extract business modules (schema, repo, service, wire, server) from a source service to a target service, for gradual microservice splitting and evolution. Target service scaffold is auto-created if it doesn't exist. ORM type is auto-detected from source service directory structure.

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
gow run [service-name]
```

### `gow ent` — Ent Code Generation

```shell
gow ent [service-name]
```

### `gow wire` — Wire Code Generation

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
- ✅ Database-driven CRUD code generation (proto, ORM, service, server, wire, config)
- ✅ Automatic generation of Ent / GORM models
- ✅ Automatic generation of Protobuf & API definitions
- ✅ Automatic generation of Wire dependency injection
- ✅ Gradual microservice splitting and evolution (module extraction)
- ✅ One-click execution and hot-reload support
- ✅ Unified CLI entry to reduce learning costs
- ✅ Desktop UI visual panel (Wails)
