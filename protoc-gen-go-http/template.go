// template.go 定义了 HTTP 代码生成所需的内嵌模板与数据结构（serviceDesc / methodDesc），
// 并负责将模板渲染为最终生成的 Go 代码。
//
// This file defines the embedded template and data structures (serviceDesc / methodDesc)
// used for HTTP code generation, and renders the template into the final Go code.
//
// このファイルは HTTP コード生成に必要な組み込みテンプレートとデータ構造
// （serviceDesc / methodDesc）を定義し、テンプレートを最終的な Go コードへレンダリングします。
package main

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"
)

// httpTemplate 是通过 go:embed 嵌入的 HTTP 服务代码生成模板（见 httpTemplate.tpl）。
// httpTemplate is the HTTP server code generation template embedded via go:embed (see httpTemplate.tpl).
// httpTemplate は go:embed によって組み込まれた HTTP サーバーコード生成テンプレートです（httpTemplate.tpl を参照）。
//
//go:embed httpTemplate.tpl
var httpTemplate string

// serviceDesc 描述一个待生成的 HTTP 服务，聚合服务名、proto 元数据与方法列表，
// 是模板渲染时的顶层数据对象。
//
// serviceDesc describes an HTTP service to be generated, aggregating the service name,
// proto metadata and method list; it is the top-level data object used for template rendering.
//
// serviceDesc は生成対象の HTTP サービスを記述し、サービス名、proto メタデータ、メソッド一覧を集約します。
// テンプレートレンダリング時の最上位データオブジェクトです。
type serviceDesc struct {
	ServiceType string // Go 服务类型名，例如 Greeter。 / Go service type name, e.g. Greeter. / Go のサービスタイプ名（例: Greeter）。
	ServiceName string // proto 全限定服务名，例如 helloworld.Greeter。 / Fully-qualified proto service name, e.g. helloworld.Greeter. / proto の完全修飾サービス名（例: helloworld.Greeter）。
	Metadata    string // 源 proto 文件路径，例如 api/helloworld/helloworld.proto。 / Source proto file path, e.g. api/helloworld/helloworld.proto. / ソース proto ファイルのパス（例: api/helloworld/helloworld.proto）。
	Methods     []*methodDesc
	MethodSets  map[string]*methodDesc
}

// methodDesc 描述一个待生成的 HTTP 方法，包含方法签名信息以及由 google.api.HttpRule
// 解析得到的路由、路径变量、请求体与响应体等细节。
//
// methodDesc describes an HTTP method to be generated, including the method signature and
// the route, path variables, request/response body details parsed from google.api.HttpRule.
//
// methodDesc は生成対象の HTTP メソッドを記述し、メソッドシグネチャ情報と、google.api.HttpRule
// から解析したルーティング、パス変数、リクエスト/レスポンスボディ等の詳細を含みます。
type methodDesc struct {
	// ---- 方法签名信息 / Method signature info / メソッドシグネチャ情報 ----
	Name         string // Go 方法名。 / Go method name. / Go のメソッド名。
	OriginalName string // proto 中原始方法名。 / Original method name in proto. / proto 内の元のメソッド名。
	Num          int    // 同名方法的去重编号，用于区分重载绑定。 / De-duplication index for methods with the same name to disambiguate bindings. / 同名メソッドの重複解除用インデックス。
	Request      string // 请求消息的 Go 类型名。 / Go type name of the request message. / リクエストメッセージの Go 型名。
	Reply        string // 响应消息的 Go 类型名。 / Go type name of the reply message. / レスポンスメッセージの Go 型名。
	Comment      string // 方法注释（含废弃标记），原样写入生成代码。 / Method comment (including deprecation), written verbatim into generated code. / メソッドコメント（非推奨表記を含む）。生成コードにそのまま出力される。
	// ---- http_rule 信息 / http_rule info / http_rule 情報 ----
	Path            string // 实际路由路径（路径变量已被规范化）。 / Actual route path (path variables normalized). / 実際のルートパス（パス変数は正規化済み）。
	PathTemplate    string // 原始路由模板，保留路径变量占位符。 / Original route template, keeping path variable placeholders. / 元のルートテンプレート。パス変数のプレースホルダを保持。
	PathVarsList    string // 路径变量名的 Go 切片字面量，例如 []string{"id", "user.name"}。 / Go slice literal of path variable names, e.g. []string{"id", "user.name"}. / パス変数名の Go スライスリテラル（例: []string{"id", "user.name"}）。
	Method          string // HTTP 方法，例如 GET、POST。 / HTTP method, e.g. GET, POST. / HTTP メソッド（例: GET、POST）。
	HasVars         bool   // 是否存在路径变量。 / Whether there are path variables. / パス変数が存在するかどうか。
	HasBody         bool   // 是否声明了请求体。 / Whether a request body is declared. / リクエストボディが宣言されているかどうか。
	Body            string // Go 字段访问器，例如 ".User"，为空表示无字段映射。 / Go field accessor, e.g. ".User"; empty means no field mapping. / Go フィールドアクセサ（例: ".User"）。空はフィールドマッピングなし。
	BodyField       string // proto 原始字段名，"*" 表示整个请求体。 / Raw proto field name; "*" means the whole request body. / proto の生フィールド名。"*" はリクエスト全体を表す。
	BodyQueryName   string // 请求体字段的 JSON 名，用于查询参数绑定。 / JSON name of the body field, used for query binding. / リクエストボディフィールドの JSON 名。クエリバインディングに使用。
	BodyHTTPBody    bool   // 请求体是否为 google.api.HttpBody 类型。 / Whether the request body is a google.api.HttpBody. / リクエストボディが google.api.HttpBody 型かどうか。
	BodyMessage     bool   // 请求体字段是否为单个消息类型（可用于流式分帧）。 / Whether the body field is a singular message type (usable for streaming framing). / リクエストボディフィールドが単一メッセージ型かどうか（ストリーミングフレーム化に使用可）。
	ResponseBody    string // 响应体字段访问器，例如 ".Body"，为空表示返回整个响应。 / Response body field accessor, e.g. ".Body"; empty means return the whole response. / レスポンスボディのフィールドアクセサ（例: ".Body"）。空はレスポンス全体を返す。
	ReplyHTTPBody   bool   // 响应消息是否为 google.api.HttpBody 类型。 / Whether the reply message is a google.api.HttpBody. / レスポンスメッセージが google.api.HttpBody 型かどうか。
	ClientStreaming bool   // 方法是否为客户端流式。 / Whether the method is client-streaming. / メソッドがクライアントストリーミングかどうか。
	ServerStreaming bool   // 方法是否为服务端流式。 / Whether the method is server-streaming. / メソッドがサーバーストリーミングかどうか。
}

// execute 使用内嵌模板 httpTemplate 渲染当前服务描述，返回生成的 Go 代码字符串。
// 渲染前会按方法名构建 MethodSets（同名方法以最后一个为准）。
//
// execute renders the current service description with the embedded httpTemplate and returns
// the generated Go code as a string. MethodSets is built by method name before rendering
// (the last one wins for duplicate names).
//
// execute は組み込みテンプレート httpTemplate で現在のサービス記述をレンダリングし、
// 生成された Go コード文字列を返します。レンダリング前にメソッド名で MethodSets を構築します
// （同名の場合は最後が優先されます）。
func (s *serviceDesc) execute() string {
	s.MethodSets = make(map[string]*methodDesc)
	for _, m := range s.Methods {
		s.MethodSets[m.Name] = m
	}
	buf := new(bytes.Buffer)
	tmpl, err := template.New("http").Parse(strings.TrimSpace(httpTemplate))
	if err != nil {
		panic(err)
	}
	if err := tmpl.Execute(buf, s); err != nil {
		panic(err)
	}
	return strings.Trim(buf.String(), "\r\n")
}
