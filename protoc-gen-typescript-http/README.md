# protoc-gen-typescript-http

[English](./README.en.md) | [日本語](./README.ja.md) | 简体中文

从带有 [HTTP 规则](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto) 注解的 Protobuf 定义生成 TypeScript 类型和服务客户端。生成的类型遵循 [Protobuf canonical JSON 编码](https://developers.google.com/protocol-buffers/docs/proto3#json) 规范。


## 功能特性

- 从 `.proto` 文件生成 TypeScript 类型和类型安全的服务客户端
- 支持 Unary（一元）、Server-Streaming（服务端流式）、Bidirectional-Streaming（双向流式）RPC
- 生成代码只输出**纯接口**，HTTP / SSE / WebSocket 传输实现完全由调用方注入
- 支持 `google.api.default_host` 自动生成 `DEFAULT_HOST` 常量
- 遵循 Protobuf canonical JSON 编码规范

## 安装

```bash
go install github.com/go-kratos/protoc-gen-typescript-http@latest
```

或从 [releases](./releases) 下载预编译二进制文件。

## 调用方式

```bash
protoc \
  --typescript-http_out [输出目录] \
  [.proto 文件 ...]
```

---

## 架构设计

生成代码定义了一个 `ClientTransport` 接口，包含三个方法：

| 方法 | 用途 | RPC 类型 |
| --- | --- | --- |
| `unary()` | 普通请求/响应 | Unary RPC |
| `serverStream()` | 服务端流式推送（返回 `ServerStream<T>`） | `returns (stream T)` |
| `duplexStream()` | 双向流式通信（返回 `DuplexStream<TIn, TOut>`） | `stream T returns (stream U)` |

**调用方负责提供 `ClientTransport` 的实现**。这样你可以完全掌控 HTTP 客户端（fetch、Axios 等）、认证头、SSE 传输方式和 WebSocket 传输方式。

完整示例请参考 [examples](./examples)。

## 基本用法

实现 `ClientTransport` 接口，传入生成的客户端工厂函数：

```typescript
import { ClientTransport, createShipperServiceClient, DEFAULT_HOST } from "./gen";

const transport: ClientTransport = {
  unary(path, method, body, _meta) {
    return fetch(`https://${DEFAULT_HOST}/${path}`, {
      method,
      body: body ?? undefined,
      headers: { Authorization: "Bearer token" },
    }).then((r) => r.json());
  },
  serverStream(_path, _meta) {
    throw new Error("not implemented");
  },
  duplexStream(_path, _meta) {
    throw new Error("not implemented");
  },
};

const client = createShipperServiceClient(transport);

// Unary 调用
const shipper = await client.GetShipper({ name: "shippers/123" });
```

## 使用 google.api.default_host

如果 proto service 定义了 `google.api.default_host` 选项，会自动生成 `DEFAULT_HOST` 常量：

```protobuf
service ShipperService {
  option (google.api.default_host) = "api.example.com";
  // ...
}
```

该常量会被导出，可在构建 transport 时引用：

```typescript
import { DEFAULT_HOST, createShipperServiceClient } from "./gen";

const baseUrl = `https://${DEFAULT_HOST}`;
```

## 流式通信

服务端流式 RPC（`returns (stream ...)`）和双向流式 RPC（`stream ... returns (stream ...)`）分别通过 `ClientTransport` 的 `serverStream()` 和 `duplexStream()` 方法支持。

生成代码**只定义 `ServerStream<T>` 和 `DuplexStream<TIn, TOut>` 接口**——实际的传输实现（基于 `fetch` + `ReadableStream` 的 SSE、`EventSource`、WebSocket 等）由调用方提供。

### 示例 Proto

```protobuf
service LogService {
  rpc GetLog(GetLogRequest) returns (GetLogResponse) {
    option (google.api.http) = {get: "/v1/{name=logs/*}"};
  }

  // 服务端流式
  rpc TailLogs(TailLogsRequest) returns (stream LogEntry) {
    option (google.api.http) = {get: "/v1/{name=logs/*}:tail"};
  }

  // 双向流式
  rpc Chat(stream ChatMessage) returns (stream ChatMessage) {
    option (google.api.http) = {get: "/v1/chat"};
  }
}
```

### 实现 ServerStream（基于 fetch + ReadableStream 的 SSE）

```typescript
import { ServerStream } from "./gen";

class FetchSSETransport<T> implements ServerStream<T> {
  private listeners: Array<(data: T) => void> = [];
  private errorHandlers: Array<(error: Error) => void> = [];
  private controller?: AbortController;

  constructor(url: string) {
    this.controller = new AbortController();
    fetch(url, { signal: this.controller.signal })
      .then(async (response) => {
        const reader = response.body!.getReader();
        const decoder = new TextDecoder();
        let buffer = "";
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop()!;
          for (const line of lines) {
            if (line.startsWith("data: ")) {
              try {
                const data = JSON.parse(line.slice(6)) as T;
                this.listeners.forEach((fn) => fn(data));
              } catch (e) {
                this.errorHandlers.forEach((fn) => fn(e as Error));
              }
            }
          }
        }
      })
      .catch((err) => {
        this.errorHandlers.forEach((fn) => fn(err));
      });
  }

  onEvent(listener: (data: T) => void): () => void {
    this.listeners.push(listener);
    return () => {
      this.listeners = this.listeners.filter((fn) => fn !== listener);
    };
  }

  onError(handler: (error: Error) => void): void {
    this.errorHandlers.push(handler);
  }

  close(): void {
    this.controller?.abort();
  }
}
```

### 传入 transport 并使用

```typescript
const transport: ClientTransport = {
  // ...unary 实现...
  serverStream<T>(path, _meta) {
    return new FetchSSETransport<T>(`https://api.example.com/${path}`);
  },
  duplexStream<TIn, TOut>(path, _meta) {
    // 返回基于 WebSocket 的 DuplexStream 实现
  },
};

const client = createLogServiceClient(transport);

// 服务端流式
const tail = client.TailLogs({ name: "log/123" });
tail.onEvent((entry) => console.log(entry.message));
tail.onError((err) => console.error(err));
// tail.close();

// 双向流式
const chat = client.Chat();
chat.onEvent((msg) => console.log(msg.text));
chat.send({ text: "hello" });
// chat.close();
```

## 统一 ApiClient

当一个 proto package 包含多个 service 时，会生成一个聚合所有服务客户端的 `ApiClient` 类。只需传入一次 transport：

```typescript
import { ApiClient, ClientTransport } from "./gen";

const transport: ClientTransport = { /* ... */ };
const api = new ApiClient(transport);
// 或：const api = createApiClient(transport);

// 按需懒加载访问各个服务
const shipper = await api.shipperService.GetShipper({ name: "shippers/123" });
```

## 许可证

[MIT](./LICENSE)
