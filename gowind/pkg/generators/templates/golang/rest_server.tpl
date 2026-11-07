package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/transport/http"

	swaggerUI "github.com/tx7do/kratos-swagger-ui"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"{{.Module}}/app/{{lower .Service}}/service/cmd/server/assets"
	"{{.Module}}/app/{{lower .Service}}/service/internal/service"
	{{apiPackageAlias (lower .Service) .ApiPackageVersion}} "{{.Module}}/api/gen/go/{{lower .Service}}/service/{{lower .ApiPackageVersion}}"
)

type RestMiddlewares []middleware.Middleware

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
) RestMiddlewares {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(bLogger.AsKratosLogger(ctx.GetLogger())))

	return ms
}

// NewRestServer create a REST server.
func NewRestServer(
	ctx *bootstrap.Context,

	middlewares RestMiddlewares,
{{range $key, $value := .Services}}
	{{camel $key}}Service *service.{{pascal $key}}Service,
{{- end}}
	// register:param ── 新模块服务形参在此行后注册(make register 工具锚点,勿删)
) (*http.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Rest == nil {
		return nil, nil
	}

	srv, err := rpc.CreateRestServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}
{{range $key, $value := .Services}}
	{{apiPackageAlias (lower $.Service) $.ApiPackageVersion}}.Register{{pascal $key}}ServiceHTTPServer(srv, {{camel $key}}Service)
{{- end}}

	// register:route ── 新模块路由在此行后注册(make register 工具锚点,勿删)

	if cfg.GetServer().GetRest().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("{{pascal .Project}} {{.Service}} Service"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	return srv, nil
}
