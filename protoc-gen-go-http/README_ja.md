# protoc-gen-go-http

[中文](README.md) | [English](README_en.md)

`protoc-gen-go-http` は [`google.api.http`](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto) アノテーションに基づいて、Protobuf サービス向けに Go HTTP サーバーコード（gRPC HTTP ゲートウェイ）を生成する [protoc](https://github.com/protocolbuffers/protobuf) プラグインです。

生成されるコードは標準ライブラリ `net/http` に基づいており、[`go-wind-toolkit`](https://github.com/tx7do/go-wind-toolkit) の `transport/http/binding` パッケージを使用してリクエストバインディング、ルーティング登録、レスポンス出力を行います。

## 特徴

- `google.api.http` アノテーションからの自動ルート生成
- `GET` / `POST` / `PUT` / `DELETE` / `PATCH` / カスタムメソッドに対応
- `additional_bindings` による複数ルートバインディングに対応
- パス変数バインディングに対応（ネストフィールド含む、例: `{message.id}`）
- リクエストボディバインディングに対応（`body: "*"` または特定フィールド）
- クエリパラメータの自動バインディングに対応
- `google.api.HttpBody` 型に対応
- `response_body` レスポンスフィールドマッピングに対応
- HTTP アノテーションを持たないメソッド向けのデフォルトルートを提供（設定可能）
- 生成コードは標準ライブラリ `net/http` ベースでフレームワーク非依存

## インストール

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-go-http@latest
```

> Go 1.25+ と `protoc` コンパイラが必要です。

## クイックスタート

### 1. proto ファイルの作成

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

### 2. コード生成

```bash
protoc \
  --proto_path=./proto \
  --go_out=. --go_opt=paths=source_relative \
  --go-http_out=. --go-http_opt=paths=source_relative \
  proto/helloworld/helloworld.proto
```

> `protoc-gen-go` と `protoc-gen-go-http` の両方がインストールされている必要があります。

### 3. 生成されるコード構成

生成ファイル名は `xxx_http.pb.go` で、以下の内容が含まれます。

- **HTTP サーバーインターフェース**: `GreeterHTTPServer`、ビジネスメソッドシグネチャを定義
- **登録関数**: `RegisterGreeterHTTPServer`、ルートを `binding.Router` に登録
- **各メソッドハンドラ**: `_Greeter_XXX_HTTP_Handler`、バインディングとレスポンス出力を実行

```go
// 生成されるインターフェース
type GreeterHTTPServer interface {
    SayHello(context.Context, *HelloRequest) (*HelloReply, error)
    CreateHello(context.Context, *CreateHelloRequest) (*HelloReply, error)
}

// 登録関数
func RegisterGreeterHTTPServer(srv binding.Router, svc GreeterHTTPServer) {
    srv.Handle("GET", "/helloworld/{name}", _Greeter_SayHello0_HTTP_Handler(svc))
    srv.Handle("POST", "/helloworld", _Greeter_CreateHello0_HTTP_Handler(svc))
}
```

### 4. ビジネスロジックの実装

```go
type greeterServer struct{}

func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello " + req.Name}, nil
}

func (s *greeterServer) CreateHello(ctx context.Context, req *pb.CreateHelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Created " + req.Name}, nil
}
```

### 5. サーバーの登録と起動

```go
func main() {
    mux := http.NewServeMux()
    router := binding.NewRouter(mux) // go-wind-toolkit の Router を使用
    pb.RegisterGreeterHTTPServer(router, &greeterServer{})
    http.ListenAndServe(":8080", mux)
}
```

## コマンドライン引数

| 引数 | デフォルト | 説明 |
|------|-----------|------|
| `version` | `false` | プラグインのバージョンを表示して終了 |
| `omitempty` | `true` | サービスに `google.api.http` アノテーションが含まれない場合、生成をスキップ |
| `omitempty_prefix` | `""` | HTTP アノテーションを持たないメソッドのデフォルトルートを生成する際のパスプレフィックス |

### 使用例

```bash
# アノテーションなしのファイルをスキップ
protoc --go-http_out=. --go-http_opt=omitempty=true proto/...

# アノテーションなしメソッドにデフォルトルートを生成、プレフィックス /api/v1
protoc --go-http_out=. --go-http_opt=omitempty=false,omitempty_prefix=/api/v1 proto/...
```

## 対応する HTTP アノテーション

### HTTP メソッド

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

### パス変数

ネストフィールドのパス変数に対応しています。

```protobuf
option (google.api.http) = {
  get: "/v1/{message.id=messages/*}"
};
```

`{message.id=messages/*}` はルート `/v1/{message.id:messages/[^/]+}` に変換され、リクエストメッセージの `message.id` フィールドに自動的にバインドされます。

### 複数ルートバインディング

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

### レスポンスボディマッピング

```protobuf
rpc DownloadFile(DownloadRequest) returns (DownloadReply) {
  option (google.api.http) = {
    get: "/v1/files/{name}"
    response_body: "data"
  };
}
```

生成コードは `out` 全体ではなく `out.Data` を出力します。

## プロジェクト構成

```
protoc-gen-go-http/
├── main.go              # プラグインのエントリポイント、引数解析と生成駆動
├── http.go              # コア生成ロジック: HTTP アノテーション解析、メソッド記述子の構築
├── template.go          # テンプレートデータ構造（serviceDesc / methodDesc）
├── httpTemplate.tpl     # コード生成テンプレート
├── version.go           # バージョン定義
├── http_test.go         # ユニットテスト
├── go.mod               # Go モジュール定義
└── go.sum               # 依存関係検証
```

## 技術スタック

| コンポーネント | バージョン |
|---------------|-----------|
| Go | 1.25+ |
| `google.golang.org/protobuf` | v1.36.11 |
| `google.golang.org/genproto/googleapis/api` | latest |

## 開発とテスト

```bash
# プラグインのビルド
go build -o protoc-gen-go-http .

# テストの実行
go test .

# 静的解析の実行
go vet .
```

## ライセンス

親リポジトリ [go-wind-toolkit](https://github.com/tx7do/go-wind-toolkit) のライセンス情報を参照してください。
