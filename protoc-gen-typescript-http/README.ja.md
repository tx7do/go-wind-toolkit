# protoc-gen-typescript-http

[简体中文](./README.md) | [English](./README.en.md) | 日本語

[HTTP ルール](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto)アノテーション付きの Protobuf 定義から TypeScript 型とサービスクライアントを生成します。生成される型は [canonical JSON エンコーディング](https://developers.google.com/protocol-buffers/docs/proto3#json)仕様に準拠します。

## 機能

- `.proto` ファイルから TypeScript 型と型安全なサービスクライアントを生成
- Unary、Server-Streaming、Bidirectional-Streaming RPC をサポート
- 生成コードは**純粋なインターフェースのみ**を出力 — HTTP / SSE / WebSocket トランスポートの実装は呼び出し側が完全に注入
- `google.api.default_host` による `DEFAULT_HOST` 定数の自動生成をサポート
- Protobuf canonical JSON エンコーディングに準拠
- int64/uint64（およびその wrapper 型 Int64Value/UInt64Value）は `number` にマッピングされます：canonical JSON は 64 ビット整数を 10 進文字列としてエンコードしますが、フロントエンドの利便性のため `number` にマッピングします（2^53 を超える値は精度が失われるため、無損失が必要な場合は proto 側で string フィールドを使用してください）

## インストール

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-typescript-http@latest
```

または [releases](../../releases) からプリビルド済みバイナリをダウンロードしてください。

## 呼び出し方法

```bash
protoc \
  --typescript-http_out [出力ディレクトリ] \
  [.proto ファイル ...]
```

---

## アーキテクチャ

生成コードは3つのメソッドを持つ `ClientTransport` インターフェースを定義します：

| メソッド | 用途 | RPC タイプ |
| --- | --- | --- |
| `unary()` | 通常のリクエスト/レスポンス | Unary RPC |
| `serverStream()` | サーバーストリーミングプッシュ（`ServerStream<T>` を返す） | `returns (stream T)` |
| `duplexStream()` | 双方向ストリーミング（`DuplexStream<TIn, TOut>` を返す） | `stream T returns (stream U)` |
| `byteStream()` | 生バイトチャンクストリーム（`ServerStream<Uint8Array>` を返す） | `returns (stream google.api.HttpBody)` |

**呼び出し側が `ClientTransport` の実装を提供する必要があります。** これにより、HTTP クライアント（fetch、Axios など）、認証ヘッダー、SSE トランスポート、WebSocket トランスポートを完全に制御できます。

完全な例は [examples](./examples) を参照してください。

## 基本的な使い方

`ClientTransport` インターフェースを実装し、生成されたクライアントファクトリに渡します：

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

// Unary 呼び出し
const shipper = await client.GetShipper({ name: "shippers/123" });
```

## google.api.default_host の使用

proto service に `google.api.default_host` オプションが定義されている場合、`DEFAULT_HOST` 定数が自動的に生成されます：

```protobuf
service ShipperService {
  option (google.api.default_host) = "api.example.com";
  // ...
}
```

この定数はエクスポートされるため、トランスポート構築時に参照できます：

```typescript
import { DEFAULT_HOST, createShipperServiceClient } from "./gen";

const baseUrl = `https://${DEFAULT_HOST}`;
```

## additional_bindings のサポート

`google.api.http` の `additional_bindings` はそれぞれ独立したメンバーとして生成され、`<Method>Alt<N>` と命名されます（N は 1 から始まり、注釈内の宣言順に採番）。インターフェース宣言とクライアント実装は同時に生成されます:

```protobuf
rpc CreateBook(CreateBookRequest) returns (Book) {
  option (google.api.http) = {
    post: "/v1/{parent=publishers/*}/books"
    body: "book"
    additional_bindings {
      post: "/v1/books"
      body: "book"
    }
    additional_bindings {
      get: "/v1/books/{isbn}"
    }
  };
}
```

```typescript
// 生成コード（各バインディングが独自のパス検証・body 選択・クエリパラメータ導出を行う）
interface BookService {
  CreateBook(request: CreateBookRequest): Promise<Book>;     // POST /v1/{parent}/books
  CreateBookAlt1(request: CreateBookRequest): Promise<Book>; // POST /v1/books
  CreateBookAlt2(request: CreateBookRequest): Promise<Book>; // GET /v1/books/{isbn}
}
```

注意:

- 各バリアントのコメントには HTTP メソッドとルートテンプレートが明記されます
- パス検証はバインディングごとに独立して適用されます（上の例では `CreateBookAlt2` のみが `isbn` 非空を要求します）
- ストリーミング RPC は主バインディングのみ対応し、追加バインディングは警告付きでスキップされます

## ストリーミング通信

サーバーストリーミング RPC（`returns (stream ...)`）と双方向ストリーミング RPC（`stream ... returns (stream ...)`）は、それぞれ `ClientTransport` の `serverStream()` メソッドと `duplexStream()` メソッドでサポートされます。

生成コードは **`ServerStream<T>` と `DuplexStream<TIn, TOut>` インターフェースのみを定義**します — 実際のトランスポート実装（`fetch` + `ReadableStream` を使った SSE、`EventSource`、WebSocket など）は呼び出し側が提供します。

ストリーム要素が [`google.api.HttpBody`](https://google.aip.dev/134#httpbody) のサーバーストリーミング RPC（ファイルダウンロードなど）は専用の `byteStream()` を使用します：クライアントは `ServerStream<Uint8Array>` の生バイトチャンクを受け取り、**JSON デコードは行いません** — protoc-gen-go-http のサーバー契約（最初のチャンクの Content-Type は HTTP レスポンスヘッダーに載り、各チャンクは生ペイロードバイト）と一致します。

### Proto の例

```protobuf
service LogService {
  rpc GetLog(GetLogRequest) returns (GetLogResponse) {
    option (google.api.http) = {get: "/v1/{name=logs/*}"};
  }

  // サーバーストリーミング
  rpc TailLogs(TailLogsRequest) returns (stream LogEntry) {
    option (google.api.http) = {get: "/v1/{name=logs/*}:tail"};
  }

  // 双方向ストリーミング
  rpc Chat(stream ChatMessage) returns (stream ChatMessage) {
    option (google.api.http) = {get: "/v1/chat"};
  }
}
```

### ServerStream の実装（fetch + ReadableStream による SSE）

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

### トランスポートの渡し方

```typescript
const transport: ClientTransport = {
  // ...unary の実装...
  serverStream<T>(path, _meta) {
    return new FetchSSETransport<T>(`https://api.example.com/${path}`);
  },
  duplexStream<TIn, TOut>(path, _meta) {
    // WebSocket ベースの DuplexStream 実装を返す
  },
};

const client = createLogServiceClient(transport);

// サーバーストリーミング
const tail = client.TailLogs({ name: "log/123" });
tail.onEvent((entry) => console.log(entry.message));
tail.onError((err) => console.error(err));
// tail.close();

// 双方向ストリーミング
const chat = client.Chat();
chat.onEvent((msg) => console.log(msg.text));
chat.send({ text: "hello" });
// chat.close();
```

## 統合 ApiClient

1 つの proto package に複数の service が含まれる場合、すべてのサービスクライアントを集約する `ApiClient` クラスが生成されます。トランスポートは一度だけ渡します：

```typescript
import { ApiClient, ClientTransport } from "./gen";

const transport: ClientTransport = { /* ... */ };
const api = new ApiClient(transport);
// または: const api = createApiClient(transport);

// 各サービスに遅延初期化でアクセス
const shipper = await api.shipperService.GetShipper({ name: "shippers/123" });
```

## ライセンス

[MIT](./LICENSE)
