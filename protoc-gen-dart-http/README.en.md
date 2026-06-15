# protoc-gen-dart-http

[中文](./README.md) | [日本語](./README.ja.md)

Generates Dart HTTP client code from Protobuf definitions annotated with [HTTP rules](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto). Generated types follow the [canonical Proto JSON encoding](https://developers.google.com/protocol-buffers/docs/proto3#json).

## Features

- **Dart-native style** — Generated code fully adheres to Dart naming conventions (PascalCase classes, lowerCamelCase fields/methods, `///` doc comments)
- **Zero runtime dependencies** — Generated code only depends on `dart:async` and `dart:convert`; no binding to any specific HTTP library
- **Transport abstraction** — The `ClientTransport` abstract interface supports any HTTP client implementation (package:http, dio, etc.)
- **Streaming support** — Server-streaming RPCs map to SSE; bidirectional streaming RPCs map to WebSocket
- **Complete data models** — Auto-generates `fromJson`, `toJson`, `toString`, `==`, `hashCode`, `copyWith`
- **Well-known type mapping** — Automatically maps `google.protobuf.Timestamp` and other Well-known types to native Dart types
- **Cross-package references** — Types referenced across protobuf packages use PascalCase prefixes (e.g., `EinrideExampleSyntaxV1Message`)
- **Nested types** — Uses the Dart protobuf convention `$` separator (e.g., `Message$NestedMessage`)

## Installation

### Install from source

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http@latest
```

### Download prebuilt binary

Go to the [Releases page](../../releases) to download the binary for your platform and add it to your `PATH`.

## Usage

### Via protoc

```bash
protoc \
  --dart-http_out=[OUTPUT DIR] \
  [.proto files ...]
```

### Via buf

Configure `buf.gen.yaml`:

```yaml
version: v2

plugins:
  - local: protoc-gen-dart-http
    out: gen/dart
```

Then run:

```bash
buf generate
```

For complete examples, see [examples](./examples).

## Generated Code Structure

Each protobuf package generates an `index.dart` file. A shared `transport.dart` is also generated at the root:

```
gen/dart/
├── transport.dart                          # Shared transport abstraction
└── einride/example/
    ├── freight/v1/index.dart               # Freight service models and clients
    ├── stream/v1/index.dart                # Stream service models and clients
    └── syntax/v1/index.dart                # Syntax test models and clients
```

### transport.dart

Defines the transport abstraction interface shared by all generated clients:

```dart
/// Metadata: service name and method name for an RPC call
class TransportMeta {
  final String service;
  final String method;
  const TransportMeta({required this.service, required this.method});
}

/// Transport abstraction — implement with your preferred HTTP client
abstract class ClientTransport {
  /// Unary call (request/response)
  Future<dynamic> unary(String path, String method, String? body, TransportMeta meta, {Map<String, String>? headers});

  /// Server streaming (SSE)
  Stream<Map<String, dynamic>> serverStream(String path, TransportMeta meta, {Map<String, String>? headers});

  /// Bidirectional streaming (WebSocket)
  DuplexConnection duplexStream(String path, TransportMeta meta, {Map<String, String>? headers});
}

/// Bidirectional connection abstraction
abstract class DuplexConnection {
  Stream<Map<String, dynamic>> get incoming;
  void send(Map<String, dynamic> data);
  Future<void> close();
}

/// Type-safe bidirectional connection wrapper
class TypedDuplexConnection<TIn, TOut> { ... }
```

## Quick Start

### 1. Define a Proto Service

```protobuf
syntax = "proto3";

package example.v1;

import "google/api/http.proto";
import "google/api/field_behavior.proto";

service ShipperService {
  option (google.api.default_host) = "api.example.com";

  rpc GetShipper(GetShipperRequest) returns (Shipper) {
    option (google.api.http) = {
      get: "/v1/{name=shippers/*}"
    };
  }

  rpc CreateShipper(CreateShipperRequest) returns (Shipper) {
    option (google.api.http) = {
      post: "/v1/shippers"
      body: "shipper"
    };
  }
}

message GetShipperRequest {
  string name = 1 [(google.api.field_behavior) = REQUIRED];
}

message CreateShipperRequest {
  Shipper shipper = 1;
}

message Shipper {
  string name = 1;
  string display_name = 2;
}
```

### 2. Generate Code

```bash
buf generate
```

### 3. Implement the Transport

```dart
import 'package:http/http.dart' as http;
import '../transport.dart';

class HttpTransport implements ClientTransport {
  final String baseUrl;
  final Map<String, String>? defaultHeaders;

  HttpTransport({required this.baseUrl, this.defaultHeaders});

  @override
  Future<dynamic> unary(
    String path,
    String method,
    String? body,
    TransportMeta meta, {
    Map<String, String>? headers,
  }) async {
    final uri = Uri.parse('$baseUrl/$path');
    final response = await http.Client().send(http.Request(method, uri)
      ..body = body ?? ''
      ..headers.addAll({...?defaultHeaders, ...?headers}));

    if (response.statusCode >= 400) {
      throw Exception('HTTP ${response.statusCode}');
    }
    final responseBody = await response.stream.bytesToString();
    return jsonDecode(responseBody);
  }

  @override
  Stream<Map<String, dynamic>> serverStream(
    String path,
    TransportMeta meta, {
    Map<String, String>? headers,
  }) {
    // Implement using EventSource (SSE)
    throw UnimplementedError();
  }

  @override
  DuplexConnection duplexStream(
    String path,
    TransportMeta meta, {
    Map<String, String>? headers,
  }) {
    // Implement using WebSocket
    throw UnimplementedError();
  }
}
```

### 4. Call the API

```dart
import 'gen/dart/einride/example/freight/v1/index.dart';
import 'gen/dart/transport.dart';

void main() async {
  final transport = HttpTransport(baseUrl: 'https://api.example.com');
  final client = createApiClient(transport);

  // Unary call
  final shipper = await client.freightService.getShipper(
    GetShipperRequest(name: 'shippers/123'),
  );
  print(shipper.displayName);

  // With custom headers
  final result = await client.freightService.getShipper(
    GetShipperRequest(name: 'shippers/123'),
    headers: {'Authorization': 'Bearer token'},
  );

  // Release resources when done
  client.dispose();
}
```

## Streaming RPCs

### Server Streaming → SSE

```protobuf
service LogService {
  rpc TailLogs(TailLogsRequest) returns (stream LogEntry) {
    option (google.api.http) = {
      get: "/v1/{name=logs/*}:tail"
    };
  }
}
```

Generated Dart code:

```dart
Stream<LogEntry> tailLogs(TailLogsRequest request, {Map<String, String>? headers}) {
  // Returns Stream<LogEntry> — consume with await for
  return _transport.serverStream(uri, TransportMeta(...), headers: headers)
      .map((json) => LogEntry.fromJson(json));
}
```

Usage:

```dart
final stream = client.streamService.tailLogs(
  TailLogsRequest(name: 'logs/1'),
);
await for (final entry in stream) {
  print('Received log: ${entry.message}');
}
```

### Bidirectional Streaming → WebSocket

```protobuf
service ChatService {
  rpc Chat(stream ChatMessage) returns (stream ChatMessage) {
    option (google.api.http) = {
      get: "/v1/chat"
    };
  }
}
```

Generated Dart code:

```dart
TypedDuplexConnection<ChatMessage, ChatMessage> chat({Map<String, String>? headers}) {
  return TypedDuplexConnection<ChatMessage, ChatMessage>(
    _transport.duplexStream(path, TransportMeta(...), headers: headers),
    (json) => ChatMessage.fromJson(json),
    (data) => data.toJson(),
  );
}
```

Usage:

```dart
final chat = client.streamService.chat();

// Receive messages
chat.stream.listen((msg) {
  print('Received: ${msg.text}');
});

// Send a message
chat.send(ChatMessage(text: 'Hello'));
// Close
await chat.close();
```

## default_host Support

If a proto service defines the `google.api.default_host` option, a `defaultHost` constant is generated:

```protobuf
service FreightService {
  option (google.api.default_host) = "freight-example.einride.tech";
}
```

```dart
// Generated code
const defaultHost = 'freight-example.einride.tech';
```

## Well-known Type Mapping

| Proto Type                    | Dart Type              | JSON Format                               |
|-------------------------------|------------------------|-------------------------------------------|
| `google.protobuf.Timestamp`   | `String`               | RFC 3339 (e.g., `"2021-01-01T00:00:00Z"`) |
| `google.protobuf.Duration`    | `String`               | e.g., `"3.5s"`                            |
| `google.protobuf.Any`         | `Map<String, dynamic>` | `{"@type": "...", ...}`                   |
| `google.protobuf.Empty`       | `Map<String, dynamic>` | `{}`                                      |
| `google.protobuf.Struct`      | `Map<String, dynamic>` | JSON object                               |
| `google.protobuf.Value`       | `dynamic`              | Any JSON value                            |
| `google.protobuf.ListValue`   | `List<dynamic>`        | JSON array                                |
| `google.protobuf.NullValue`   | `String`               | `"NULL_VALUE"`                            |
| `google.protobuf.FieldMask`   | `String`               | Comma-separated camelCase paths           |
| `google.protobuf.BoolValue`   | `bool`                 | `true`/`false`                            |
| `google.protobuf.BytesValue`  | `String`               | Base64                                    |
| `google.protobuf.DoubleValue` | `double`               | Number                                    |
| `google.protobuf.FloatValue`  | `double`               | Number                                    |
| `google.protobuf.Int32Value`  | `int`                  | Number                                    |
| `google.protobuf.Int64Value`  | `int`                  | Number                                    |
| `google.protobuf.UInt32Value` | `int`                  | Number                                    |
| `google.protobuf.UInt64Value` | `int`                  | Number                                    |
| `google.protobuf.StringValue` | `String`               | String                                    |

## Naming Conventions

Generated code strictly follows the [Dart style guide](https://dart.dev/guides/language/effective-dart/style):

| Element                  | Convention          | Example                                                   |
|--------------------------|---------------------|-----------------------------------------------------------|
| Classes / Enums          | PascalCase          | `Shipment`, `LogEntry`                                    |
| Fields / Methods         | lowerCamelCase      | `displayName`, `createShipment`                           |
| Enum Values              | lowerCamelCase      | `enumOne`, `enumUnspecified`                              |
| Private Members          | `_` prefix          | `_transport`, `_freightService`                           |
| Constants                | lowerCamelCase      | `defaultHost`                                             |
| Nested Types             | `$` separator       | `Message$NestedMessage`                                   |
| Cross-package References | PascalCase prefix   | `EinrideExampleSyntaxV1Message`                           |
| Doc Comments             | `///`               | `/// The resource name.`                                  |
| File Header              | `// Code generated` | `// Code generated by protoc-gen-dart-http. DO NOT EDIT.` |

## License

[MIT](../LICENSE)
