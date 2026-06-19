# protoc-gen-go-http

[English](README_en.md) | [日本語](README_ja.md)

`protoc-gen-go-http` 是一个 [protoc](https://github.com/protocolbuffers/protobuf) 插件，它根据 [`google.api.http`](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto) 注解，为 Protobuf 服务生成 Go HTTP 服务端代码（gRPC HTTP 网关）。

生成的代码基于标准库 `net/http`，并通过 [`go-wind-toolkit`](https://github.com/tx7do/go-wind-toolkit) 的 `transport/http/binding` 包完成请求参数绑定、路由注册与统一响应输出。

## 特性

- 根据 `google.api.http` 注解自动生成路由
- 支持 `GET` / `POST` / `PUT` / `DELETE` / `PATCH` / 自定义方法
- 支持 `additional_bindings` 多路由绑定
- 支持路径变量绑定（含嵌套字段，如 `{message.id}`）
- 支持请求体绑定（`body: "*"` 或指定字段）
- 支持查询参数自动绑定
- 支持 `google.api.HttpBody` 类型
- 支持 `response_body` 响应字段映射
- 为无 HTTP 注解的方法提供默认路由（可通过参数配置）
- 生成代码不依赖特定 Web 框架，基于标准库 `net/http`

## 安装

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-go-http@latest
```

> 要求 Go 1.25+，并已安装 `protoc` 编译器。

## 快速开始

### 1. 编写 proto 文件

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

### 2. 生成代码

```bash
protoc \
  --proto_path=./proto \
  --go_out=. --go_opt=paths=source_relative \
  --go-http_out=. --go-http_opt=paths=source_relative \
  proto/helloworld/helloworld.proto
```

> 上述命令需要同时安装 `protoc-gen-go` 与 `protoc-gen-go-http`。

### 3. 生成的代码结构

生成文件名为 `xxx_http.pb.go`，包含以下内容：

- **HTTP 服务接口**：`GreeterHTTPServer`，定义业务方法签名
- **注册函数**：`RegisterGreeterHTTPServer`，将路由注册到 `binding.Router`
- **各方法处理器**：`_Greeter_XXX_HTTP_Handler`，完成参数绑定与响应输出

```go
// 生成的接口
type GreeterHTTPServer interface {
    SayHello(context.Context, *HelloRequest) (*HelloReply, error)
    CreateHello(context.Context, *CreateHelloRequest) (*HelloReply, error)
}

// 注册函数
func RegisterGreeterHTTPServer(srv binding.Router, svc GreeterHTTPServer) {
    srv.Handle("GET", "/helloworld/{name}", _Greeter_SayHello0_HTTP_Handler(svc))
    srv.Handle("POST", "/helloworld", _Greeter_CreateHello0_HTTP_Handler(svc))
}
```

### 4. 实现业务逻辑

```go
type greeterServer struct{}

func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello " + req.Name}, nil
}

func (s *greeterServer) CreateHello(ctx context.Context, req *pb.CreateHelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Created " + req.Name}, nil
}
```

### 5. 注册并启动服务

```go
func main() {
    mux := http.NewServeMux()
    router := binding.NewRouter(mux) // 使用 go-wind-toolkit 的 Router
    pb.RegisterGreeterHTTPServer(router, &greeterServer{})
    http.ListenAndServe(":8080", mux)
}
```

## 使用 Buf

除了直接使用 `protoc`，你也可以使用 [Buf](https://buf.build) 来管理代码生成。Buf 提供了更简洁的配置方式和依赖管理能力。

### 1. 配置 `buf.gen.yaml`

在项目根目录下创建 `buf.gen.yaml` 文件：

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

> 如果不需要为无 HTTP 注解的方法生成默认路由，可省略 `omitempty` 与 `omitempty_prefix` 选项。

### 2. 执行代码生成

```bash
buf generate
```

更多关于 Buf 的使用方法，请参考 [Buf 官方文档](https://buf.build/docs)。

## 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `version` | `false` | 打印插件版本号后退出 |
| `omitempty` | `true` | 当文件中的 Service 均不含 `google.api.http` 注解时，跳过生成 |
| `omitempty_prefix` | `""` | 为无 HTTP 注解的方法生成默认路由时的路径前缀 |

### 用法示例

```bash
# 跳过无注解的文件
protoc --go-http_out=. --go-http_opt=omitempty=true proto/...

# 为无注解的方法生成默认路由，前缀为 /api/v1
protoc --go-http_out=. --go-http_opt=omitempty=false,omitempty_prefix=/api/v1 proto/...
```

## 支持的 HTTP 注解

### HTTP 方法

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

### 路径变量

支持嵌套字段路径变量：

```protobuf
option (google.api.http) = {
  get: "/v1/{message.id=messages/*}"
};
```

`{message.id=messages/*}` 会被转换为路由 `/v1/{message.id:messages/[^/]+}`，并自动绑定到请求消息的 `message.id` 字段。

### 多路由绑定

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

### 响应体映射

```protobuf
rpc DownloadFile(DownloadRequest) returns (DownloadReply) {
  option (google.api.http) = {
    get: "/v1/files/{name}"
    response_body: "data"
  };
}
```

生成代码会输出 `out.Data` 而非整个 `out`。

## 项目结构

```
protoc-gen-go-http/
├── main.go              # 插件入口，解析参数并驱动生成流程
├── http.go              # 核心生成逻辑：解析 HTTP 注解、构建方法描述
├── template.go          # 模板数据结构（serviceDesc / methodDesc）
├── httpTemplate.tpl     # 代码生成模板
├── version.go           # 版本号定义
├── http_test.go         # 单元测试
├── go.mod               # Go 模块定义
└── go.sum               # 依赖校验
```

## 技术栈

| 项目 | 版本 |
|------|------|
| Go | 1.25+ |
| `google.golang.org/protobuf` | v1.36.11 |
| `google.golang.org/genproto/googleapis/api` | latest |

## 开发与测试

```bash
# 编译插件
go build -o protoc-gen-go-http .

# 运行测试
go test .

# 运行静态检查
go vet .
```

## 许可证

请参考上级仓库 [go-wind-toolkit](https://github.com/tx7do/go-wind-toolkit) 的许可证信息。
