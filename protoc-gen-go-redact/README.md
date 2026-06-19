protoc-gen-redact (PGR)
=======================
[![Build and Publish](https://github.com/menta2k/protoc-gen-redact/workflows/Build%20and%20Publish/badge.svg)](https://github.com/menta2k/protoc-gen-redact/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/menta2k/protoc-gen-redact/v3?dropcache)](https://goreportcard.com/report/github.com/menta2k/protoc-gen-redact/v3)
[![Go Reference](https://pkg.go.dev/badge/github.com/menta2k/protoc-gen-redact/v3.svg)](https://pkg.go.dev/github.com/menta2k/protoc-gen-redact/v3)
[![License](https://img.shields.io/badge/license-apache2-mildgreen.svg)](./LICENSE)
[![GitHub release](https://img.shields.io/github/release/menta2k/protoc-gen-redact.svg)](https://github.com/menta2k/protoc-gen-redact/releases)

_protoc-gen-redact (PGR)_ is a protoc plugin to redact field values in GRPC client calls from the server. This plugin
adds support to protoc-generated code to redact certain fields in the GRPC calls.

## Attribution

This project is a derivative work based on the original [protoc-gen-redact](https://github.com/arrakis-digital/protoc-gen-redact) by **Shivam Rathore** (Copyright 2020).

**Original Author:** Shivam Rathore
**Original Project:** https://github.com/arrakis-digital/protoc-gen-redact
**Contributors:** John Castronuovo

This fork includes enhancements and modifications including:
- Comprehensive error handling and validation
- Extensive test suite (374+ tests)
- Oneof field support with type-safe switch statement generation
- Support for proto3 optional fields with correct pointer semantics
- Custom template file support for code generation
- Integration tests with actual protoc compilation
- Improved documentation and examples

All modifications are licensed under the Apache License 2.0, consistent with the original project.

Developers only need to import the PGR extension and annotate the messages or fields in their proto files to redact:

```protobuf
syntax = "proto3";

package user;

import "redact/v3/redact.proto";
import "google/protobuf/empty.proto";

option go_package = "github.com/menta2k/protoc-gen-redact/v3/examples/user/pb;user";

message User {
    string username = 1;
    string password = 2 [(redact.v3.value).string = "REDACTED"];
    string email = 3 [(redact.v3.value).string = "r*d@ct*d"];
    string name = 4;
    Location home = 5 [(redact.v3.value).message.apply = true];

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

## Field-Level Redaction

### Scalar Fields

Annotate individual fields with custom redaction values:

```protobuf
string password = 1 [(redact.v3.value).string = "REDACTED"];
int32 age = 2 [(redact.v3.value).int32 = 0];
bool is_active = 3 [(redact.v3.value).bool = false];
bytes signature = 4 [(redact.v3.value).bytes = ""];
double score = 5 [(redact.v3.value).double = 0.0];
```

All proto scalar types are supported: `float`, `double`, `int32`, `int64`, `uint32`, `uint64`, `sint32`, `sint64`, `fixed32`, `fixed64`, `sfixed32`, `sfixed64`, `bool`, `string`, `bytes`, and `enum`.

### Message Fields

Control redaction of nested messages:

```protobuf
// Recursively apply redaction rules to nested message fields
Profile profile = 1 [(redact.v3.value).message.apply = true];

// Set the entire message to nil
Settings settings = 2 [(redact.v3.value).message.nil = true];

// Replace with an empty instance
Metadata metadata = 3 [(redact.v3.value).message.empty = true];

// Skip redaction entirely for this field
AuditLog log = 4 [(redact.v3.value).message.skip = true];
```

### Repeated and Map Fields

```protobuf
// Clear the collection to empty
map<string, string> attributes = 1 [(redact.v3.value).element.empty = true];

// Apply default redaction to each element
repeated Address addresses = 2 [(redact.v3.value).element.nested = true];

// Apply custom redaction to each item
repeated int32 scores = 3 [(redact.v3.value).element.item.int32 = 0];
```

## Proto3 Optional Fields

Proto3 `optional` fields use pointer semantics in Go. The generator handles this correctly by creating temporary variables and assigning pointers:

```protobuf
message User {
    optional string email = 1 [(redact.v3.value).string = "r*d@ct*d"];
    optional int32 age = 2 [(redact.v3.value).int32 = 0];
    optional bool is_active = 3 [(redact.v3.value).bool = false];
    optional bytes signature = 4 [(redact.v3.value).bytes = ""];
    optional Profile profile = 5 [(redact.v3.value).message.nil = true];
}
```

Generated code correctly uses pointer assignment:
```go
tmp := "r*d@ct*d"
x.Email = &tmp
```

## Oneof Fields

The generator supports `oneof` groups with type-safe switch statements. Each oneof variant is matched by its Go wrapper type, and only fields with redaction annotations generate case branches:

```protobuf
message OneofMessage {
    string id = 1;

    // All fields redacted
    oneof contact {
        string email = 2 [(redact.v3.value).string = "r*d@ct*d"];
        string phone = 3 [(redact.v3.value).string = "XXX-XXX-XXXX"];
        int32 phone_code = 4 [(redact.v3.value).int32 = 0];
    }

    // Message fields in oneofs
    oneof payload {
        Profile user_profile = 5 [(redact.v3.value).message.apply = true];
        Settings user_settings = 6 [(redact.v3.value).message.nil = true];
        string raw_data = 7 [(redact.v3.value).string = "REDACTED"];
    }

    // Mixed: some fields redacted, some not
    oneof identifier {
        string username = 8 [(redact.v3.value).string = "REDACTED"];
        string public_id = 9;  // not redacted
        int64 internal_id = 10 [(redact.v3.value).int64 = 0];
    }
}
```

Generated code for a oneof group:
```go
// Redacting oneof: Contact
switch v := x.Contact.(type) {
case *OneofMessage_Email:
    v.Email = "r*d@ct*d"
case *OneofMessage_Phone:
    v.Phone = "XXX-XXX-XXXX"
case *OneofMessage_PhoneCode:
    v.PhoneCode = 0
}
```

If a oneof group has no redacted fields, the switch statement is omitted entirely to avoid compilation errors.

## Message-Level Options

Control redaction behavior for entire messages:

```protobuf
// Skip all redaction for this message
message PublicData {
    option (redact.v3.ignored) = true;
    string data = 1;
}

// Always set to nil
message SensitiveData {
    option (redact.v3.nil) = true;
    string secret = 1;
}

// Always replace with empty instance
message EmptyData {
    option (redact.v3.empty) = true;
    string field1 = 1;
}
```

## Service and Method Options

Control redaction at the service and method level:

```protobuf
service MyService {
    // Normal RPC with response redaction
    rpc GetUser(GetUserRequest) returns (User);

    // Skip redaction for this method
    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse) {
        option (redact.v3.method_skip) = true;
    }

    // Mark method as internal (returns PermissionDenied)
    rpc AdminOperation(AdminRequest) returns (AdminResponse) {
        option (redact.v3.internal_method) = true;
    }
}
```

## Advanced Features

### Custom Code Generation Templates

protoc-gen-redact supports using custom templates for code generation, allowing you to modify the generated code to match your specific requirements.

#### Using a Custom Template

To use a custom template, pass the `template_file` parameter to protoc via `--redact_opt`:

```bash
protoc \
  --plugin=protoc-gen-redact=/path/to/protoc-gen-redact \
  --redact_out=. \
  --redact_opt=template_file=/path/to/your/template.tmpl \
  your_proto_file.proto
```

#### Example Template

An example template is provided in `examples/custom-template.tmpl`. You can use this as a starting point for your customizations:

```bash
protoc \
  --plugin=protoc-gen-redact=./protoc-gen-redact \
  --redact_out=. \
  --redact_opt=template_file=./examples/custom-template.tmpl \
  examples/user/pb/user.proto
```

#### Documentation

For complete documentation on custom templates including:
- Template structure and data types
- Available template functions
- Use cases and examples
- Troubleshooting guide

See [examples/CUSTOM_TEMPLATE.md](examples/CUSTOM_TEMPLATE.md)

## Development and CI/CD

This project includes a comprehensive build system and CI/CD pipeline:

### Build System (Makefile)

The project uses Make for build automation with 50+ targets organized into categories:

```bash
# See all available targets
make help

# Development workflow
make fmt              # Format code
make lint             # Run all linters
make test             # Run all tests
make test-short       # Quick tests during development
make build            # Build the plugin

# Before committing
make pre-commit       # Run fmt + lint + test-short

# Full CI pipeline
make ci-full          # Full CI with coverage and buf checks
```



Request for Contribution
------------------------
Contributors are more than welcome and much appreciated. Please feel free to open a PR to improve anything you don't
like, or would like to add.

Please make your changes in a specific branch and create a pull request into master! If you can, please make sure all
the changes work properly and does not affect the existing functioning.

No PR is too small! Even the smallest effort is countable.

License and Attribution
-----------------------

This project is licensed under the [Apache License 2.0](./LICENSE).

Copyright 2020 Shivam Rathore (Original Work)
Copyright 2025 Contributors (Modifications)

This is a derivative work based on the original protoc-gen-redact project. All attribution notices, copyright statements, and license terms from the original work have been retained in accordance with the Apache License 2.0.

See the [NOTICE](./NOTICE) file for detailed attribution and a list of modifications

