package plugin

import (
	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/codegen"
	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/httprule"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func generateTransportInfra(f *codegen.File, defaultHost string) {
	f.P("interface TransportMeta {")
	f.P(t(1), "service: string;")
	f.P(t(1), "method: string;")
	f.P("}")
	f.P()
	f.P("export interface ClientTransport {")
	f.P(t(1), "unary(")
	f.P(t(2), "path: string,")
	f.P(t(2), "method: string,")
	f.P(t(2), "body: null | string,")
	f.P(t(2), "meta: TransportMeta,")
	f.P(t(1), "): Promise<unknown>;")
	f.P(t(1), "serverStream<T>(path: string, meta: TransportMeta): ServerStream<T>;")
	f.P(t(1), "// Raw byte chunks for google.api.HttpBody server streams: the first")
	f.P(t(1), "// chunk's content type rides the HTTP response Content-Type header,")
	f.P(t(1), "// every chunk is raw payload bytes (no JSON framing).")
	f.P(t(1), "byteStream(path: string, meta: TransportMeta): ServerStream<Uint8Array>;")
	f.P(t(1), "duplexStream<TIn, TOut>(path: string, meta: TransportMeta): DuplexStream<TIn, TOut>;")
	f.P("}")
	f.P()
	f.P("export interface ServerStream<T> {")
	f.P(t(1), "onEvent(listener: (data: T) => void): () => void;")
	f.P(t(1), "onError(handler: (error: Error) => void): void;")
	f.P(t(1), "close(): void;")
	f.P("}")
	f.P()
	f.P("export interface DuplexStream<TIn, TOut> extends ServerStream<TOut> {")
	f.P(t(1), "send(data: TIn): void;")
	f.P("}")
	f.P()
	if defaultHost != "" {
		f.P("export const DEFAULT_HOST = ", tsSingleQuote(defaultHost), ";")
		f.P()
	}
}

func generateStreamInterfaceMethod(f *codegen.File, pkg protoreflect.FullName, method protoreflect.MethodDescriptor, rule httprule.Rule) {
	commentGenerator{descriptor: method}.generateLeading(f, 1)
	input := typeFromMessage(pkg, method.Input())
	if method.IsStreamingClient() {
		output := typeFromMessage(pkg, method.Output())
		if rule.Template.HasVariables() {
			f.P(t(1), method.Name(), "(")
			f.P(t(2), "request: ", input.Reference(), ",")
			f.P(t(1), "): DuplexStream<", input.Reference(), ", ", output.Reference(), ">;")
		} else {
			f.P(t(1), method.Name(), "(): DuplexStream<", input.Reference(), ", ", output.Reference(), ">;")
		}
	} else if isHttpBodyStream(method) {
		// HttpBody 下载流:客户端拿到原始字节块,不生成消息接口。
		f.P(t(1), method.Name(), "(")
		f.P(t(2), "request: ", input.Reference(), ",")
		f.P(t(1), "): ServerStream<Uint8Array>;")
	} else {
		output := typeFromMessage(pkg, method.Output())
		f.P(t(1), method.Name(), "(")
		f.P(t(2), "request: ", input.Reference(), ",")
		f.P(t(1), "): ServerStream<", output.Reference(), ">;")
	}
}

func generateStreamClientMethod(
	f *codegen.File,
	pkg protoreflect.FullName,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	if method.IsStreamingClient() {
		generateBidiStreamMethod(f, method, rule)
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
	isBytes := isHttpBodyStream(method)
	var output Type
	if !isBytes {
		output = typeFromMessage(pkg, method.Output())
	}
	paramName := "request"
	if !httprule.MethodUsesRequest(rule, method.Input(), IsWellKnownType) {
		paramName = "_request"
	}
	f.P(t(2), method.Name(), "(", paramName, ") {")
	generateMethodPathValidation(f, method.Input(), rule)
	generateMethodPath(f, method.Input(), rule)
	hasQP := generateMethodQuery(f, method.Input(), rule)
	uriVar := "path"
	if hasQP {
		f.P(t(3), "let uri = path;")
		f.P(t(3), "if (queryParams.length > 0) {")
		f.P(t(4), "uri += `?${queryParams.join('&')}`;")
		f.P(t(3), "}")
		uriVar = "uri"
	}
	if isBytes {
		f.P(t(3), "return transport.byteStream(", uriVar, ", {")
		f.P(t(4), "service: '", method.Parent().Name(), "',")
		f.P(t(4), "method: '", method.Name(), "',")
		f.P(t(3), "});")
		f.P(t(2), "},")
		return
	}
	f.P(t(3), "return transport.serverStream<", output.Reference(), ">(", uriVar, ", {")
	f.P(t(4), "service: '", method.Parent().Name(), "',")
	f.P(t(4), "method: '", method.Name(), "',")
	f.P(t(3), "});")
	f.P(t(2), "},")
}

func generateBidiStreamMethod(
	f *codegen.File,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	if rule.Template.HasVariables() {
		generateBidiStreamWithParams(f, method, rule)
	} else {
		generateBidiStreamLiteral(f, method, rule)
	}
}

func generateBidiStreamLiteral(
	f *codegen.File,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	path := rule.Template.LiteralPath(tsSingleQuote)
	f.P(t(2), method.Name(), "() {")
	f.P(t(3), "const path = ", path, ";")
	f.P(t(3), "return transport.duplexStream(path, {")
	f.P(t(4), "service: '", method.Parent().Name(), "',")
	f.P(t(4), "method: '", method.Name(), "',")
	f.P(t(3), "});")
	f.P(t(2), "},")
}

func generateBidiStreamWithParams(
	f *codegen.File,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	f.P(t(2), method.Name(), "(request) {")
	generateMethodPathValidation(f, method.Input(), rule)
	generateMethodPath(f, method.Input(), rule)
	f.P(t(3), "return transport.duplexStream(path, {")
	f.P(t(4), "service: '", method.Parent().Name(), "',")
	f.P(t(4), "method: '", method.Name(), "',")
	f.P(t(3), "});")
	f.P(t(2), "},")
}
