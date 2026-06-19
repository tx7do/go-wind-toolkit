protoc-gen-redact (PGR)
=======================

[中文](README.md) | [English](README_EN.md) | **[日本語](README_JA.md)**

[![Build and Publish](https://github.com/menta2k/protoc-gen-redact/workflows/Build%20and%20Publish/badge.svg)](https://github.com/menta2k/protoc-gen-redact/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/menta2k/protoc-gen-redact/v3?dropcache)](https://goreportcard.com/report/github.com/menta2k/protoc-gen-redact/v3)
[![Go Reference](https://pkg.go.dev/badge/github.com/menta2k/protoc-gen-redact/v3.svg)](https://pkg.go.dev/github.com/menta2k/protoc-gen-redact/v3)
[![License](https://img.shields.io/badge/license-apache2-mildgreen.svg)](./LICENSE)
[![GitHub release](https://img.shields.io/github/release/menta2k/protoc-gen-redact.svg)](https://github.com/menta2k/protoc-gen-redact/releases)

_protoc-gen-redact (PGR)_ は、サーバー側で gRPC レスポンスのフィールド値を自動的にマスキング（秘匿化）する protoc プラグインです。

---

## 目次

- [帰属表示](#帰属表示)
- [クイックスタート](#クイックスタート)
- [インストール](#インストール)
- [フィールドレベルのマスキングルール](#フィールドレベルのマスキングルール)
  - [スカラーフィールド](#スカラーフィールド)
  - [メッセージフィールド](#メッセージフィールド)
  - [Repeated / Map フィールド](#repeated--map-フィールド)
  - [Proto3 Optional フィールド](#proto3-optional-フィールド)
  - [Oneof フィールド](#oneof-フィールド)
  - [正規表現マスキング (Regex)](#正規表現マスキング-regex)
  - [位置ベースマスク (Mask)](#位置ベースマスク-mask)
  - [メールマスキング (Email)](#メールマスキング-email)
  - [切り詰め (Truncate)](#切り詰め-truncate)
  - [ハッシュ (Hash)](#ハッシュ-hash)
  - [UUID 置換](#uuid-置換)
  - [IP アドレスマスキング](#ip-アドレスマスキング)
  - [URL マスキング](#url-マスキング)
  - [固定長マスク (FixedLength)](#固定長マスク-fixedlength)
  - [カスタムマスキング (Custom)](#カスタムマスキング-custom)
  - [条件付きマスキング (Condition)](#条件付きマスキング-condition)
- [ファイルレベル自動検出 (AutoDetect)](#ファイルレベル自動検出-autodetect)
- [メッセージレベルオプション](#メッセージレベルオプション)
- [サービス・メソッドレベルオプション](#サービスメソッドレベルオプション)
- [カスタムテンプレート](#カスタムテンプレート)
- [Buf 設定](#buf-設定)
- [開発と CI/CD](#開発と-cicd)
- [コントリビュート](#コントリビュート)
- [ライセンスと帰属](#ライセンスと帰属)

---

## 帰属表示

本プロジェクトは、**Shivam Rathore**（Copyright 2020）のオリジナルプロジェクト [protoc-gen-redact](https://github.com/arrakis-digital/protoc-gen-redact) をベースにした派生物です。

- **原作者：** Shivam Rathore
- **オリジナルプロジェクト：** https://github.com/arrakis-digital/protoc-gen-redact
- **コントリビューター：** John Castronuovo

本フォークには以下の拡張が含まれています：
- 包括的なエラーハンドリングとバリデーションシステム
- 豊富なテストスイート（374+ テストケース）
- Oneof フィールドサポート（タイプセーフな switch 文生成）
- Proto3 optional フィールドサポート（正しいポインタセマンティクス）
- カスタムテンプレートファイルサポート
- 実际の protoc コンパイルを使用した統合テスト
- 15種類のマスキングルール（正規表現、マスク、メール、切り詰め、ハッシュ、UUID、IP、URL、固定長マスク、カスタム、条件付きなど）
- ファイルレベル自動検出（フィールド名による自動マスキング）
- アノテーション使用時のみ `.pb.redact.go` を生成する条件付き生成

すべての変更は Apache License 2.0 の下でライセンスされています。

---

## クイックスタート

PGR 拡張をインポートし、proto ファイルのメッセージまたはフィールドにアノテーションを付けるだけです：

```protobuf
syntax = "proto3";

package user;

import "redact/v3/redact.proto";
import "google/protobuf/empty.proto";

option go_package = "github.com/menta2k/protoc-gen-redact/v3/examples/user/pb;user";

message User {
    string username = 1;
    string password = 2 [(redact.v3.value).string = "REDACTED"];
    string email    = 3 [(redact.v3.value).email = { keep_local_first: 2 }];
    string name     = 4;
    Location home   = 5 [(redact.v3.value).message.apply = true];

    message Location {
        double lat = 1 [(redact.v3.value).double = 0.0];
        double lng = 2 [(redact.v3.value).double = 0.0];
    }
}

service Chat {
    rpc GetUser(GetUserRequest) returns (User);
    rpc GetUserInternal(GetUserRequest) returns (User) {
        option (redact.v3.method_skip) = true;
    }
    rpc ListUsers (google.protobuf.Empty) returns (ListUsersResponse) {
        option (redact.v3.internal_method) = true;
    }
}
```

---

## インストール

```bash
go install github.com/menta2k/protoc-gen-redact/v3@latest
```

---

## フィールドレベルのマスキングルール

### スカラーフィールド

各フィールドにカスタムマスキング値を指定します：

```protobuf
string password = 1 [(redact.v3.value).string = "REDACTED"];
int32  age      = 2 [(redact.v3.value).int32 = 0];
bool   active   = 3 [(redact.v3.value).bool = false];
bytes  sign     = 4 [(redact.v3.value).bytes = ""];
double score    = 5 [(redact.v3.value).double = 0.0];
```

すべての proto スカラー型をサポート：`float`、`double`、`int32`、`int64`、`uint32`、`uint64`、`sint32`、`sint64`、`fixed32`、`fixed64`、`sfixed32`、`sfixed64`、`bool`、`string`、`bytes`、`enum`。

### メッセージフィールド

ネストされたメッセージのマスキング動作を制御します：

```protobuf
// ネストされたメッセージに再帰的にマスキングを適用
Profile profile = 1 [(redact.v3.value).message.apply = true];

// メッセージ全体を nil に設定
Settings settings = 2 [(redact.v3.value).message.nil = true];

// 空のインスタンスで置換
Metadata metadata = 3 [(redact.v3.value).message.empty = true];

// このフィールドのマスキングを完全にスキップ
AuditLog log = 4 [(redact.v3.value).message.skip = true];
```

### Repeated / Map フィールド

```protobuf
// コレクションを空にする
map<string, string> attributes = 1 [(redact.v3.value).element.empty = true];

// 各要素にデフォルトマスキングを適用
repeated Address addresses = 2 [(redact.v3.value).element.nested = true];

// 各要素にカスタムルールを適用
repeated int32 scores = 3 [(redact.v3.value).element.item.int32 = 0];
repeated string phones = 4 [(redact.v3.value).element.item.mask = { keep_first: 3 keep_last: 4 }];
```

### Proto3 Optional フィールド

Proto3 `optional` フィールドは Go でポインタセマンティクスを使用します：

```protobuf
message User {
    optional string email = 1 [(redact.v3.value).string = "r*d@ct*d"];
    optional int32 age    = 2 [(redact.v3.value).int32 = 0];
}
```

生成されるコードはポインタ代入を正しく処理します：
```go
tmp := "r*d@ct*d"
x.Email = &tmp
```

### Oneof フィールド

タイプセーフな switch 文を生成します：

```protobuf
message OneofMessage {
    oneof contact {
        string email = 1 [(redact.v3.value).string = "r*d@ct*d"];
        string phone = 2 [(redact.v3.value).mask = { keep_first: 3 keep_last: 4 }];
    }
}
```

生成されるコード：
```go
switch v := x.Contact.(type) {
case *OneofMessage_Email:
    v.Email = "r*d@ct*d"
case *OneofMessage_Phone:
    v.Phone = _redactMask(v.Phone, 3, 4, "*")
}
```

### 正規表現マスキング (Regex)

正規表現で部分マスキングを行います。キャプチャグループは `${1}`、`${2}` で参照できます：

```protobuf
string phone = 1 [(redact.v3.value).regex = {
    pattern: "^(\\d{3})\\d{4}(\\d{4})$"
    replacement: "${1}****${2}"
}];
// 13812345678 → 138****5678
```

### 位置ベースマスク (Mask)

先頭 N 文字と末尾 M 文字を保持し、残りをマスク文字で置換します：

```protobuf
string phone   = 1 [(redact.v3.value).mask = { keep_first: 3 keep_last: 4 }];
// 13812345678 → 138****5678

string id_card = 2 [(redact.v3.value).mask = { keep_first: 6 keep_last: 4 mask_char: "X" }];
// 110101199001011234 → 110101XXXXXXXX1234
```

| パラメータ | 説明 | デフォルト |
|-----------|-------------|---------|
| `keep_first` | 先頭に保持する文字数 | 0 |
| `keep_last` | 末尾に保持する文字数 | 0 |
| `mask_char` | マスク文字 | `"*"` |

### メールマスキング (Email)

`@` で分割し、ローカル部分とドメインを個別にマスキングします：

```protobuf
string email  = 1 [(redact.v3.value).email = { keep_local_first: 2 }];
// alice@example.com → al***@example.com

string email2 = 2 [(redact.v3.value).email = { keep_local_first: 1 mask_domain: true }];
// bob@test.com → ***@********
```

| パラメータ | 説明 | デフォルト |
|-----------|-------------|---------|
| `keep_local_first` | ローカル部分の先頭に保持する文字数 | 0 |
| `mask_domain` | ドメイン部分をマスキングするか | `false` |
| `mask_char` | マスク文字 | `"*"` |

### 切り詰め (Truncate)

先頭 N 文字のみを保持し、オプションで接尾辞を追加します：

```protobuf
string name = 1 [(redact.v3.value).truncate = { length: 1 suffix: "**" }];
// Alexander → A**
```

| パラメータ | 説明 | デフォルト |
|-----------|-------------|---------|
| `length` | 保持する文字数 | — |
| `suffix` | 切り詰め後に追加する接尾辞 | `"..."` |

### ハッシュ (Hash)

フィールド値をハッシュダイジェスト（16進数）で置換します：

```protobuf
string token = 1 [(redact.v3.value).hash = { algo: SHA256 }];

repeated string tokens = 2 [(redact.v3.value).element.item.hash = { algo: MD5 }];
```

| アルゴリズム | 出力長 |
|-----------|--------|
| `MD5` | 32文字 |
| `SHA1` | 40文字 |
| `SHA256` | 64文字 |

### UUID 置換

フィールド値を決定論的 UUID v5（SHA-1 ベース）で置換します。同じ入力は常に同じ UUID を生成します：

```protobuf
string user_id = 1 [(redact.v3.value).uuid = {}];
// alice@example.com → a2b4c6d8-e9f0-5a1b-8c2d-3e4f5a6b7c8d
```

### IP アドレスマスキング

IP アドレス（IPv4 / IPv6）をマスキングし、先頭 N オクテットを保持します：

```protobuf
string client_ip = 1 [(redact.v3.value).ip = { keep_octets: 2 }];
// 192.168.1.100 → 192.168.x.x
```

| パラメータ | 説明 | デフォルト |
|-----------|-------------|---------|
| `keep_octets` | 保持する先頭オクテット数（IPv4）/ ヘクステット数（IPv6） | 2 |
| `mask_char` | マスク文字 | `"x"` |

### URL マスキング

URL のクエリパラメータ値をマスキングします：

```protobuf
string callback = 1 [(redact.v3.value).url = { mask_query: true }];
// https://api.example.com/cb?token=secret → ...?token=******
```

### 固定長マスク (FixedLength)

値全体を同文字数のマスクで置換します：

```protobuf
string bank_account = 1 [(redact.v3.value).fixed_length = { char: "X" }];
// 6225880123456789 → XXXXXXXXXXXXXXXX
```

### カスタムマスキング (Custom)

実行時に登録されたカスタムマスキング関数を呼び出します：

```go
import "github.com/menta2k/protoc-gen-redact/v3/redact/v3"

func init() {
    redact.RegisterCustomRedactor("myRedactor", func(s string) string {
        return "***" + s[len(s)-4:]
    })
}
```

```protobuf
string ssn = 1 [(redact.v3.value).custom = { name: "myRedactor" }];
// 123456789 → ***6789
```

### 条件付きマスキング (Condition)

環境変数が条件を満たした場合のみマスキングを適用します：

```protobuf
string phone = 1 [(redact.v3.value).condition = {
    env_var: "APP_ENV"
    env_val: "production"
    rules: { mask: { keep_first: 3 keep_last: 4 } }
}];
// APP_ENV=production の場合のみマスキング
```

---

## ファイルレベル自動検出 (AutoDetect)

フィールド名のパターンマッチングで自動的にマスキングルールを適用します：

```protobuf
option (redact.v3.auto_detect) = {
    patterns: ["password", "token", "secret", "api_key"]
    default_action: { mask: { keep_first: 2 keep_last: 2 } }
};

message LoginRequest {
    string username = 1;  // 一致なし — マスキングなし
    string password = 2;  // "password" に一致 → 自動マスキング
    string api_key  = 3;  // "api_key" に一致 → 自動マスキング
}
```

マッチングは大文字小文字を区別しない部分一致です。明示的なルールが設定されたフィールドは上書きされません。

---

## メッセージレベルオプション

```protobuf
message PublicData {
    option (redact.v3.ignored) = true;
    string data = 1;
}

message SensitiveData {
    option (redact.v3.nil) = true;
    string secret = 1;
}

message EmptyData {
    option (redact.v3.empty) = true;
    string field1 = 1;
}
```

---

## サービス・メソッドレベルオプション

```protobuf
service MyService {
    rpc GetUser(GetUserRequest) returns (User);

    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse) {
        option (redact.v3.method_skip) = true;
    }

    rpc AdminOperation(AdminRequest) returns (AdminResponse) {
        option (redact.v3.internal_method) = true;
    }
}
```

---

## カスタムテンプレート

```bash
protoc \
  --plugin=protoc-gen-redact=/path/to/protoc-gen-redact \
  --redact_out=. \
  --redact_opt=template_file=/path/to/your/template.tmpl \
  your_proto_file.proto
```

詳細は [examples/CUSTOM_TEMPLATE.md](examples/CUSTOM_TEMPLATE.md) を参照してください。

---

## Buf 設定

本プロジェクトは [Buf](https://buf.build/) を使用して、Protobuf ファイルの管理、lint チェック、破壊的変更の検出、コード生成を行います。プロジェクトルートには以下の buf 設定ファイルが含まれています：

### buf.yaml - モジュール設定

Buf モジュール、lint ルール、破壊的変更ポリシーを定義します：

```yaml
version: v1
name: buf.build/menta2k-org/redact
breaking:
  use:
    - FILE
lint:
  use:
    - STANDARD
```

| フィールド | 説明 |
|------|------|
| `name` | Buf モジュールの一意識別子。Buf Schema Registry (BSR) へのプッシュに使用 |
| `breaking.use` | 破壊的変更チェックレベル。`FILE` はファイルレベルで API 互換性をチェック |
| `lint.use` | lint ルールセット。`STANDARD` は Buf 推奨の標準ルール |

### buf.gen.yaml - コード生成設定

コード生成プラグインの設定を定義します。現在の設定は Go protobuf と gRPC コードを生成します：

```yaml
version: v1
plugins:
  # Generate Go protobuf code
  - plugin: go
    out: .
    opt:
      - paths=source_relative

  # Generate Go gRPC code
  - plugin: go-grpc
    out: .
    opt:
      - paths=source_relative
```

Buf 経由でマスキングコードも生成するには、`plugins` 配下に `redact` プラグインを追加します（事前に `protoc-gen-redact` のインストールが必要）：

```yaml
version: v1
plugins:
  - plugin: go
    out: .
    opt:
      - paths=source_relative

  - plugin: go-grpc
    out: .
    opt:
      - paths=source_relative

  # マスキングコードを生成（protoc-gen-redact のインストールが必要）
  - plugin: redact
    out: .
    opt:
      - paths=source_relative
```

設定後、`buf generate` を実行するだけで、protobuf、gRPC、マスキングコードを一度に生成できます。

### .bufignore - 除外ファイル

Buf が除外するディレクトリを指定し、サンプルやテストデータに対する lint チェックを回避します：

```
# Test data - examples and tests can have non-standard formatting
testdata
examples
```

### buf.lock - 依存関係ロック

Buf が自動生成します。手動で編集しないでください。

### 一般的な Buf コマンド

本プロジェクトは Makefile 経由で一般的な Buf コマンドをラップしています：

```bash
make buf-lint                      # proto ファイルを lint
make buf-format                    # proto ファイルをフォーマット
make buf-breaking                  # 破壊的変更をチェック（main ブランチと比較）
make buf-generate                  # コードを生成
make buf-push                      # Buf Schema Registry にプッシュ
make buf-push-tag TAG=v1.0.0       # タグ付きで BSR にプッシュ
```

Buf コマンドを直接使用することもできます：

```bash
buf lint                           # proto ファイルを lint
buf format -w                      # proto ファイルをフォーマット（-w でファイルに書き込み）
buf breaking --against '.git#branch=main'  # 破壊的変更をチェック
buf generate                       # コードを生成
buf push                           # BSR にプッシュ
```

---

## 開発と CI/CD

```bash
make help           # 全ターゲットを表示
make fmt            # コードフォーマット
make lint           # リンター実行
make test           # 全テスト実行
make build          # プラグインをビルド
make pre-commit     # fmt + lint + test-short
make ci-full        # フル CI パイプライン
```

---

## コントリビュート

コントリビューションを歓迎します！PR をお気軽に提出してください。提出前にすべてのテストがパスすることを確認してください。

---

## ライセンスと帰属

[Apache License 2.0](./LICENSE) の下でライセンスされています。

- Copyright 2020 Shivam Rathore（オリジナル作品）
- Copyright 2025 Contributors（変更）

詳細は [NOTICE](./NOTICE) を参照してください。
