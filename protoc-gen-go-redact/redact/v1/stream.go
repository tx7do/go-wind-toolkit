package redact

import "google.golang.org/grpc"

// ServerStreamRedactor wraps a grpc.ServerStreamingServer so that each
// response message is redacted before being sent to the client.
//
// It is used by protoc-gen-redact to transparently apply redaction on
// server-streaming RPCs. Only the Send method is overridden; all other
// methods (Context, SetHeader, etc.) are delegated to the embedded stream.
type ServerStreamRedactor[Res any] struct {
	grpc.ServerStreamingServer[Res]
}

// Send redacts the response message then forwards it to the underlying stream.
func (s *ServerStreamRedactor[Res]) Send(m *Res) error {
	Apply(m)
	return s.ServerStreamingServer.Send(m)
}

// BidiStreamRedactor wraps a grpc.BidiStreamingServer so that each response
// message is redacted before being sent to the client.
//
// Request messages received from the client via Recv are forwarded untouched,
// since protoc-gen-redact only redacts server responses.
type BidiStreamRedactor[Req any, Res any] struct {
	grpc.BidiStreamingServer[Req, Res]
}

// Send redacts the response message then forwards it to the underlying stream.
func (s *BidiStreamRedactor[Req, Res]) Send(m *Res) error {
	Apply(m)
	return s.BidiStreamingServer.Send(m)
}

// ClientStreamRedactor wraps a grpc.ClientStreamingServer so that the single
// response message is redacted before being sent to the client.
type ClientStreamRedactor[Req any, Res any] struct {
	grpc.ClientStreamingServer[Req, Res]
}

// SendAndClose redacts the response message then forwards it to the
// underlying stream and closes the stream.
func (s *ClientStreamRedactor[Req, Res]) SendAndClose(m *Res) error {
	Apply(m)
	return s.ClientStreamingServer.SendAndClose(m)
}
