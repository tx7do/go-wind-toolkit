# protoc-gen-wind-errors

Wind 版错误代码生成器：从带 kratos 风格 `(errors.code)` / `(errors.default_code)` 注解的错误枚举生成 **WindError 型** helper（`github.com/tx7do/go-wind/errors`），替代 kratos 官方 `protoc-gen-go-errors` 的 kratos `*errors.Error` 产物。输入 proto 与 kratos 官方生成器完全兼容（同一份 `errors/errors.proto` 注解，buf 依赖 `buf.build/kratos/apis`）。二进制名刻意与 kratos 官方插件区分，避免 GOBIN 撞名。

Wind error generator: emits **WindError-typed** helpers from error enums annotated with the kratos-style `(errors.code)` / `(errors.default_code)` options — a drop-in wind replacement for the kratos `protoc-gen-go-errors` output. Input protos are identical to what the kratos generator consumes.

## 生成契约 / Generated contract

对每个"错误枚举"（枚举带 `errors.default_code`，或任一值带 `errors.code`）的每个值生成一对函数 / For every value of an error enum, a pair of functions is generated:

| 函数 / Function | 说明 / Description |
|---|---|
| `IsXxx(err error) bool` | reason + code 双判定，与 v1 kratos 产物同形 / reason + code match, same shape as the kratos output |
| `ErrorXxx(format string, args ...interface{}) *windErrors.WindError` | 构造错误 / constructs the error |

关键语义 / Key semantics:

- `(errors.code)` 注解的是 **HTTP 状态码**，直写 `WindError.Code`，由 `go-wind/errors` 的 `CodeToHTTP` 400–599 直通段在传输层**原样复现**——与 v1 kratos 产物（`errors.New(code, …)` 的 code 即 HTTP 状态）契约一致。/ The `(errors.code)` value is an HTTP status, stored directly in `Code` and reproduced verbatim by `CodeToHTTP`'s 400–599 passthrough — matching the v1 kratos contract where the code IS the HTTP status.
- `ErrorXxx` 的格式化文案经 `WithCause` 挂入错误链：日志可读（`Error()` 含 cause），线上响应体保持 `{reason, details}`——**后端 message 不出网**（前端仅按 reason 本地化）。/ The formatted message is attached via `WithCause`: readable in logs (`Error()` includes the cause), while the wire body stays `{reason, details}` — the backend message never leaves the service.
- 扩展值用 `protowire` 从 options 原始字节读取（字段号 1108/1109），本生成器**不链接 kratos 模块**。/ Option values are read with `protowire` from the raw options bytes (field numbers 1108/1109); the generator never links the kratos module.

## 用法 / Usage

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-wind-errors@latest
```

```yaml
# buf.gen.yaml (v2)
plugins:
  - local: protoc-gen-wind-errors
    out: gen/go
    opt:
      - paths=source_relative
```

生成的文件名与 kratos 官方产物一致（`<proto>_errors.pb.go`），v2 分支重生成时可直接原位替换；调用侧 `IsXxx` 无感，`ErrorXxx` 返回类型从 kratos `*errors.Error` 变为 `*windErrors.WindError`（以 error 返回的调用点零改动）。

The output file name matches the kratos generator (`<proto>_errors.pb.go`), so regeneration replaces it in place on a v2 branch. `IsXxx` call sites are unaffected; `ErrorXxx` returns `*windErrors.WindError` instead of kratos `*errors.Error` (call sites returning it as `error` need no change).

## 前端 i18n 词表 / Frontend i18n vocabularies

设置 `i18n_langs` 参数后，除 Go 文件外还会为每个含错误枚举的 proto 文件输出一份
reason → `{code, message}` 的 JSON 词表，供前端按 reason 本地化（与"后端 message
不出网"契约配套）：

With the `i18n_langs` parameter, a JSON vocabulary (`reason` → `{code, message}`) is
emitted alongside the Go file for frontend localization by reason:

```yaml
# buf.gen.yaml (v2) — 注意:protogen 按逗号切分参数,值内分隔符用 ";"
# Note: protogen splits plugin params on commas; separate locales with ";"
plugins:
  - local: protoc-gen-wind-errors
    out: gen/go
    opt:
      - paths=source_relative
      - i18n_langs=zh;en
```

产物 / Output (`<proto>_errors_i18n.<lang>.json`):

```json
{
  "USER_NOT_FOUND": {
    "code": 404,
    "message": "满足请求的资源不存在"
  }
}
```

文案来源是枚举值的注释：正文按 `|` 分段，段序与 `i18n_langs` 声明序位置对应
（段1→第一个语言，段2→第二个语言）；缺段为空串，留给译者补齐。注释里不含 `|`
时所有语言共用该段文本。/ Messages come from the enum value comments: the comment
text is split on `|`, segments map positionally to the declared locales (missing
segments are empty strings); a comment without `|` is shared by every locale.
