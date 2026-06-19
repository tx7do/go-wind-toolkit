module github.com/menta2k/protoc-gen-redact/v3/examples/user

go 1.24.0

require (
	github.com/golang/protobuf v1.5.4
	github.com/menta2k/protoc-gen-redact/v3 v3.0.0
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251103181224-f26f9409b101 // indirect
)

replace github.com/menta2k/protoc-gen-redact/v3 => ../..
