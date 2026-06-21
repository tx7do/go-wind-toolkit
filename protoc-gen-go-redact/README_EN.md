protoc-gen-redact (PGR)
=======================

[中文](README.md) | **[English](README_EN.md)** | [日本語](README_JA.md)

[![Build and Publish](https://github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact/workflows/Build%20and%20Publish/badge.svg)](https://github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact?dropcache)](https://goreportcard.com/report/github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact)
[![Go Reference](https://pkg.go.dev/badge/github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact.svg)](https://pkg.go.dev/github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact)
[![License](https://img.shields.io/badge/license-apache2-mildgreen.svg)](./LICENSE)
[![GitHub release](https://img.shields.io/github/release/tx7do/go-wind-toolkit/protoc-gen-go-redact.svg)](https://github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact/releases)

_protoc-gen-redact (PGR)_ is a protoc plugin for automatically redacting field values in gRPC responses on the server side.

---

## Table of Contents

- [Attribution](#attribution)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Field-Level Redaction Rules](#field-level-redaction-rules)
  - [Scalar Fields](#scalar-fields)
  - [Message Fields](#message-fields)
  - [Repeated / Map Fields](#repeated--map-fields)
  - [Proto3 Optional Fields](#proto3-optional-fields)
  - [Oneof Fields](#oneof-fields)
  - [Regex Masking](#regex-masking)
  - [Position-Based Mask](#position-based-mask)
  - [Email Masking](#email-masking)
  - [Truncate](#truncate)
  - [Hash](#hash)
  - [UUID Replacement](#uuid-replacement)
  - [IP Address Masking](#ip-address-masking)
  - [URL Masking](#url-masking)
  - [Fixed-Length Mask](#fixed-length-mask)
  - [Custom Redactor](#custom-redactor)
  - [Conditional Redaction](#conditional-redaction)
- [File-Level AutoDetect](#file-level-autodetect)
- [Message-Level Options](#message-level-options)
- [Service and Method Options](#service-and-method-options)
- [Custom Templates](#custom-templates)
- [Buf Configuration](#buf-configuration)
- [Development and CI/CD](#development-and-cicd)
- [Contributing](#contributing)
- [License and Attribution](#license-and-attribution)

---

## Attribution

This project is a derivative work based on the original [protoc-gen-redact](https://github.com/arrakis-digital/protoc-gen-redact) by **Shivam Rathore** (Copyright 2020).

- **Original Author:** Shivam Rathore
- **Original Project:** https://github.com/arrakis-digital/protoc-gen-redact
- **Contributors:** John Castronuovo

This fork includes the following enhancements:
- Comprehensive error handling and validation system
- Extensive test suite (374+ tests)
- Oneof field support with type-safe switch statement generation
- Proto3 optional field support with correct pointer semantics
- Custom template file support
- Integration tests with actual protoc compilation
- 15 redaction rules (regex, mask, email, truncate, hash, UUID, IP, URL, fixed-length, custom, conditional, etc.)
- File-level auto-detection (match fields by name automatically)
- Conditional `.pb.redact.go` generation (only when redact annotations are used)

All modifications are licensed under the Apache License 2.0, consistent with the original project.

---

## Quick Start

Import the PGR extension and annotate messages or fields in your proto files:

```protobuf
syntax = "proto3";

package user;

import "redact/v1/redact.proto";
import "google/protobuf/empty.proto";

option go_package = "github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact/examples/user/pb;user";

message User {
    string username = 1;
    string password = 2 [(redact.value).string = "REDACTED"];
    string email    = 3 [(redact.value).email = { keep_local_first: 2 }];
    string name     = 4;
    Location home   = 5 [(redact.value).message.apply = true];

    message Location {
        double lat = 1 [(redact.value).double = 0.0];
        double lng = 2 [(redact.value).double = 0.0];
    }
}

service Chat {
    rpc GetUser(GetUserRequest) returns (User);
    rpc GetUserInternal(GetUserRequest) returns (User) {
        option (redact.method_skip) = true;
    }
    rpc ListUsers (google.protobuf.Empty) returns (ListUsersResponse) {
        option (redact.internal_method) = true;
    }
}
```

---

## Installation

```bash
go install github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact@latest
```

---

## Field-Level Redaction Rules

### Scalar Fields

Annotate individual fields with custom redaction values:

```protobuf
string password = 1 [(redact.value).string = "REDACTED"];
int32  age      = 2 [(redact.value).int32 = 0];
bool   active   = 3 [(redact.value).bool = false];
bytes  sign     = 4 [(redact.value).bytes = ""];
double score    = 5 [(redact.value).double = 0.0];
```

All proto scalar types are supported: `float`, `double`, `int32`, `int64`, `uint32`, `uint64`, `sint32`, `sint64`, `fixed32`, `fixed64`, `sfixed32`, `sfixed64`, `bool`, `string`, `bytes`, and `enum`.

### Message Fields

Control redaction of nested messages:

```protobuf
// Recursively apply redaction rules
Profile profile = 1 [(redact.value).message.apply = true];

// Set the entire message to nil
Settings settings = 2 [(redact.value).message.nil = true];

// Replace with an empty instance
Metadata metadata = 3 [(redact.value).message.empty = true];

// Skip redaction entirely
AuditLog log = 4 [(redact.value).message.skip = true];
```

### Repeated / Map Fields

```protobuf
// Clear the collection
map<string, string> attributes = 1 [(redact.value).element.empty = true];

// Apply default redaction to each element
repeated Address addresses = 2 [(redact.value).element.nested = true];

// Apply custom rules to each item
repeated int32 scores = 3 [(redact.value).element.item.int32 = 0];
repeated string phones = 4 [(redact.value).element.item.mask = { keep_first: 3 keep_last: 4 }];
```

### Proto3 Optional Fields

Proto3 `optional` fields use pointer semantics in Go. The generator handles this correctly:

```protobuf
message User {
    optional string email = 1 [(redact.value).string = "r*d@ct*d"];
    optional int32 age    = 2 [(redact.value).int32 = 0];
}
```

Generated code correctly uses pointer assignment:
```go
tmp := "r*d@ct*d"
x.Email = &tmp
```

### Oneof Fields

The generator supports `oneof` groups with type-safe switch statements:

```protobuf
message OneofMessage {
    oneof contact {
        string email = 1 [(redact.value).string = "r*d@ct*d"];
        string phone = 2 [(redact.value).mask = { keep_first: 3 keep_last: 4 }];
    }
}
```

Generated code:
```go
switch v := x.Contact.(type) {
case *OneofMessage_Email:
    v.Email = "r*d@ct*d"
case *OneofMessage_Phone:
    v.Phone = _redactMask(v.Phone, 3, 4, "*")
}
```

### Regex Masking

Use regular expressions for partial masking. Capture groups can be referenced via `${1}`, `${2}`:

```protobuf
string phone = 1 [(redact.value).regex = {
    pattern: "^(\\d{3})\\d{4}(\\d{4})$"
    replacement: "${1}****${2}"
}];
// 13812345678 → 138****5678
```

### Position-Based Mask

Keep the first N and last M characters, mask the rest:

```protobuf
string phone   = 1 [(redact.value).mask = { keep_first: 3 keep_last: 4 }];
// 13812345678 → 138****5678

string id_card = 2 [(redact.value).mask = { keep_first: 6 keep_last: 4 mask_char: "X" }];
// 110101199001011234 → 110101XXXXXXXX1234
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `keep_first` | Characters to keep at the start | 0 |
| `keep_last` | Characters to keep at the end | 0 |
| `mask_char` | Mask character | `"*"` |

### Email Masking

Split on `@`, mask the local part and/or domain separately:

```protobuf
string email  = 1 [(redact.value).email = { keep_local_first: 2 }];
// alice@example.com → al***@example.com

string email2 = 2 [(redact.value).email = { keep_local_first: 1 mask_domain: true }];
// bob@test.com → ***@********
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `keep_local_first` | Characters to keep at the start of local part | 0 |
| `mask_domain` | Whether to mask the domain | `false` |
| `mask_char` | Mask character | `"*"` |

### Truncate

Keep only the first N characters, optionally append a suffix:

```protobuf
string name = 1 [(redact.value).truncate = { length: 1 suffix: "**" }];
// Alexander → A**
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `length` | Characters to keep | — |
| `suffix` | Suffix appended after truncation | `"..."` |

### Hash

Replace the field value with its hex-encoded hash digest:

```protobuf
string token = 1 [(redact.value).hash = { algo: SHA256 }];

repeated string tokens = 2 [(redact.value).element.item.hash = { algo: MD5 }];
```

| Algorithm | Output Length |
|-----------|---------------|
| `MD5` | 32 chars |
| `SHA1` | 40 chars |
| `SHA256` | 64 chars |

### UUID Replacement

Replace the field value with a deterministic UUID v5 (SHA-1 based). Same input always produces the same UUID:

```protobuf
string user_id = 1 [(redact.value).uuid = {}];
// alice@example.com → a2b4c6d8-e9f0-5a1b-8c2d-3e4f5a6b7c8d
```

### IP Address Masking

Mask an IP address (IPv4 or IPv6), preserving the first N octets/hextets:

```protobuf
string client_ip = 1 [(redact.value).ip = { keep_octets: 2 }];
// 192.168.1.100 → 192.168.x.x
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `keep_octets` | Leading octets (IPv4) or hextets (IPv6) to preserve | 2 |
| `mask_char` | Mask character | `"x"` |

### URL Masking

Mask URL query parameter values:

```protobuf
string callback = 1 [(redact.value).url = { mask_query: true }];
// https://api.example.com/cb?token=secret → ...?token=******
```

### Fixed-Length Mask

Replace the entire value with a mask of the same length:

```protobuf
string bank_account = 1 [(redact.value).fixed_length = { char: "X" }];
// 6225880123456789 → XXXXXXXXXXXXXXXX
```

### Custom Redactor

Invoke a user-registered redactor function at runtime:

```go
import "github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact/redact/v1"

func init() {
    redact.RegisterCustomRedactor("myRedactor", func(s string) string {
        return "***" + s[len(s)-4:]
    })
}
```

```protobuf
string ssn = 1 [(redact.value).custom = { name: "myRedactor" }];
// 123456789 → ***6789
```

### Conditional Redaction

Apply inner rules only when an environment variable matches:

```protobuf
string phone = 1 [(redact.value).condition = {
    env_var: "APP_ENV"
    env_val: "production"
    rules: { mask: { keep_first: 3 keep_last: 4 } }
}];
// Only redacts when APP_ENV=production
```

---

## File-Level AutoDetect

Automatically apply redaction rules to fields matching name patterns:

```protobuf
option (redact.auto_detect) = {
    patterns: ["password", "token", "secret", "api_key"]
    default_action: { mask: { keep_first: 2 keep_last: 2 } }
};

message LoginRequest {
    string username = 1;  // No match — no redaction
    string password = 2;  // Matches "password" → auto-redacted
    string api_key  = 3;  // Matches "api_key" → auto-redacted
}
```

Matching is case-insensitive substring matching. Fields with explicit rules are not overridden.

---

## Message-Level Options

```protobuf
message PublicData {
    option (redact.ignored) = true;
    string data = 1;
}

message SensitiveData {
    option (redact.nil) = true;
    string secret = 1;
}

message EmptyData {
    option (redact.empty) = true;
    string field1 = 1;
}
```

---

## Service and Method Options

```protobuf
service MyService {
    rpc GetUser(GetUserRequest) returns (User);

    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse) {
        option (redact.method_skip) = true;
    }

    rpc AdminOperation(AdminRequest) returns (AdminResponse) {
        option (redact.internal_method) = true;
    }
}
```

Service-level options: `service_skip`, `internal_service`, `internal_service_code`, `internal_service_err_message`.

### What is an "internal service / internal method"?

`internal_service` and `internal_method` are access-gate options:

- `internal_service = true`: all RPC methods in the service require internal access.
- `internal_method = true`: only that RPC method requires internal access.

Generated `*.pb.redact.go` wrappers call `bypass.CheckInternal(ctx)` before invoking the real handler:

- `true`: allow request and call the actual service method.
- `false`: reject request immediately (default is `PermissionDenied`; customizable with `*_code` and `*_err_message`).

Notes:

- If you pass `nil` as bypass during registration, generated code falls back to `redact.Falsy` (always `false`), so internal RPCs are always blocked.
- If your `redact.Wrapper(...)` returns `true` for trusted requests (for example, requests carrying an internal gateway header), internal RPCs are allowed by design.
- These gates (and redaction wrappers) are effective only when registering with `RegisterRedacted<Service>Server(...)`.

Example:

```go
pb.RegisterRedactedMyServiceServer(s, impl,
    redact.Wrapper(func(ctx context.Context) bool {
        md, ok := metadata.FromIncomingContext(ctx)
        return ok && len(md["x-internal"]) > 0
    }),
)
```

---

## Custom Templates

```bash
protoc \
  --plugin=protoc-gen-go-redact=/path/to/protoc-gen-go-redact \
  --go-redact_out=. \
  --go-redact_opt=template_file=/path/to/your/template.tmpl \
  your_proto_file.proto
```

See [examples/CUSTOM_TEMPLATE.md](examples/CUSTOM_TEMPLATE.md) for full documentation.

---

## Buf Configuration

This project uses [Buf](https://buf.build/) to manage Protobuf files, run lint checks, detect breaking changes, and generate code. The project root contains the following buf configuration files:

### buf.yaml - Module Configuration

Defines the Buf module, lint rules, and breaking change policy:

```yaml
version: v2

modules:
  - path: .
    lint:
      use:
        - STANDARD
    breaking:
      use:
        - FILE

deps:
  - "buf.build/go-wind/redact"

breaking:
  use:
    - FILE

lint:
  use:
    - DEFAULT
```

| Field | Description |
|------|-------------|
| `name` | Unique identifier of the Buf module, used when pushing to the Buf Schema Registry (BSR) |
| `breaking.use` | Breaking change check level; `FILE` checks API compatibility at the file level |
| `lint.use` | Lint rule set; `STANDARD` is the recommended default from Buf |

### buf.gen.yaml - Code Generation Configuration

Defines the code generation plugin configuration. The current config generates Go protobuf and gRPC code:

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

To also generate redaction code via Buf, add the `go-redact` plugin under `plugins` (requires `protoc-gen-go-redact` to be installed first):

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

  # Generate redaction code (requires protoc-gen-go-redact to be installed)
  - plugin: go-redact
    out: .
    opt:
      - paths=source_relative
```

Once configured, run `buf generate` to generate protobuf, gRPC, and redaction code in one step.

### .bufignore - Ignore File

Specifies directories Buf should ignore, avoiding lint checks on examples and test data:

```
# Test data - examples and tests can have non-standard formatting
testdata
examples
```

### buf.lock - Dependency Lock

Auto-generated by Buf. Do not edit manually.

### Common Buf Commands

This project wraps common Buf commands via the Makefile:

```bash
make buf-lint                      # Lint proto files
make buf-format                    # Format proto files
make buf-breaking                  # Check for breaking changes (against main branch)
make buf-generate                  # Generate code
make buf-push                      # Push to the Buf Schema Registry
make buf-push-tag TAG=v1.0.0       # Push to BSR with a tag
```

You can also use the Buf commands directly:

```bash
buf lint                           # Lint proto files
buf format -w                      # Format proto files (-w writes to files)
buf breaking --against '.git#branch=main'  # Check for breaking changes
buf generate                       # Generate code
buf push                           # Push to BSR
```

---

## Development and CI/CD

```bash
make help           # List all targets
make fmt            # Format code
make lint           # Run linters
make test           # Run all tests
make build          # Build the plugin
make pre-commit     # fmt + lint + test-short
make ci-full        # Full CI pipeline
```

---

## Contributing

Contributions are welcome! Please open a PR with your changes. Ensure all tests pass before submitting.

---

## License and Attribution

Licensed under the [Apache License 2.0](./LICENSE).

- Copyright 2020 Shivam Rathore (Original Work)
- Copyright 2025 Contributors (Modifications)

See [NOTICE](./NOTICE) for detailed attribution.
