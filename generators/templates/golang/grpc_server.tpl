package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"{{.Module}}/app/{{lower .Service}}/service/internal/service"
{{range $key, $value := .Packages}}
    {{lower $value}}V1 "{{lower $.Module}}/api/gen/go/{{lower $value}}/service/v1"
{{- end}}
)

type GrpcMiddlewares []middleware.Middleware

func NewGrpcMiddleware(ctx *bootstrap.Context) GrpcMiddlewares {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(ctx.GetLogger()))
	return ms
}

// NewGrpcServer creates a gRPC server.
func NewGrpcServer(
	ctx *bootstrap.Context,
{{range $key, $value := .Services}}
    {{lower $key}}Service *service.{{pascal $key}}Service,
{{- end}}
) (*grpc.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Grpc == nil {
		return nil, nil
	}

	srv, err := rpc.CreateGrpcServer(
		cfg,
		logging.Server(ctx.GetLogger()),
	)
	if err != nil {
		return nil, err
	}
{{range $key, $value := .Services}}
    {{lower $value}}V1.Register{{pascal $key}}ServiceServer(srv, {{lower $key}}Service)
{{- end}}

	return srv, nil
}
