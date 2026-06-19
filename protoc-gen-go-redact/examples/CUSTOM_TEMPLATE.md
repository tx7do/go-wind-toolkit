# Using Custom Templates with protoc-gen-redact

This guide explains how to use custom code generation templates with protoc-gen-redact.

## Overview

By default, protoc-gen-redact uses an embedded template to generate redaction code. However, you can override this behavior by providing your own custom template file using the `template_file` parameter.

## Usage

To use a custom template, pass the `template_file` parameter to protoc via `--redact_opt`:

```bash
protoc \
  --plugin=protoc-gen-redact=/path/to/protoc-gen-redact \
  --redact_out=. \
  --redact_opt=template_file=/path/to/your/template.tmpl \
  your_proto_file.proto
```

### Example with Relative Path

```bash
protoc \
  --plugin=protoc-gen-redact=./protoc-gen-redact \
  --redact_out=. \
  --redact_opt=template_file=./examples/custom-template.tmpl \
  examples/user/pb/user.proto
```

### Example with Absolute Path

```bash
protoc \
  --plugin=protoc-gen-redact=./protoc-gen-redact \
  --redact_out=. \
  --redact_opt=template_file=/home/user/project/my-template.tmpl \
  examples/user/pb/user.proto
```

### Combining Multiple Options

You can combine `template_file` with other options:

```bash
protoc \
  --plugin=protoc-gen-redact=./protoc-gen-redact \
  --redact_out=. \
  --redact_opt=template_file=./my-template.tmpl,paths=source_relative \
  your_proto_file.proto
```

## Template Structure

The template receives a `ProtoFileData` object with the following structure:

```go
type ProtoFileData struct {
    Source     string              // Source proto file name
    Package    string              // Go package name
    Imports    map[string]string   // Import aliases -> import paths
    References []string            // Import references to suppress unused warnings
    Services   []*ServiceData      // gRPC services
    Messages   []*MessageData      // Proto messages
}

type ServiceData struct {
    Name    string          // Service name
    Skip    bool            // Whether to skip redaction for this service
    Methods []*MethodData   // Service methods
}

type MethodData struct {
    Name            string        // Method name
    Skip            bool          // Skip redaction for this method
    Input           string        // Input message type name
    Output          *MessageData  // Output message with redaction options
    Internal        bool          // Whether this is an internal method
    StatusCode      string        // gRPC status code for internal methods
    ErrMessage      string        // Error message for internal methods
    ClientStreaming bool          // Client streaming RPC
    ServerStreaming bool          // Server streaming RPC
}

type MessageData struct {
    Name      string        // Message name
    WithAlias string        // Message name with import alias
    Fields    []*FieldData  // Message fields
    Ignore    bool          // Ignore all redaction for this message
    ToNil     bool          // Set message to nil
    ToEmpty   bool          // Set message to empty struct
}

type FieldData struct {
    Name           string  // Field name
    Redact         bool    // Whether to redact this field
    RedactionValue string  // Value to use for redaction
    FieldGoType    string  // Go type (int32, string, bool, etc.)
    IsMap          bool    // Is a map field
    IsRepeated     bool    // Is a repeated field
    IsMessage      bool    // Is a message field
    IsOptional     bool    // Is an optional field (proto3 pointer)
    Iterate        bool    // Iterate over elements (for repeated/map)
    NestedEmbedCall bool   // Call nested message redaction
    EmbedSkip      bool    // Skip embedded message redaction
    EmbedMessageName          string  // Embedded message name
    EmbedMessageNameWithAlias string  // Embedded message name with alias
}
```

## Template Functions

The template has access to the following functions:

- `package` - Returns the Go package name for an entity
- `name` - Returns the Go name for an entity
- `eq` - Equality comparison (built-in Go template function)

## Example Template

See [`custom-template.tmpl`](./custom-template.tmpl) for a complete example that matches the default embedded template. You can use this as a starting point for your customizations.

## Template Validation

The template file must:
- Exist and be readable
- Be a regular file (not a directory or special file)
- Be under 10MB in size
- Contain valid Go template syntax

If the template fails validation or parsing, protoc-gen-redact will report a detailed error message.

## Use Cases

Custom templates are useful for:

1. **Adding Custom Comments or Documentation**
   - Add your company's copyright header
   - Include generated code warnings in different formats
   - Add links to internal documentation

2. **Modifying Redaction Behavior**
   - Add logging when redaction occurs
   - Implement custom redaction strategies
   - Add metrics or telemetry

3. **Changing Code Style**
   - Adjust formatting to match your style guide
   - Use different naming conventions
   - Add custom error handling

4. **Integration with Other Tools**
   - Add compatibility shims for other frameworks
   - Generate additional helper functions
   - Add instrumentation or tracing

## Example Customization

Here's a simple example of adding a custom header to the generated code:

```go
{{ $data := . }}
// Code generated by protoc-gen-redact. DO NOT EDIT.
// source: {{ $data.Source }}
//
// Copyright (c) 2025 My Company
// This file is auto-generated. Do not modify manually.
//
// For questions, contact: devops@mycompany.com

package {{ $data.Package }}

// ... rest of template ...
```

## Troubleshooting

### Template Not Found

If you get "file does not exist" error:
- Verify the path is correct (use absolute path to test)
- Check file permissions
- Ensure the file exists before running protoc

### Template Parse Error

If you get template parsing errors:
- Check Go template syntax
- Ensure all `{{ }}` blocks are balanced
- Verify field names match the data structures

### Generated Code Doesn't Compile

If the generated code has compilation errors:
- Verify the template generates valid Go code
- Check that field names and types are correct
- Test with a simple proto file first

## Getting Help

For issues with custom templates:
1. Check that the default template works (omit `template_file`)
2. Start with the example template and make small changes
3. Report bugs at https://github.com/menta2k/protoc-gen-redact/issues
