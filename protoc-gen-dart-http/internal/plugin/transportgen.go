package plugin

import (
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http/internal/codegen"
	"github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http/internal/httprule"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// generateTransportSharedFile writes the shared transport.dart file that is
// emitted once at the root of the output directory and imported by all
// generated package index files.
func generateTransportSharedFile(f *codegen.File) {
	f.P("// Shared transport infrastructure for protoc-gen-dart-http.")
	f.P("// Auto-generated. DO NOT EDIT.")
	f.P()
	f.P("import 'dart:async';")
	f.P()

	// --- TransportMeta ---
	f.P("/// Metadata for an RPC call, passed to the transport for routing and diagnostics.")
	f.P("class TransportMeta {")
	f.P(t(1), "final String service;")
	f.P(t(1), "final String method;")
	f.P(t(1), "const TransportMeta({required this.service, required this.method});")
	f.P()
	f.P(t(1), "@override")
	f.P(t(1), "String toString() => 'TransportMeta(service: $service, method: $method)';")
	f.P()
	f.P(t(1), "@override")
	f.P(t(1), "bool operator ==(Object other) =>")
	f.P(t(2), "identical(this, other) ||")
	f.P(t(2), "other is TransportMeta &&")
	f.P(t(3), "runtimeType == other.runtimeType &&")
	f.P(t(3), "service == other.service &&")
	f.P(t(3), "method == other.method;")
	f.P()
	f.P(t(1), "@override")
	f.P(t(1), "int get hashCode => Object.hash(service, method);")
	f.P("}")
	f.P()

	// --- ClientTransport ---
	f.P("/// Abstract transport interface for making HTTP requests.")
	f.P("///")
	f.P("/// Implement this with your preferred HTTP client (package:http, dio, etc.).")
	f.P("abstract class ClientTransport {")
	f.P(t(1), "/// Performs a unary (request/response) RPC.")
	f.P(t(1), "Future<dynamic> unary(")
	f.P(t(2), "String path,")
	f.P(t(2), "String method,")
	f.P(t(2), "String? body,")
	f.P(t(2), "TransportMeta meta, {")
	f.P(t(2), "Map<String, String>? headers,")
	f.P(t(1), "});")
	f.P()
	f.P(t(1), "/// Opens a server-streaming connection (e.g. SSE).")
	f.P(t(1), "/// Returns a stream of JSON-decoded event payloads.")
	f.P(t(1), "Stream<Map<String, dynamic>> serverStream(")
	f.P(t(2), "String path,")
	f.P(t(2), "TransportMeta meta, {")
	f.P(t(2), "Map<String, String>? headers,")
	f.P(t(1), "});")
	f.P()
	f.P(t(1), "/// Opens a bidirectional streaming connection (e.g. WebSocket).")
	f.P(t(1), "DuplexConnection duplexStream(")
	f.P(t(2), "String path,")
	f.P(t(2), "TransportMeta meta, {")
	f.P(t(2), "Map<String, String>? headers,")
	f.P(t(1), "});")
	f.P("}")
	f.P()

	// --- DuplexConnection ---
	f.P("/// Abstract bidirectional connection for duplex streaming.")
	f.P("abstract class DuplexConnection {")
	f.P(t(1), "/// Stream of incoming JSON messages from the server.")
	f.P(t(1), "Stream<Map<String, dynamic>> get incoming;")
	f.P(t(1), "/// Sends a JSON message to the server.")
	f.P(t(1), "void send(Map<String, dynamic> data);")
	f.P(t(1), "/// Closes the connection and releases resources.")
	f.P(t(1), "Future<void> close();")
	f.P("}")
	f.P()

	// --- TypedDuplexConnection ---
	f.P("/// Type-safe wrapper around [DuplexConnection] that handles JSON (de)serialization.")
	f.P("class TypedDuplexConnection<TIn, TOut> {")
	f.P(t(1), "final DuplexConnection _conn;")
	f.P(t(1), "final TOut Function(Map<String, dynamic>) _fromJson;")
	f.P(t(1), "final Map<String, dynamic> Function(TIn) _toJson;")
	f.P()
	f.P(t(1), "TypedDuplexConnection(this._conn, this._fromJson, this._toJson);")
	f.P()
	f.P(t(1), "/// Typed stream of incoming messages.")
	f.P(t(1), "Stream<TOut> get stream => _conn.incoming.map(_fromJson);")
	f.P()
	f.P(t(1), "/// Sends a typed message, serialized to JSON.")
	f.P(t(1), "void send(TIn data) => _conn.send(_toJson(data));")
	f.P()
	f.P(t(1), "/// Closes the connection and releases resources.")
	f.P(t(1), "Future<void> close() => _conn.close();")
	f.P("}")
}

// transportImportPath returns the relative import path from a package's
// index.dart to the shared transport.dart at the output root.
func transportImportPath(pkg protoreflect.FullName) string {
	segments := strings.Split(string(pkg), ".")
	parts := make([]string, len(segments))
	for i := range segments {
		parts[i] = ".."
	}
	return strings.Join(parts, "/") + "/transport.dart"
}

// --- Streaming method generators ---

func generateStreamClientMethod(
	f *codegen.File,
	pkg protoreflect.FullName,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	if method.IsStreamingClient() {
		generateBidiStreamMethod(f, pkg, method, rule)
	} else {
		generateServerStreamMethod(f, pkg, method, rule)
	}
}

func generateServerStreamMethod(
	f *codegen.File,
	pkg protoreflect.FullName,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	output := typeFromMessage(pkg, method.Output())
	dartMethodName := lowerCamel(string(method.Name()))

	paramName := "request"
	if !methodUsesRequest(rule, method.Input()) {
		paramName = "_request"
	}

	commentGenerator{descriptor: method}.generateLeading(f, 1)
	f.P(t(1), "Stream<", output.Reference(), "> ", dartMethodName, "(", typeFromMessage(pkg, method.Input()).Reference(), " ", paramName, ", {Map<String, String>? headers}) {")
	generateMethodPathValidation(f, method.Input(), rule)
	generateMethodPath(f, method.Input(), rule)
	hasQP := generateMethodQuery(f, method.Input(), rule)
	uriVar := "path"
	if hasQP {
		f.P(t(2), "var uri = path;")
		f.P(t(2), "if (queryParams.isNotEmpty) {")
		f.P(t(3), "uri += '?${queryParams.join(\"&\")}';")
		f.P(t(2), "}")
		uriVar = "uri"
	}
	f.P(t(2), "return _transport.serverStream(", uriVar, ", TransportMeta(")
	f.P(t(3), "service: ", dartString(string(method.Parent().Name())), ",")
	f.P(t(3), "method: ", dartString(string(method.Name())), ",")
	if _, isWKT := WellKnownType(method.Output()); !isWKT {
		f.P(t(2), "), headers: headers)")
		f.P(t(3), ".map((json) => ", output.Name, ".fromJson(json));")
	} else {
		f.P(t(2), "), headers: headers);")
	}
	f.P(t(1), "}")
}

func generateBidiStreamMethod(
	f *codegen.File,
	pkg protoreflect.FullName,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	if hasPathVariables(rule) {
		generateBidiStreamWithParams(f, pkg, method, rule)
	} else {
		generateBidiStreamLiteral(f, pkg, method, rule)
	}
}

func generateBidiStreamLiteral(
	f *codegen.File,
	pkg protoreflect.FullName,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	inputType := typeFromMessage(pkg, method.Input())
	outputType := typeFromMessage(pkg, method.Output())
	dartMethodName := lowerCamel(string(method.Name()))
	path := literalPath(rule)

	commentGenerator{descriptor: method}.generateLeading(f, 1)
	f.P(t(1), "TypedDuplexConnection<", inputType.Reference(), ", ", outputType.Reference(), "> ", dartMethodName, "({Map<String, String>? headers}) {")
	f.P(t(2), "final path = ", path, ";")
	f.P(t(2), "return TypedDuplexConnection<", inputType.Reference(), ", ", outputType.Reference(), ">( ")
	f.P(t(3), "_transport.duplexStream(path, TransportMeta(")
	f.P(t(4), "service: ", dartString(string(method.Parent().Name())), ",")
	f.P(t(4), "method: ", dartString(string(method.Name())), ",")
	f.P(t(3), "), headers: headers),")
	if _, isWKT := WellKnownType(method.Output()); isWKT {
		f.P(t(3), "(json) => json,")
	} else {
		f.P(t(3), "(json) => ", outputType.Name, ".fromJson(json),")
	}
	if _, isWKT := WellKnownType(method.Input()); isWKT {
		f.P(t(3), "(data) => {},")
	} else {
		f.P(t(3), "(data) => data.toJson(),")
	}
	f.P(t(2), ");")
	f.P(t(1), "}")
}

func generateBidiStreamWithParams(
	f *codegen.File,
	pkg protoreflect.FullName,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	inputType := typeFromMessage(pkg, method.Input())
	outputType := typeFromMessage(pkg, method.Output())
	dartMethodName := lowerCamel(string(method.Name()))

	commentGenerator{descriptor: method}.generateLeading(f, 1)
	f.P(t(1), "TypedDuplexConnection<", inputType.Reference(), ", ", outputType.Reference(), "> ", dartMethodName, "(", inputType.Reference(), " request, {Map<String, String>? headers}) {")
	generateMethodPathValidation(f, method.Input(), rule)
	generateMethodPath(f, method.Input(), rule)
	f.P(t(2), "return TypedDuplexConnection<", inputType.Reference(), ", ", outputType.Reference(), ">( ")
	f.P(t(3), "_transport.duplexStream(path, TransportMeta(")
	f.P(t(4), "service: ", dartString(string(method.Parent().Name())), ",")
	f.P(t(4), "method: ", dartString(string(method.Name())), ",")
	f.P(t(3), "), headers: headers),")
	if _, isWKT := WellKnownType(method.Output()); isWKT {
		f.P(t(3), "(json) => json,")
	} else {
		f.P(t(3), "(json) => ", outputType.Name, ".fromJson(json),")
	}
	if _, isWKT := WellKnownType(method.Input()); isWKT {
		f.P(t(3), "(data) => {},")
	} else {
		f.P(t(3), "(data) => data.toJson(),")
	}
	f.P(t(2), ");")
	f.P(t(1), "}")
}

func literalPath(rule httprule.Rule) string {
	parts := make([]string, 0, len(rule.Template.Segments))
	for _, seg := range rule.Template.Segments {
		switch seg.Kind {
		case httprule.SegmentKindLiteral:
			parts = append(parts, seg.Literal)
		case httprule.SegmentKindMatchSingle:
			parts = append(parts, "*")
		case httprule.SegmentKindMatchMultiple:
			parts = append(parts, "**")
		}
	}
	path := strings.Join(parts, "/")
	if rule.Template.Verb != "" {
		path += ":" + rule.Template.Verb
	}
	return dartString(path)
}

func hasPathVariables(rule httprule.Rule) bool {
	for _, seg := range rule.Template.Segments {
		if seg.Kind == httprule.SegmentKindVariable {
			return true
		}
	}
	return false
}

func isStreamingMethod(method protoreflect.MethodDescriptor) bool {
	return method.IsStreamingClient() || method.IsStreamingServer()
}

// getDefaultHost reads the google.api.default_host extension from a service descriptor.
func getDefaultHost(service protoreflect.ServiceDescriptor) string {
	if service.Options() == nil {
		return ""
	}
	ext := proto.GetExtension(service.Options(), annotations.E_DefaultHost)
	if host, ok := ext.(string); ok && host != "" {
		return host
	}
	return ""
}
