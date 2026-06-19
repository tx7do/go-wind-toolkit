module github.com/menta2k/protoc-gen-redact/v3/examples/user

go 1.25.0

require (
	github.com/golang/protobuf v1.5.4
	github.com/menta2k/protoc-gen-redact/v3 v3.0.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260420184626-e10c466a9529 // indirect
)

replace github.com/menta2k/protoc-gen-redact/v3 => ../..
