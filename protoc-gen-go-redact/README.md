protoc-gen-redact (PGR)
=======================

**[中文](README.md)** | [English](README_EN.md) | [日本語](README_JA.md)

[![Build and Publish](https://github.com/menta2k/protoc-gen-redact/workflows/Build%20and%20Publish/badge.svg)](https://github.com/menta2k/protoc-gen-redact/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/menta2k/protoc-gen-redact/v3?dropcache)](https://goreportcard.com/report/github.com/menta2k/protoc-gen-redact/v3)
[![Go Reference](https://pkg.go.dev/badge/github.com/menta2k/protoc-gen-redact/v3.svg)](https://pkg.go.dev/github.com/menta2k/protoc-gen-redact/v3)
[![License](https://img.shields.io/badge/license-apache2-mildgreen.svg)](./LICENSE)
[![GitHub release](https://img.shields.io/github/release/menta2k/protoc-gen-redact.svg)](https://github.com/menta2k/protoc-gen-redact/releases)

_protoc-gen-redact (PGR)_ 是一个 protoc 插件，用于在服务端对 gRPC 调用中的字段值进行自动脱敏。

---

## 目录

- [归属说明](#归属说明)
- [快速开始](#快速开始)
- [安装](#安装)
- [字段级脱敏规则](#字段级脱敏规则)
  - [标量字段](#标量字段)
  - [消息字段](#消息字段)
  - [Repeated / Map 字段](#repeated--map-字段)
  - [Proto3 Optional 字段](#proto3-optional-字段)
  - [Oneof 字段](#oneof-字段)
  - [正则脱敏 (Regex)](#正则脱敏-regex)
  - [位置遮罩 (Mask)](#位置遮罩-mask)
  - [邮箱脱敏 (Email)](#邮箱脱敏-email)
  - [截断脱敏 (Truncate)](#截断脱敏-truncate)
  - [哈希脱敏 (Hash)](#哈希脱敏-hash)
  - [UUID 替换](#uuid-替换)
  - [IP 地址脱敏](#ip-地址脱敏)
  - [URL 脱敏](#url-脱敏)
  - [等长掩码 (FixedLength)](#等长掩码-fixedlength)
  - [自定义脱敏 (Custom)](#自定义脱敏-custom)
  - [条件脱敏 (Condition)](#条件脱敏-condition)
- [文件级自动检测 (AutoDetect)](#文件级自动检测-autodetect)
- [消息级选项](#消息级选项)
- [服务与方法级选项](#服务与方法级选项)
- [自定义模板](#自定义模板)
- [Buf 配置](#buf-配置)
- [开发与 CI/CD](#开发与-cicd)
- [贡献指南](#贡献指南)
- [许可证与归属](#许可证与归属)

---

## 归属说明

本项目基于 **Shivam Rathore**（Copyright 2020）的原始项目 [protoc-gen-redact](https://github.com/arrakis-digital/protoc-gen-redact) 衍生而来。

- **原作者：** Shivam Rathore
- **原始项目：** https://github.com/arrakis-digital/protoc-gen-redact
- **贡献者：** John Castronuovo

本分支包含以下增强和改进：
- 全面的错误处理和验证系统
- 完整的测试套件（374+ 测试用例）
- Oneof 字段支持（类型安全的 switch 语句生成）
- Proto3 optional 字段支持（正确的指针语义）
- 自定义模板文件支持
- 集成测试（真实 protoc 编译）
- 15 种脱敏规则（正则、遮罩、邮箱、截断、哈希、UUID、IP、URL、等长掩码、自定义、条件等）
- 文件级自动检测（按字段名自动匹配脱敏规则）
- 按需条件生成 `.pb.redact.go` 文件

所有修改遵循 Apache License 2.0 许可证，与原始项目保持一致。

---

## 快速开始

只需导入 PGR 扩展并在 proto 文件中为消息或字段添加注解即可：

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

## 安装

```bash
go install github.com/menta2k/protoc-gen-redact/v3@latest
```

---

## 字段级脱敏规则

### 标量字段

为各字段指定自定义脱敏值：

```protobuf
string password = 1 [(redact.v3.value).string = "REDACTED"];
int32  age      = 2 [(redact.v3.value).int32 = 0];
bool   active   = 3 [(redact.v3.value).bool = false];
bytes  sign     = 4 [(redact.v3.value).bytes = ""];
double score    = 5 [(redact.v3.value).double = 0.0];
```

支持所有 proto 标量类型：`float`、`double`、`int32`、`int64`、`uint32`、`uint64`、`sint32`、`sint64`、`fixed32`、`fixed64`、`sfixed32`、`sfixed64`、`bool`、`string`、`bytes`、`enum`。

### 消息字段

控制嵌套消息的脱敏行为：

```protobuf
// 递归应用脱敏规则到嵌套消息的字段
Profile profile = 1 [(redact.v3.value).message.apply = true];

// 将整个消息设为 nil
Settings settings = 2 [(redact.v3.value).message.nil = true];

// 替换为空实例
Metadata metadata = 3 [(redact.v3.value).message.empty = true];

// 完全跳过该字段的脱敏
AuditLog log = 4 [(redact.v3.value).message.skip = true];
```

### Repeated / Map 字段

```protobuf
// 清空集合
map<string, string> attributes = 1 [(redact.v3.value).element.empty = true];

// 对每个元素应用默认脱敏
repeated Address addresses = 2 [(redact.v3.value).element.nested = true];

// 对每个元素应用自定义脱敏规则
repeated int32 scores = 3 [(redact.v3.value).element.item.int32 = 0];
repeated string phones = 4 [(redact.v3.value).element.item.mask = { keep_first: 3 keep_last: 4 }];
```

### Proto3 Optional 字段

Proto3 `optional` 字段在 Go 中使用指针语义。生成器会正确处理指针赋值：

```protobuf
message User {
    optional string email = 1 [(redact.v3.value).string = "r*d@ct*d"];
    optional int32 age    = 2 [(redact.v3.value).int32 = 0];
}
```

生成的代码正确使用指针赋值：
```go
tmp := "r*d@ct*d"
x.Email = &tmp
```

### Oneof 字段

生成器支持 `oneof` 分组，生成类型安全的 switch 语句。只有标注了脱敏的字段才会生成对应的 case 分支：

```protobuf
message OneofMessage {
    oneof contact {
        string email = 1 [(redact.v3.value).string = "r*d@ct*d"];
        string phone = 2 [(redact.v3.value).mask = { keep_first: 3 keep_last: 4 }];
    }
}
```

生成的代码：
```go
switch v := x.Contact.(type) {
case *OneofMessage_Email:
    v.Email = "r*d@ct*d"
case *OneofMessage_Phone:
    v.Phone = _redactMask(v.Phone, 3, 4, "*")
}
```

### 正则脱敏 (Regex)

使用正则表达式进行部分掩码，捕获组可通过 `${1}`、`${2}` 引用：

```protobuf
string phone = 1 [(redact.v3.value).regex = {
    pattern: "^(\\d{3})\\d{4}(\\d{4})$"
    replacement: "${1}****${2}"
}];
// 13812345678 → 138****5678

repeated string id_cards = 2 [(redact.v3.value).element.item.regex = {
    pattern: "(\\d{4})\\d{10}(\\d{4})"
    replacement: "${1}**********${2}"
}];
```

### 位置遮罩 (Mask)

保留首尾指定数量的字符，中间用掩码字符替换：

```protobuf
string phone    = 1 [(redact.v3.value).mask = { keep_first: 3 keep_last: 4 }];
// 13812345678 → 138****5678

string id_card  = 2 [(redact.v3.value).mask = { keep_first: 6 keep_last: 4 mask_char: "X" }];
// 110101199001011234 → 110101XXXXXXXX1234

repeated string emails = 3 [(redact.v3.value).element.item.mask = { keep_first: 2 keep_last: 0 }];
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `keep_first` | 保留开头的字符数 | 0 |
| `keep_last` | 保留结尾的字符数 | 0 |
| `mask_char` | 掩码字符 | `"*"` |

### 邮箱脱敏 (Email)

按 `@` 分割，分别对本地部分和域名进行掩码：

```protobuf
string email  = 1 [(redact.v3.value).email = { keep_local_first: 2 }];
// alice@example.com → al***@example.com

string email2 = 2 [(redact.v3.value).email = { keep_local_first: 1 mask_domain: true }];
// bob@test.com → ***@********
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `keep_local_first` | 保留 `@` 前面开头的字符数 | 0 |
| `mask_domain` | 是否掩码域名部分 | `false` |
| `mask_char` | 掩码字符 | `"*"` |

### 截断脱敏 (Truncate)

只保留前 N 个字符，后接可选后缀：

```protobuf
string name = 1 [(redact.v3.value).truncate = { length: 1 suffix: "**" }];
// Alexander → A**

string bio  = 2 [(redact.v3.value).truncate = { length: 2 }];
// HelloWorld → He...
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `length` | 保留的字符数 | — |
| `suffix` | 截断后追加的后缀 | `"..."` |

### 哈希脱敏 (Hash)

将字段值替换为单向哈希摘要（十六进制）：

```protobuf
string token   = 1 [(redact.v3.value).hash = { algo: SHA256 }];
// secret123 → 5f2c...（64 位十六进制）

string session = 2 [(redact.v3.value).hash = { algo: MD5 }];

repeated string tokens = 3 [(redact.v3.value).element.item.hash = { algo: SHA1 }];
```

| 算法 | 输出长度 |
|------|----------|
| `MD5` | 32 字符 |
| `SHA1` | 40 字符 |
| `SHA256` | 64 字符 |

### UUID 替换

将字段值替换为确定性 UUID v5（基于 SHA-1 哈希生成）。相同输入始终产生相同 UUID，适合匿名化场景：

```protobuf
string user_id   = 1 [(redact.v3.value).uuid = {}];
// alice@example.com → a2b4c6d8-e9f0-5a1b-8c2d-3e4f5a6b7c8d

repeated string ids = 2 [(redact.v3.value).element.item.uuid = {}];
```

### IP 地址脱敏

掩码 IP 地址（支持 IPv4 和 IPv6），保留前 N 个段：

```protobuf
string client_ip = 1 [(redact.v3.value).ip = { keep_octets: 2 }];
// 192.168.1.100 → 192.168.x.x

string server_ip = 2 [(redact.v3.value).ip = { keep_octets: 3 mask_char: "0" }];
// 10.0.0.1 → 10.0.0.0

string ipv6_addr = 3 [(redact.v3.value).ip = { keep_octets: 4 }];
// 2001:db8::1 → 2001:db8:0:0:x:x:x:x
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `keep_octets` | 保留的前段数（IPv4 为 octet，IPv6 为 hextet） | 2 |
| `mask_char` | 掩码字符 | `"x"` |

### URL 脱敏

掩码 URL 的查询参数值：

```protobuf
string callback = 1 [(redact.v3.value).url = { mask_query: true }];
// https://api.example.com/cb?token=secret123 → https://api.example.com/cb?token=*********

string link = 2 [(redact.v3.value).url = { mask_query: true mask_char: "#" }];
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `mask_query` | 是否掩码所有查询参数值 | — |
| `mask_char` | 掩码字符 | `"*"` |

### 等长掩码 (FixedLength)

用等长掩码替换整个值：

```protobuf
string bank_account = 1 [(redact.v3.value).fixed_length = { char: "X" }];
// 6225880123456789 → XXXXXXXXXXXXXXXX

string card = 2 [(redact.v3.value).fixed_length = { char: "#" }];
// 1234 → ####
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `char` | 掩码字符 | `"X"` |

### 自定义脱敏 (Custom)

调用运行时注册的自定义脱敏函数。通过 `redact.RegisterCustomRedactor` 注册：

```go
import "github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact"

func init() {
    redact.RegisterCustomRedactor("myRedactor", func(s string) string {
        // 自定义脱敏逻辑
        return "***" + s[len(s)-4:]
    })
}
```

```protobuf
string ssn = 1 [(redact.v3.value).custom = { name: "myRedactor" }];
// 123456789 → ***6789
```

### 条件脱敏 (Condition)

根据环境变量决定是否执行脱敏，内部可包裹任意其他规则：

```protobuf
string phone = 1 [(redact.v3.value).condition = {
    env_var: "APP_ENV"
    env_val: "production"
    rules: { mask: { keep_first: 3 keep_last: 4 } }
}];
// APP_ENV=production 时脱敏，否则不处理

string debug_data = 2 [(redact.v3.value).condition = {
    env_var: "DEBUG"
    env_val: ""
    rules: { hash: { algo: SHA256 } }
}];
// DEBUG 变量存在（任意非空值）时执行哈希脱敏
```

| 参数 | 说明 |
|------|------|
| `env_var` | 环境变量名 |
| `env_val` | 期望值（为空时检查变量是否存在且非空） |
| `rules` | 条件满足时应用的脱敏规则 |

---

## 文件级自动检测 (AutoDetect)

通过文件级选项，按字段名自动匹配并应用脱敏规则，无需逐个标注：

```protobuf
syntax = "proto3";

import "redact/v3/redact.proto";

option (redact.v3.auto_detect) = {
    patterns: ["password", "token", "secret", "api_key"]
    default_action: { mask: { keep_first: 2 keep_last: 2 } }
};

message LoginRequest {
    string username    = 1;  // 不匹配，不脱敏
    string password    = 2;  // 匹配 "password" → 自动脱敏
    string api_key     = 3;  // 匹配 "api_key" → 自动脱敏
    string session_id  = 4;  // 不匹配，不脱敏
}
```

已显式标注脱敏规则的字段不会被 AutoDetect 覆盖。匹配为大小写不敏感的子串匹配。

---

## 消息级选项

控制整个消息的脱敏行为：

```protobuf
// 跳过该消息的所有脱敏
message PublicData {
    option (redact.v3.ignored) = true;
    string data = 1;
}

// 始终设为 nil
message SensitiveData {
    option (redact.v3.nil) = true;
    string secret = 1;
}

// 替换为空实例
message EmptyData {
    option (redact.v3.empty) = true;
    string field1 = 1;
}
```

---

## 服务与方法级选项

在服务和方法级别控制脱敏行为：

```protobuf
service MyService {
    // 普通 RPC，自动脱敏响应
    rpc GetUser(GetUserRequest) returns (User);

    // 跳过该方法的脱敏
    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse) {
        option (redact.v3.method_skip) = true;
    }

    // 标记为内部方法（客户端收到 PermissionDenied 错误）
    rpc AdminOperation(AdminRequest) returns (AdminResponse) {
        option (redact.v3.internal_method) = true;
    }
}
```

也可在服务级别设置：
- `service_skip`：跳过整个服务的脱敏
- `internal_service`：标记整个服务为内部服务
- `internal_service_code`：自定义错误码（默认 PermissionDenied）
- `internal_service_err_message`：自定义错误消息（支持 `%service%` 和 `%method%` 占位符）

---

## 自定义模板

PGR 支持使用自定义模板进行代码生成：

```bash
protoc \
  --plugin=protoc-gen-redact=/path/to/protoc-gen-redact \
  --redact_out=. \
  --redact_opt=template_file=/path/to/your/template.tmpl \
  your_proto_file.proto
```

示例模板见 [examples/custom-template.tmpl](examples/custom-template.tmpl)。

完整文档请参考 [examples/CUSTOM_TEMPLATE.md](examples/CUSTOM_TEMPLATE.md)。

---

## Buf 配置

本项目使用 [Buf](https://buf.build/) 进行 Protobuf 文件的管理、lint 检查、破坏性变更检测和代码生成。项目根目录包含以下 buf 配置文件：

### buf.yaml - 模块配置

定义 Buf 模块、lint 规则和破坏性变更检查策略：

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

| 字段 | 说明 |
|------|------|
| `name` | Buf 模块的唯一标识，用于推送到 Buf Schema Registry (BSR) |
| `breaking.use` | 破坏性变更检查级别，`FILE` 表示在文件级别检查 API 兼容性 |
| `lint.use` | lint 规则集，`STANDARD` 为 Buf 推荐的标准规则 |

### buf.gen.yaml - 代码生成配置

定义代码生成插件的配置。当前配置生成 Go protobuf 和 gRPC 代码：

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

若需通过 Buf 一并生成脱敏代码，可在 `plugins` 下添加 `redact` 插件配置（需先安装 `protoc-gen-redact`）：

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

  # 生成脱敏代码（需先安装 protoc-gen-redact）
  - plugin: redact
    out: .
    opt:
      - paths=source_relative
```

配置完成后，运行 `buf generate` 即可一次性生成 protobuf、gRPC 和脱敏代码。

### .bufignore - 忽略文件

指定 Buf 忽略的目录，避免对示例和测试数据进行 lint 检查：

```
# Test data - examples and tests can have non-standard formatting
testdata
examples
```

### buf.lock - 依赖锁定

由 Buf 自动生成，请勿手动编辑。

### 常用 Buf 命令

本项目通过 Makefile 封装了常用的 Buf 命令：

```bash
make buf-lint                      # Lint proto 文件
make buf-format                    # 格式化 proto 文件
make buf-breaking                  # 检查破坏性变更（与 main 分支对比）
make buf-generate                  # 生成代码
make buf-push                      # 推送到 Buf Schema Registry
make buf-push-tag TAG=v1.0.0       # 带标签推送到 BSR
```

也可以直接使用 Buf 命令：

```bash
buf lint                           # Lint proto 文件
buf format -w                      # 格式化 proto 文件（-w 写入文件）
buf breaking --against '.git#branch=main'  # 检查破坏性变更
buf generate                       # 生成代码
buf push                           # 推送到 BSR
```

---

## 开发与 CI/CD

本项目包含完整的构建系统和 CI/CD 流水线：

```bash
# 查看所有可用目标
make help

# 开发工作流
make fmt              # 格式化代码
make lint             # 运行所有 linter
make test             # 运行所有测试
make test-short       # 快速测试
make build            # 构建插件

# 提交前检查
make pre-commit       # fmt + lint + test-short

# 完整 CI 流水线
make ci-full          # 完整 CI 含覆盖率和 buf 检查
```

---

## 贡献指南

欢迎提交 Pull Request！请在特定分支中进行修改，并向 master 分支发起 PR。

请确保所有更改正常工作，且不影响现有功能。即使是最小的贡献也非常欢迎。

---

## 许可证与归属

本项目基于 [Apache License 2.0](./LICENSE) 许可证。

- Copyright 2020 Shivam Rathore（原始工作）
- Copyright 2025 Contributors（修改）

本项目是基于原始 protoc-gen-redact 项目的衍生作品。所有归属声明、版权声明和许可证条款均已根据 Apache License 2.0 的要求予以保留。

详见 [NOTICE](./NOTICE) 文件。
