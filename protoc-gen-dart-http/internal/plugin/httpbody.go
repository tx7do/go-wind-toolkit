package plugin

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// httpBodyFullName 是 HttpBody 下载流的服务端元素类型;protoc-gen-go-http 对
// 这类方法生成的服务端契约是"首帧 Content-Type 走 HTTP 响应头,其余为原始字节块"。
const httpBodyFullName = protoreflect.FullName("google.api.HttpBody")

// isHttpBodyStream 报告方法是否为"元素为 google.api.HttpBody 的服务端流式 RPC"。
// 这类方法在客户端不应按 JSON 消息解码,而应暴露原始字节流。
func isHttpBodyStream(method protoreflect.MethodDescriptor) bool {
	return method.IsStreamingServer() && !method.IsStreamingClient() &&
		method.Output().FullName() == httpBodyFullName
}
