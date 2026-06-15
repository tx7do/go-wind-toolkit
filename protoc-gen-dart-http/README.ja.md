# protoc-gen-dart-http

[中文](./README.md) | [English](./README.en.md)

[HTTP ルールアノテーション](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto)付きの Protobuf 定義から Dart HTTP クライアントコードを生成します。生成される型は[Proto JSON エンコーディング仕様](https://developers.google.com/protocol-buffers/docs/proto3#json)に準拠します。

## 特徴

- **Dart ネイティブスタイル** — 生成コードは Dart の命名規則に完全準拠（PascalCase クラス名、lowerCamelCase フィールド/メソッド、`///` ドキュメントコメント）
- **ランタイム依存ゼロ** — 生成コードは `dart:async` と `dart:convert` のみに依存し、特定の HTTP ライブラリに束縛されません
- **トランスポート抽象化** — `ClientTransport` 抽象インターフェースにより、任意の HTTP クライアント実装（package:http、dio など）をサポート
- **ストリーミングサポート** — サーバーストリーミング RPC は SSE に、双方向ストリーミング RPC は WebSocket にマッピング
- **完全なデータモデル** — `fromJson`、`toJson`、`toString`、`==`、`hashCode`、`copyWith` を自動生成
- **Well-known 型マッピング** — `google.protobuf.Timestamp` などの Well-known 型を Dart ネイティブ型に自動マッピング
- **クロスパッケージ参照** — protobuf パッケージ間の型参照は PascalCase プレフィックスを使用（例: `EinrideExampleSyntaxV1Message`）
- **ネスト型** — Dart protobuf 慣例の `$` セパレータを使用（例: `Message$NestedMessage`）

## インストール

### ソースからインストール

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http@latest
```

### プリビルドバイナリをダウンロード

[Releases ページ](../../releases)から各プラットフォーム用のバイナリをダウンロードし、`PATH` に追加してください。

## 使用方法

### protoc 経由で呼び出し

```bash
protoc \
  --dart-http_out=[出力ディレクトリ] \
  [.proto ファイル ...]
```

### buf 経由で呼び出し

`buf.gen.yaml` を設定:

```yaml
version: v2

plugins:
  - local: protoc-gen-dart-http
    out: gen/dart
```

実行:

```bash
buf generate
```

完全な例は [examples](./examples) を参照してください。

## 生成コードの構造

各 protobuf パッケージごとに `index.dart` ファイルが生成されます。また、ルートディレクトリに共有の `transport.dart` が生成されます:

```
gen/dart/
├── transport.dart                          # 共有トランスポート抽象
└── einride/example/
    ├── freight/v1/index.dart               # 貨物サービスモデルとクライアント
    ├── stream/v1/index.dart                # ストリームサービスモデルとクライアント
    └── syntax/v1/index.dart                # 構文テストモデルとクライアント
```

### transport.dart

すべての生成クライアントが共有するトランスポート抽象インターフェースを定義します:

```dart
/// メタデータ: RPC コールのサービス名とメソッド名
class TransportMeta {
  final String service;
  final String method;
  const TransportMeta({required this.service, required this.method});
}

/// トランスポート抽象 — お好みの HTTP クライアントで実装してください
abstract class ClientTransport {
  /// 単項呼び出し（リクエスト/レスポンス）
  Future<dynamic> unary(String path, String method, String? body, TransportMeta meta, {Map<String, String>? headers});

  /// サーバーストリーミング（SSE）
  Stream<Map<String, dynamic>> serverStream(String path, TransportMeta meta, {Map<String, String>? headers});

  /// 双方向ストリーミング（WebSocket）
  DuplexConnection duplexStream(String path, TransportMeta meta, {Map<String, String>? headers});
}

/// 双方向接続抽象
abstract class DuplexConnection {
  Stream<Map<String, dynamic>> get incoming;
  void send(Map<String, dynamic> data);
  Future<void> close();
}

/// 型安全な双方向接続ラッパー
class TypedDuplexConnection<TIn, TOut> { ... }
```

## クイックスタート

### 1. Proto サービスを定義

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

### 2. コードを生成

```bash
buf generate
```

### 3. トランスポートを実装

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
    // EventSource (SSE) を使用して実装
    throw UnimplementedError();
  }

  @override
  DuplexConnection duplexStream(
    String path,
    TransportMeta meta, {
    Map<String, String>? headers,
  }) {
    // WebSocket を使用して実装
    throw UnimplementedError();
  }
}
```

### 4. API を呼び出す

```dart
import 'gen/dart/einride/example/freight/v1/index.dart';
import 'gen/dart/transport.dart';

void main() async {
  final transport = HttpTransport(baseUrl: 'https://api.example.com');
  final client = createApiClient(transport);

  // 単項呼び出し
  final shipper = await client.freightService.getShipper(
    GetShipperRequest(name: 'shippers/123'),
  );
  print(shipper.displayName);

  // カスタムヘッダー付き
  final result = await client.freightService.getShipper(
    GetShipperRequest(name: 'shippers/123'),
    headers: {'Authorization': 'Bearer token'},
  );

  // 使用後にリソースを解放
  client.dispose();
}
```

## ストリーミング RPC

### サーバーストリーミング → SSE

```protobuf
service LogService {
  rpc TailLogs(TailLogsRequest) returns (stream LogEntry) {
    option (google.api.http) = {
      get: "/v1/{name=logs/*}:tail"
    };
  }
}
```

生成される Dart コード:

```dart
Stream<LogEntry> tailLogs(TailLogsRequest request, {Map<String, String>? headers}) {
  // Stream<LogEntry> を返す — await for で消費
  return _transport.serverStream(uri, TransportMeta(...), headers: headers)
      .map((json) => LogEntry.fromJson(json));
}
```

使用例:

```dart
final stream = client.streamService.tailLogs(
  TailLogsRequest(name: 'logs/1'),
);
await for (final entry in stream) {
  print('ログを受信: ${entry.message}');
}
```

### 双方向ストリーミング → WebSocket

```protobuf
service ChatService {
  rpc Chat(stream ChatMessage) returns (stream ChatMessage) {
    option (google.api.http) = {
      get: "/v1/chat"
    };
  }
}
```

生成される Dart コード:

```dart
TypedDuplexConnection<ChatMessage, ChatMessage> chat({Map<String, String>? headers}) {
  return TypedDuplexConnection<ChatMessage, ChatMessage>(
    _transport.duplexStream(path, TransportMeta(...), headers: headers),
    (json) => ChatMessage.fromJson(json),
    (data) => data.toJson(),
  );
}
```

使用例:

```dart
final chat = client.streamService.chat();

// メッセージを受信
chat.stream.listen((msg) {
  print('受信: ${msg.text}');
});

// メッセージを送信
chat.send(ChatMessage(text: 'こんにちは'));
// 閉じる
await chat.close();
```

## default_host サポート

Proto サービスで `google.api.default_host` オプションが定義されている場合、`defaultHost` 定数が自動生成されます:

```protobuf
service FreightService {
  option (google.api.default_host) = "freight-example.einride.tech";
}
```

```dart
// 生成コード
const defaultHost = 'freight-example.einride.tech';
```

## Well-known 型マッピング

| Proto 型                       | Dart 型                 | JSON フォーマット                           |
|-------------------------------|------------------------|---------------------------------------|
| `google.protobuf.Timestamp`   | `String`               | RFC 3339（例: `"2021-01-01T00:00:00Z"`） |
| `google.protobuf.Duration`    | `String`               | 例: `"3.5s"`                           |
| `google.protobuf.Any`         | `Map<String, dynamic>` | `{"@type": "...", ...}`               |
| `google.protobuf.Empty`       | `Map<String, dynamic>` | `{}`                                  |
| `google.protobuf.Struct`      | `Map<String, dynamic>` | JSON オブジェクト                           |
| `google.protobuf.Value`       | `dynamic`              | 任意の JSON 値                            |
| `google.protobuf.ListValue`   | `List<dynamic>`        | JSON 配列                               |
| `google.protobuf.NullValue`   | `String`               | `"NULL_VALUE"`                        |
| `google.protobuf.FieldMask`   | `String`               | カンマ区切りの camelCase パス                  |
| `google.protobuf.BoolValue`   | `bool`                 | `true`/`false`                        |
| `google.protobuf.BytesValue`  | `String`               | Base64                                |
| `google.protobuf.DoubleValue` | `double`               | 数値                                    |
| `google.protobuf.FloatValue`  | `double`               | 数値                                    |
| `google.protobuf.Int32Value`  | `int`                  | 数値                                    |
| `google.protobuf.Int64Value`  | `int`                  | 数値                                    |
| `google.protobuf.UInt32Value` | `int`                  | 数値                                    |
| `google.protobuf.UInt64Value` | `int`                  | 数値                                    |
| `google.protobuf.StringValue` | `String`               | 文字列                                   |

## 命名規則

生成コードは [Dart スタイルガイド](https://dart.dev/guides/language/effective-dart/style)に厳密に従います:

| 要素           | 規則                  | 例                                                         |
|--------------|---------------------|-----------------------------------------------------------|
| クラス / 列挙型    | PascalCase          | `Shipment`、`LogEntry`                                     |
| フィールド / メソッド | lowerCamelCase      | `displayName`、`createShipment`                            |
| 列挙値          | lowerCamelCase      | `enumOne`、`enumUnspecified`                               |
| プライベートメンバー   | `_` プレフィックス         | `_transport`、`_freightService`                            |
| 定数           | lowerCamelCase      | `defaultHost`                                             |
| ネスト型         | `$` セパレータ           | `Message$NestedMessage`                                   |
| クロスパッケージ参照   | PascalCase プレフィックス  | `EinrideExampleSyntaxV1Message`                           |
| ドキュメントコメント   | `///`               | `/// The resource name.`                                  |
| ファイルヘッダー     | `// Code generated` | `// Code generated by protoc-gen-dart-http. DO NOT EDIT.` |

## License

[MIT](../LICENSE)
