package plugin

import (
	"strconv"
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-typescript-http/internal/codegen"
	"github.com/tx7do/go-wind-toolkit/protoc-gen-typescript-http/internal/httprule"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func generateTransportInfra(f *codegen.File, defaultHost string) {
	f.P("interface TransportMeta { service: string; method: string; }")
	f.P()
	f.P("export interface ClientTransport {")
	f.P(t(1), "unary(path: string, method: string, body: string | null, meta: TransportMeta): Promise<unknown>;")
	f.P(t(1), "serverStream<T>(path: string, meta: TransportMeta): ServerStream<T>;")
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
		f.P("const DEFAULT_HOST = ", strconv.Quote(defaultHost), ";")
		f.P()
	}
	f.P("export interface TransportOptions {")
	f.P(t(1), "baseUrl?: string;")
	f.P(t(1), "headers?: Record<string, string>;")
	f.P(t(1), "request?: typeof fetch;")
	f.P("}")
	f.P()
	// SSETransport
	f.P("export class SSETransport<T> implements ServerStream<T> {")
	f.P(t(1), "private eventSource: EventSource;")
	f.P(t(1), "private listeners: Array<(data: T) => void> = [];")
	f.P(t(1), "private errorHandlers: Array<(error: Error) => void> = [];")
	f.P()
	f.P(t(1), "constructor(url: string) {")
	f.P(t(2), "this.eventSource = new EventSource(url);")
	f.P(t(2), "this.eventSource.onmessage = (event) => {")
	f.P(t(3), "try {")
	f.P(t(4), "const data = JSON.parse(event.data) as T;")
	f.P(t(4), "this.listeners.forEach(fn => fn(data));")
	f.P(t(3), "} catch (err) {")
	f.P(t(4), "this.errorHandlers.forEach(fn => fn(err as Error));")
	f.P(t(3), "}")
	f.P(t(2), "};")
	f.P(t(2), "this.eventSource.onerror = () => {")
	f.P(t(3), "this.errorHandlers.forEach(fn => fn(new Error('SSE connection error')));")
	f.P(t(2), "};")
	f.P(t(1), "}")
	f.P()
	f.P(t(1), "onEvent(listener: (data: T) => void): () => void {")
	f.P(t(2), "this.listeners.push(listener);")
	f.P(t(2), "return () => {")
	f.P(t(3), "this.listeners = this.listeners.filter(fn => fn !== listener);")
	f.P(t(2), "};")
	f.P(t(1), "}")
	f.P()
	f.P(t(1), "onError(handler: (error: Error) => void): void {")
	f.P(t(2), "this.errorHandlers.push(handler);")
	f.P(t(1), "}")
	f.P()
	f.P(t(1), "close(): void {")
	f.P(t(2), "this.eventSource.close();")
	f.P(t(1), "}")
	f.P("}")
	f.P()
	// WSTransport
	f.P("export class WSTransport<TIn, TOut> implements DuplexStream<TIn, TOut> {")
	f.P(t(1), "private socket: WebSocket;")
	f.P(t(1), "private listeners: Array<(data: TOut) => void> = [];")
	f.P(t(1), "private errorHandlers: Array<(error: Error) => void> = [];")
	f.P()
	f.P(t(1), "constructor(url: string) {")
	f.P(t(2), "this.socket = new WebSocket(url);")
	f.P(t(2), "this.socket.onmessage = (event) => {")
	f.P(t(3), "try {")
	f.P(t(4), "const data = JSON.parse(event.data as string) as TOut;")
	f.P(t(4), "this.listeners.forEach(fn => fn(data));")
	f.P(t(3), "} catch (err) {")
	f.P(t(4), "this.errorHandlers.forEach(fn => fn(err as Error));")
	f.P(t(3), "}")
	f.P(t(2), "};")
	f.P(t(2), "this.socket.onerror = () => {")
	f.P(t(3), "this.errorHandlers.forEach(fn => fn(new Error('WebSocket connection error')));")
	f.P(t(2), "};")
	f.P(t(1), "}")
	f.P()
	f.P(t(1), "send(data: TIn): void {")
	f.P(t(2), "if (this.socket.readyState === WebSocket.OPEN) {")
	f.P(t(3), "this.socket.send(JSON.stringify(data));")
	f.P(t(2), "}")
	f.P(t(1), "}")
	f.P()
	f.P(t(1), "onEvent(listener: (data: TOut) => void): () => void {")
	f.P(t(2), "this.listeners.push(listener);")
	f.P(t(2), "return () => {")
	f.P(t(3), "this.listeners = this.listeners.filter(fn => fn !== listener);")
	f.P(t(2), "};")
	f.P(t(1), "}")
	f.P()
	f.P(t(1), "onError(handler: (error: Error) => void): void {")
	f.P(t(2), "this.errorHandlers.push(handler);")
	f.P(t(1), "}")
	f.P()
	f.P(t(1), "close(): void {")
	f.P(t(2), "this.socket.close();")
	f.P(t(1), "}")
	f.P("}")
	f.P()
	// createDefaultTransport
	f.P("export function createDefaultTransport(opts?: TransportOptions): ClientTransport {")
	f.P(t(1), "const baseUrl = opts?.baseUrl ?? (typeof DEFAULT_HOST !== 'undefined' ? `https://${DEFAULT_HOST}` : undefined);")
	f.P(t(1), "const resolve = (path: string) => baseUrl ? `${baseUrl}/${path}` : path;")
	f.P(t(1), "const doRequest = opts?.request ?? globalThis.fetch.bind(globalThis);")
	f.P(t(1), "const headers = opts?.headers;")
	f.P()
	f.P(t(1), "return {")
	f.P(t(2), "unary(path, method, body, _meta) {")
	f.P(t(3), "const init: RequestInit = { method, body: body ?? undefined };")
	f.P(t(3), "if (headers) { init.headers = headers; }")
	f.P(t(3), "return doRequest(resolve(path), init).then(r => r.json());")
	f.P(t(2), "},")
	f.P()
	f.P(t(2), "serverStream<T>(path, _meta) {")
	f.P(t(3), "return new SSETransport<T>(resolve(path));")
	f.P(t(2), "},")
	f.P()
	f.P(t(2), "duplexStream<TIn, TOut>(path, _meta) {")
	f.P(t(3), "const wsUrl = resolve(path).replace(/^http/, 'ws');")
	f.P(t(3), "return new WSTransport<TIn, TOut>(wsUrl);")
	f.P(t(2), "},")
	f.P(t(1), "};")
	f.P("}")
	f.P()
}

func generateStreamInterfaceMethod(f *codegen.File, pkg protoreflect.FullName, method protoreflect.MethodDescriptor, rule httprule.Rule) {
	commentGenerator{descriptor: method}.generateLeading(f, 1)
	input := typeFromMessage(pkg, method.Input())
	output := typeFromMessage(pkg, method.Output())
	if method.IsStreamingClient() {
		if hasPathVariables(rule) {
			f.P(t(1), method.Name(), "(request: ", input.Reference(), "): DuplexStream<", input.Reference(), ", ", output.Reference(), ">;")
		} else {
			f.P(t(1), method.Name(), "(): DuplexStream<", input.Reference(), ", ", output.Reference(), ">;")
		}
	} else {
		f.P(t(1), method.Name(), "(request: ", input.Reference(), "): ServerStream<", output.Reference(), ">;")
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
	output := typeFromMessage(pkg, method.Output())
	f.P(t(2), method.Name(), "(request) {")
	generateMethodPathValidation(f, method.Input(), rule)
	generateMethodPath(f, method.Input(), rule)
	generateMethodQuery(f, method.Input(), rule)
	f.P(t(3), "let uri = path;")
	f.P(t(3), "if (queryParams.length > 0) {")
	f.P(t(4), "uri += `?${queryParams.join(\"&\")}`;")
	f.P(t(3), "}")
	f.P(t(3), "return transport.serverStream<", output.Reference(), ">(uri, {")
	f.P(t(4), "service: \"", method.Parent().Name(), "\",")
	f.P(t(4), "method: \"", method.Name(), "\",")
	f.P(t(3), "});")
	f.P(t(2), "},")
}

func generateBidiStreamMethod(
	f *codegen.File,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
) {
	if hasPathVariables(rule) {
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
	path := literalPath(rule)
	f.P(t(2), method.Name(), "() {")
	f.P(t(3), "const path = ", path, ";")
	f.P(t(3), "return transport.duplexStream(path, {")
	f.P(t(4), "service: \"", method.Parent().Name(), "\",")
	f.P(t(4), "method: \"", method.Name(), "\",")
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
	f.P(t(4), "service: \"", method.Parent().Name(), "\",")
	f.P(t(4), "method: \"", method.Name(), "\",")
	f.P(t(3), "});")
	f.P(t(2), "},")
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
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, "\"", "\\\"")
	return `"` + path + `"`
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
