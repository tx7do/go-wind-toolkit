# protoc-gen-dart-http

[English](./README.en.md) | [日本語](./README.ja.md)

从 Protobuf 定义生成 Dart HTTP 客户端代码。基于 [HTTP 注解规则](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto) 自动映射 RPC 方法到 RESTful 接口，生成的类型遵循 [Proto JSON 编码规范](https://developers.google.com/protocol-buffers/docs/proto3#json)。

## 特性

- **Dart 原生风格** — 生成的代码完全遵循 Dart 命名规范（PascalCase 类名、lowerCamelCase 字段/方法、`///` 文档注释）
- **零运行时依赖** — 生成的代码仅依赖 `dart:async` 和 `dart:convert`，不绑定任何特定 HTTP 库
- **传输层抽象** — 通过 `ClientTransport` 抽象接口支持任意 HTTP 客户端实现（package:http、dio 等）
- **流式支持** — 服务端流式 RPC 映射为 SSE，双向流式 RPC 映射为 WebSocket
- **完整的数据模型** — 自动生成 `fromJson`、`toJson`、`toString`、`==`、`hashCode`、`copyWith`
- **Well-known 类型映射** — 自动将 `google.protobuf.Timestamp` 等 Well-known 类型映射为 Dart 原生类型
- **跨包引用** — 跨 protobuf 包的类型引用使用 PascalCase 前缀（如 `EinrideExampleSyntaxV1Message`）
- **嵌套类型** — 使用 Dart protobuf 惯例的 `$` 分隔符（如 `Message$NestedMessage`）

## 安装

### 从源码安装

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http@latest
```

### 从 Release 下载预编译二进制

前往 [Releases 页面](../../releases) 下载对应平台的二进制文件，并将其添加到 `PATH` 环境变量中。

## 使用方法

### 通过 protoc 调用

```bash
protoc \
  --dart-http_out=[输出目录] \
  [.proto 文件 ...]
```

### 通过 buf 调用

在 `buf.gen.yaml` 中配置：

```yaml
version: v2

plugins:
  - local: protoc-gen-dart-http
    out: gen/dart
```

然后执行：

```bash
buf generate
```

完整示例请参考 [examples](./examples)。

## 生成的代码结构

每个 protobuf 包会生成一个 `index.dart` 文件，此外还会在根目录生成一个共享的 `transport.dart`：

```
gen/dart/
├── transport.dart                          # 共享传输层抽象
└── einride/example/
    ├── freight/v1/index.dart               # 货运服务模型和客户端
    ├── stream/v1/index.dart                # 流式服务模型和客户端
    └── syntax/v1/index.dart                # 语法测试模型和客户端
```

### transport.dart

定义了所有生成客户端共享的传输层抽象接口：

```dart
/// 元数据：RPC 调用的服务名和方法名
class TransportMeta {
  final String service;
  final String method;
  const TransportMeta({required this.service, required this.method});
}

/// 传输层抽象接口 — 用你喜欢的 HTTP 客户端实现它
abstract class ClientTransport {
  /// 一元调用（请求/响应）
  Future<dynamic> unary(String path, String method, String? body, TransportMeta meta, {Map<String, String>? headers});

  /// 服务端流式（SSE）
  Stream<Map<String, dynamic>> serverStream(String path, TransportMeta meta, {Map<String, String>? headers});

  /// 双向流式（WebSocket）
  DuplexConnection duplexStream(String path, TransportMeta meta, {Map<String, String>? headers});
}

/// 双向连接抽象
abstract class DuplexConnection {
  Stream<Map<String, dynamic>> get incoming;
  void send(Map<String, dynamic> data);
  Future<void> close();
}

/// 类型安全的双向连接包装器
class TypedDuplexConnection<TIn, TOut> { ... }
```

## 快速上手

### 1. 定义 Proto 服务

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

### 2. 生成代码

```bash
buf generate
```

### 3. 实现传输层

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
    // 使用 EventSource (SSE) 实现
    throw UnimplementedError();
  }

  @override
  DuplexConnection duplexStream(
    String path,
    TransportMeta meta, {
    Map<String, String>? headers,
  }) {
    // 使用 WebSocket 实现
    throw UnimplementedError();
  }
}
```

### 4. 调用 API

```dart
import 'gen/dart/einride/example/freight/v1/index.dart';
import 'gen/dart/transport.dart';

void main() async {
  final transport = HttpTransport(baseUrl: 'https://api.example.com');
  final client = createApiClient(transport);

  // 一元调用
  final shipper = await client.freightService.getShipper(
    GetShipperRequest(name: 'shippers/123'),
  );
  print(shipper.displayName);

  // 带自定义请求头
  final result = await client.freightService.getShipper(
    GetShipperRequest(name: 'shippers/123'),
    headers: {'Authorization': 'Bearer token'},
  );

  // 使用完毕后释放资源
  client.dispose();
}
```

## 流式 RPC

### 服务端流式 → SSE

```protobuf
service LogService {
  rpc TailLogs(TailLogsRequest) returns (stream LogEntry) {
    option (google.api.http) = {
      get: "/v1/{name=logs/*}:tail"
    };
  }
}
```

生成的 Dart 代码：

```dart
Stream<LogEntry> tailLogs(TailLogsRequest request, {Map<String, String>? headers}) {
  // 返回 Stream<LogEntry>，直接 await for 消费
  return _transport.serverStream(uri, TransportMeta(...), headers: headers)
      .map((json) => LogEntry.fromJson(json));
}
```

使用方式：

```dart
final stream = client.streamService.tailLogs(
  TailLogsRequest(name: 'logs/1'),
);
await for (final entry in stream) {
  print('收到日志: ${entry.message}');
}
```

### 双向流式 → WebSocket

```protobuf
service ChatService {
  rpc Chat(stream ChatMessage) returns (stream ChatMessage) {
    option (google.api.http) = {
      get: "/v1/chat"
    };
  }
}
```

生成的 Dart 代码：

```dart
TypedDuplexConnection<ChatMessage, ChatMessage> chat({Map<String, String>? headers}) {
  return TypedDuplexConnection<ChatMessage, ChatMessage>(
    _transport.duplexStream(path, TransportMeta(...), headers: headers),
    (json) => ChatMessage.fromJson(json),
    (data) => data.toJson(),
  );
}
```

使用方式：

```dart
final chat = client.streamService.chat();

// 接收消息
chat.stream.listen((msg) {
  print('收到: ${msg.text}');
});

// 发送消息
chat.send(ChatMessage(text: '你好'));
// 关闭
await chat.close();
```

## default_host 支持

如果 proto 服务定义了 `google.api.default_host` 选项，会自动生成 `defaultHost` 常量：

```protobuf
service FreightService {
  option (google.api.default_host) = "freight-example.einride.tech";
}
```

```dart
// 生成的代码
const defaultHost = 'freight-example.einride.tech';
```

## Well-known 类型映射

| Proto 类型                      | Dart 类型                | JSON 格式                              |
|-------------------------------|------------------------|--------------------------------------|
| `google.protobuf.Timestamp`   | `String`               | RFC 3339（如 `"2021-01-01T00:00:00Z"`） |
| `google.protobuf.Duration`    | `String`               | 如 `"3.5s"`                           |
| `google.protobuf.Any`         | `Map<String, dynamic>` | `{"@type": "...", ...}`              |
| `google.protobuf.Empty`       | `Map<String, dynamic>` | `{}`                                 |
| `google.protobuf.Struct`      | `Map<String, dynamic>` | JSON 对象                              |
| `google.protobuf.Value`       | `dynamic`              | 任意 JSON 值                            |
| `google.protobuf.ListValue`   | `List<dynamic>`        | JSON 数组                              |
| `google.protobuf.NullValue`   | `String`               | `"NULL_VALUE"`                       |
| `google.protobuf.FieldMask`   | `String`               | 逗号分隔的 camelCase 路径                   |
| `google.protobuf.BoolValue`   | `bool`                 | `true`/`false`                       |
| `google.protobuf.BytesValue`  | `String`               | Base64                               |
| `google.protobuf.DoubleValue` | `double`               | 数字                                   |
| `google.protobuf.FloatValue`  | `double`               | 数字                                   |
| `google.protobuf.Int32Value`  | `int`                  | 数字                                   |
| `google.protobuf.Int64Value`  | `int`                  | 数字                                   |
| `google.protobuf.UInt32Value` | `int`                  | 数字                                   |
| `google.protobuf.UInt64Value` | `int`                  | 数字                                   |
| `google.protobuf.StringValue` | `String`               | 字符串                                  |

## 命名规范

生成的代码严格遵循 [Dart 风格指南](https://dart.dev/guides/language/effective-dart/style)：

| 元素      | 规范                  | 示例                                                        |
|---------|---------------------|-----------------------------------------------------------|
| 类 / 枚举  | PascalCase          | `Shipment`、`LogEntry`                                     |
| 字段 / 方法 | lowerCamelCase      | `displayName`、`createShipment`                            |
| 枚举值     | lowerCamelCase      | `enumOne`、`enumUnspecified`                               |
| 私有成员    | `_` 前缀              | `_transport`、`_freightService`                            |
| 常量      | lowerCamelCase      | `defaultHost`                                             |
| 嵌套类型    | `$` 分隔              | `Message$NestedMessage`                                   |
| 跨包引用    | PascalCase 前缀       | `EinrideExampleSyntaxV1Message`                           |
| 文档注释    | `///`               | `/// The resource name.`                                  |
| 文件头     | `// Code generated` | `// Code generated by protoc-gen-dart-http. DO NOT EDIT.` |

## License

[MIT](../LICENSE)
