package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"{{.Module}}/app/{{lower .Service}}/service/internal/service"
{{range $key, $value := .Packages}}
	{{apiPackageAlias (lower $value) $.ApiPackageVersion}} "{{lower $.Module}}/api/gen/go/{{lower $value}}/service/{{lower $.ApiPackageVersion}}"
{{- end}}
)

type GrpcMiddlewares []middleware.Middleware

func NewGrpcMiddleware(ctx *bootstrap.Context) GrpcMiddlewares {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(bLogger.AsKratosLogger(ctx.GetLogger())))
	return ms
}

// NewGrpcServer creates a gRPC server.
func NewGrpcServer(
	ctx *bootstrap.Context,

	middlewares GrpcMiddlewares,
{{range $key, $value := .Services}}
	{{camel $key}}Service *service.{{pascal $key}}Service,
{{- end}}
	// register:param ── 新模块服务形参在此行后注册(make register 工具锚点,勿删)
) (*grpc.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Grpc == nil {
		return nil, nil
	}

	srv, err := rpc.CreateGrpcServer(
		cfg,
		middlewares...,
	)
	if err != nil {
		return nil, err
	}
{{range $key, $value := .Services}}
	{{apiPackageAlias (lower $value) $.ApiPackageVersion}}.Register{{pascal $key}}ServiceServer(srv, {{camel $key}}Service)
{{- end}}

	// register:route ── 新模块路由在此行后注册(make register 工具锚点,勿删)

	return srv, nil
}
