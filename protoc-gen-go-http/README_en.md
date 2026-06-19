# protoc-gen-go-http

[中文](README.md) | [日本語](README_ja.md)

`protoc-gen-go-http` is a [protoc](https://github.com/protocolbuffers/protobuf) plugin that generates Go HTTP server code (a gRPC HTTP gateway) for Protobuf services based on the [`google.api.http`](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto) annotation.

The generated code is built on the standard library `net/http` and uses the [`go-wind-toolkit`](https://github.com/tx7do/go-wind-toolkit) `transport/http/binding` package for request binding, route registration and response writing.

## Features

- Automatic route generation from `google.api.http` annotations
- Supports `GET` / `POST` / `PUT` / `DELETE` / `PATCH` / custom methods
- Supports `additional_bindings` for multiple route bindings
- Supports path variable binding (including nested fields, e.g. `{message.id}`)
- Supports request body binding (`body: "*"` or a specific field)
- Supports automatic query parameter binding
- Supports the `google.api.HttpBody` type
- Supports `response_body` response field mapping
- Provides default routes for methods without HTTP annotations (configurable)
- Generated code is framework-agnostic, built on the standard library `net/http`

## Installation

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-go-http@latest
```

> Requires Go 1.25+ and the `protoc` compiler to be installed.

## Quick Start

### 1. Write a proto file

```protobuf
syntax = "proto3";

package helloworld;

import "google/api/annotations.proto";

service Greeter {
  rpc SayHello(HelloRequest) returns (HelloReply) {
    option (google.api.http) = {
      get: "/helloworld/{name}"
    };
  }

  rpc CreateHello(CreateHelloRequest) returns (HelloReply) {
    option (google.api.http) = {
      post: "/helloworld"
      body: "*"
    };
  }
}

message HelloRequest {
  string name = 1;
}

message CreateHelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

### 2. Generate code

```bash
protoc \
  --proto_path=./proto \
  --go_out=. --go_opt=paths=source_relative \
  --go-http_out=. --go-http_opt=paths=source_relative \
  proto/helloworld/helloworld.proto
```

> Both `protoc-gen-go` and `protoc-gen-go-http` must be installed.

### 3. Generated code structure

The output file is named `xxx_http.pb.go` and contains:

- **HTTP server interface**: `GreeterHTTPServer`, defining the business method signatures
- **Registration function**: `RegisterGreeterHTTPServer`, registering routes onto a `binding.Router`
- **Per-method handlers**: `_Greeter_XXX_HTTP_Handler`, performing binding and response writing

```go
// Generated interface
type GreeterHTTPServer interface {
    SayHello(context.Context, *HelloRequest) (*HelloReply, error)
    CreateHello(context.Context, *CreateHelloRequest) (*HelloReply, error)
}

// Registration function
func RegisterGreeterHTTPServer(srv binding.Router, svc GreeterHTTPServer) {
    srv.Handle("GET", "/helloworld/{name}", _Greeter_SayHello0_HTTP_Handler(svc))
    srv.Handle("POST", "/helloworld", _Greeter_CreateHello0_HTTP_Handler(svc))
}
```

### 4. Implement the business logic

```go
type greeterServer struct{}

func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello " + req.Name}, nil
}

func (s *greeterServer) CreateHello(ctx context.Context, req *pb.CreateHelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Created " + req.Name}, nil
}
```

### 5. Register and start the server

```go
func main() {
    mux := http.NewServeMux()
    router := binding.NewRouter(mux) // Use the go-wind-toolkit Router
    pb.RegisterGreeterHTTPServer(router, &greeterServer{})
    http.ListenAndServe(":8080", mux)
}
```

## Using Buf

In addition to using `protoc` directly, you can also use [Buf](https://buf.build) to manage code generation. Buf provides a cleaner configuration approach along with dependency management.

### 1. Configure `buf.gen.yaml`

Create a `buf.gen.yaml` file in the project root:

```yaml
version: v2
plugins:
  - local: protoc-gen-go
    out: .
    opt: paths=source_relative
  - local: protoc-gen-go-http
    out: .
    opt:
      - paths=source_relative
      - omitempty=false
      - omitempty_prefix=/api/v1
```

> If you do not need default routes for methods without HTTP annotations, the `omitempty` and `omitempty_prefix` options can be omitted.

### 2. Run code generation

```bash
buf generate
```

For more information about using Buf, please refer to the [Buf documentation](https://buf.build/docs).

## Command-line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `version` | `false` | Print the plugin version and exit |
| `omitempty` | `true` | Skip generation when none of the services contain a `google.api.http` annotation |
| `omitempty_prefix` | `""` | Path prefix used when generating default routes for methods without an HTTP annotation |

### Usage examples

```bash
# Skip files without annotations
protoc --go-http_out=. --go-http_opt=omitempty=true proto/...

# Generate default routes for methods without annotations, prefix /api/v1
protoc --go-http_out=. --go-http_opt=omitempty=false,omitempty_prefix=/api/v1 proto/...
```

## Supported HTTP Annotations

### HTTP methods

```protobuf
option (google.api.http) = {
  get: "/v1/users/{id}"
};
option (google.api.http) = {
  post: "/v1/users"
  body: "*"
};
option (google.api.http) = {
  put: "/v1/users/{id}"
  body: "user"
};
option (google.api.http) = {
  delete: "/v1/users/{id}"
};
option (google.api.http) = {
  patch: "/v1/users/{id}"
  body: "*"
};
```

### Path variables

Nested field path variables are supported:

```protobuf
option (google.api.http) = {
  get: "/v1/{message.id=messages/*}"
};
```

`{message.id=messages/*}` is converted to the route `/v1/{message.id:messages/[^/]+}` and automatically bound to the `message.id` field of the request message.

### Multiple route bindings

```protobuf
rpc ListUsers(ListUsersRequest) returns (ListUsersReply) {
  option (google.api.http) = {
    get: "/v1/users"
    additional_bindings {
      get: "/v1/groups/{group_id}/users"
    }
  };
}
```

### Response body mapping

```protobuf
rpc DownloadFile(DownloadRequest) returns (DownloadReply) {
  option (google.api.http) = {
    get: "/v1/files/{name}"
    response_body: "data"
  };
}
```

The generated code writes `out.Data` instead of the entire `out`.

## Project Structure

```
protoc-gen-go-http/
├── main.go              # Plugin entry point, parses flags and drives generation
├── http.go              # Core generation logic: parses HTTP annotations, builds method descriptors
├── template.go          # Template data structures (serviceDesc / methodDesc)
├── httpTemplate.tpl     # Code generation template
├── version.go           # Version definition
├── http_test.go         # Unit tests
├── go.mod               # Go module definition
└── go.sum               # Dependency verification
```

## Tech Stack

| Component | Version |
|-----------|---------|
| Go | 1.25+ |
| `google.golang.org/protobuf` | v1.36.11 |
| `google.golang.org/genproto/googleapis/api` | latest |

## Development & Testing

```bash
# Build the plugin
go build -o protoc-gen-go-http .

# Run tests
go test .

# Run static analysis
go vet .
```

## License

Please refer to the parent repository [go-wind-toolkit](https://github.com/tx7do/go-wind-toolkit) for license information.
