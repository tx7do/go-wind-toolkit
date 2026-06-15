# protoc-gen-typescript-http

[简体中文](./README.md) | [日本語](./README.ja.md) | English

Generates TypeScript types and service clients from Protobuf definitions annotated with [HTTP rules](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto). The generated types follow the [canonical JSON encoding](https://developers.google.com/protocol-buffers/docs/proto3#json) specification.

## Features

- Generate TypeScript types and type-safe service clients from `.proto` files
- Support for Unary, Server-Streaming, and Bidirectional-Streaming RPCs
- Generated code outputs **pure interfaces only** — HTTP / SSE / WebSocket transport implementations are fully injected by the caller
- Support for `google.api.default_host` to auto-generate a `DEFAULT_HOST` constant
- Follows Protobuf canonical JSON encoding

## Installation

```bash
go install github.com/go-kratos/protoc-gen-typescript-http@latest
```

Or download a prebuilt binary from [releases](./releases).

## Invocation

```bash
protoc \
  --typescript-http_out [OUTPUT_DIR] \
  [.proto files ...]
```

---

## Architecture

The generated code defines a `ClientTransport` interface with three methods:

| Method | Purpose | RPC Type |
| --- | --- | --- |
| `unary()` | Standard request/response | Unary RPC |
| `serverStream()` | Server-streaming push (returns `ServerStream<T>`) | `returns (stream T)` |
| `duplexStream()` | Bidirectional streaming (returns `DuplexStream<TIn, TOut>`) | `stream T returns (stream U)` |

**The caller is responsible for providing a `ClientTransport` implementation.** This gives you full control over the HTTP client (fetch, Axios, etc.), authentication headers, SSE transport, and WebSocket transport.

For complete examples, see [examples](./examples).

## Basic Usage

Implement the `ClientTransport` interface and pass it to the generated client factory:

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

// Unary call
const shipper = await client.GetShipper({ name: "shippers/123" });
```

## With google.api.default_host

If your proto service defines the `google.api.default_host` option, a `DEFAULT_HOST` constant is generated automatically:

```protobuf
service ShipperService {
  option (google.api.default_host) = "api.example.com";
  // ...
}
```

The constant is exported so you can reference it when building your transport:

```typescript
import { DEFAULT_HOST, createShipperServiceClient } from "./gen";

const baseUrl = `https://${DEFAULT_HOST}`;
```

## Streaming

Server-streaming RPCs (`returns (stream ...)`) and bidirectional streaming RPCs (`stream ... returns (stream ...)`) are supported through the `serverStream()` and `duplexStream()` methods on `ClientTransport`.

The generated code **only defines `ServerStream<T>` and `DuplexStream<TIn, TOut>` interfaces** — the actual transport implementation (SSE via `fetch` + `ReadableStream`, `EventSource`, WebSocket, etc.) is provided by the caller.

### Example Proto

```protobuf
service LogService {
  rpc GetLog(GetLogRequest) returns (GetLogResponse) {
    option (google.api.http) = {get: "/v1/{name=logs/*}"};
  }

  // Server-streaming
  rpc TailLogs(TailLogsRequest) returns (stream LogEntry) {
    option (google.api.http) = {get: "/v1/{name=logs/*}:tail"};
  }

  // Bidirectional streaming
  rpc Chat(stream ChatMessage) returns (stream ChatMessage) {
    option (google.api.http) = {get: "/v1/chat"};
  }
}
```

### Implementing ServerStream (fetch + ReadableStream for SSE)

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

### Passing the Transport

```typescript
const transport: ClientTransport = {
  // ...unary implementation...
  serverStream<T>(path, _meta) {
    return new FetchSSETransport<T>(`https://api.example.com/${path}`);
  },
  duplexStream<TIn, TOut>(path, _meta) {
    // return your WebSocket-based DuplexStream implementation
  },
};

const client = createLogServiceClient(transport);

// Server-streaming
const tail = client.TailLogs({ name: "log/123" });
tail.onEvent((entry) => console.log(entry.message));
tail.onError((err) => console.error(err));
// tail.close();

// Bidirectional streaming
const chat = client.Chat();
chat.onEvent((msg) => console.log(msg.text));
chat.send({ text: "hello" });
// chat.close();
```

## Unified ApiClient

When a proto package contains multiple services, an `ApiClient` class is generated that aggregates all service clients. Pass your transport once:

```typescript
import { ApiClient, ClientTransport } from "./gen";

const transport: ClientTransport = { /* ... */ };
const api = new ApiClient(transport);
// or: const api = createApiClient(transport);

// Access individual services lazily
const shipper = await api.shipperService.GetShipper({ name: "shippers/123" });
```

## License

[MIT](./LICENSE)
