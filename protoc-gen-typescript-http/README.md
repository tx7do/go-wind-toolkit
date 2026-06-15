# protoc-gen-typescript-http

Generates Typescript types and service clients from protobuf definitions
annotated with
[http rules](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto).
The generated types follow the
[canonical JSON encoding](https://developers.google.com/protocol-buffers/docs/proto3#json).

**Experimental**: This library is under active development and breaking changes
to config files, APIs and generated code are expected between releases.

## Using the plugin

For examples of correctly annotated protobuf defintions and the generated code,
look at [examples](./examples).

### Install the plugin

```bash
go install github.com/go-kratos/protoc-gen-typescript-http@latest
```

Or download a prebuilt binary from [releases](./releases).

### Invocation

```bash
protoc 
  --typescript-http_out [OUTPUT DIR] \
  [.proto files ...]
```

______________________________________________________________________

The generated clients use a `ClientTransport` interface that handles all
communication — unary requests, server-streaming (SSE), and bidirectional
streaming (WebSocket).

### Basic usage

```typescript
import { createDefaultTransport, createShipperServiceClient } from "./gen";

const transport = createDefaultTransport({
  baseUrl: "https://api.example.com",
});

const client = createShipperServiceClient(transport);

// Unary call
const shipper = await client.GetShipper({ name: "shippers/123" });
```

### With google.api.default_host

If your proto service defines the `google.api.default_host` option, a
`DEFAULT_HOST` constant is generated automatically:

```protobuf
service ShipperService {
  option (google.api.default_host) = "api.example.com";
  // ...
}
```

```typescript
// No baseUrl needed — uses DEFAULT_HOST
const transport = createDefaultTransport();
const client = createShipperServiceClient(transport);
```

### Custom request and headers

`request` accepts a function with the same signature as `fetch`. You can use
it to add logging, error handling, or delegate to another HTTP library:

```typescript
function fetchRequest(url: string, init: RequestInit): Promise<Response> {
  console.log("requesting", init.method, url);
  return fetch(url, init).then((response) => {
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    return response;
  });
}

const transport = createDefaultTransport({
  baseUrl: "/api",
  headers: { Authorization: "Bearer token" },
  request: fetchRequest,
});
```

### Streaming

Server-streaming RPCs (`returns (stream ...)`) are generated as **SSE** (Server-Sent Events).
Bidirectional streaming RPCs (`stream ... returns (stream ...)`) are generated as **WebSocket**.

Example proto:

```protobuf
service LogService {
  rpc GetLog(GetLogRequest) returns (GetLogResponse) {
    option (google.api.http) = {get: "/v1/{name=logs/*}"};
  }

  // Server-streaming → SSE
  rpc TailLogs(TailLogsRequest) returns (stream LogEntry) {
    option (google.api.http) = {get: "/v1/{name=logs/*}:tail"};
  }

  // Bidirectional streaming → WebSocket
  rpc Chat(stream ChatMessage) returns (stream ChatMessage) {
    option (google.api.http) = {get: "/v1/chat"};
  }
}
```

Generated usage:

```typescript
const transport = createDefaultTransport({ baseUrl: "https://api.example.com" });
const client = createLogServiceClient(transport);

// Unary
const log = await client.GetLog({ name: "log/123" });

// Server-streaming (SSE)
const tail = client.TailLogs({ name: "log/123" });
const off = tail.onEvent((entry) => {
  console.log("log entry:", entry.message);
});
tail.onError((err) => {
  console.error("tail error:", err);
});
// off();  // unsubscribe
// tail.close();

// Bidirectional streaming (WebSocket)
const chat = client.Chat();
chat.onEvent((msg) => {
  console.log("received:", msg.text);
});
chat.onError((err) => {
  console.error("chat error:", err);
});
chat.send({ text: "hello" });
// chat.close();
```

### Implementing a custom transport

The `ClientTransport` interface can be implemented to use any underlying HTTP
library (Axios, Node.js http, etc.):

```typescript
import { ClientTransport } from "./gen";

const myTransport: ClientTransport = {
  unary(path, method, body, meta) {
    // use your own HTTP client
    return myHttpClient.request({ url: path, method, body }).then(r => r.json());
  },
  serverStream<T>(path, meta) {
    // return a custom ServerStream implementation
  },
  duplexStream<TIn, TOut>(path, meta) {
    // return a custom DuplexStream implementation
  },
};

const client = createMyServiceClient(myTransport);
```
