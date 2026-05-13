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

### 4. Code Generation

#### Ent Code Generation

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

- ✅ One\-click creation of standard Kratos projects
- ✅ One\-click addition of multi\-protocol microservices \(gRPC \+ REST\)
- ✅ Automatic generation of Ent / GORM models
- ✅ Automatic generation of Protobuf \&amp; API definitions
- ✅ Automatic generation of Wire dependency injection
- ✅ One\-click execution and hot\-reload support
- ✅ Unified CLI entry to reduce learning costs
