package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"

	authzEngine "github.com/tx7do/kratos-authz/engine"
	authz "github.com/tx7do/kratos-authz/middleware"

	authnEngine "github.com/tx7do/kratos-authn/engine"

	swaggerUI "github.com/tx7do/kratos-swagger-ui"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"{{.Module}}/app/{{lower .Service}}/service/cmd/server/assets"
	"{{.Module}}/app/{{lower .Service}}/service/internal/service"

	{{lower .Service}}V1 "{{.Module}}/api/gen/go/{{lower .Service}}/service/v1"

	"{{.Module}}/pkg/middleware/auth"
	applogging "{{.Module}}/pkg/middleware/logging"
)

type RestMiddlewares []middleware.Middleware

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	authorizer authzEngine.Engine,
) RestMiddlewares {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(ctx.GetLogger()))

	// add white list for authentication.
	rpc.AddWhiteList()

	ms = append(ms, selector.Server(
		authn.Server(authenticator),
		auth.Server(),
		authz.Server(authorizer),
	).Match(newRestWhiteListMatcher()).Build())

	return ms
}

// NewRestServer create a REST server.
func NewRestServer(
	ctx *bootstrap.Context,

    middlewares []middleware.Middleware,
	authorizer authzEngine.Engine,
{{range $key, $value := .Services}}
    {{lower $key}}Service *service.{{pascal $key}}Service,
{{- end}}
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
    {{lower $value}}V1.Register{{pascal $key}}ServiceHTTPServer(srv, {{lower $key}}Service)
{{- end}}

	if cfg.GetServer().GetRest().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("{{pascal .Project}} {{.Service}} Service"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

    if authorizer != nil {
    }

	return srv, nil
}
