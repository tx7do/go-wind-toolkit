package client

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	{{.ApiPackage}} "{{.Module}}/api/gen/go/{{lower .Service}}/service/{{.ApiPackageVersion}}"
)

// New{{pascal .Model}}ServiceClient 创建{{pascal .Model}}服务的gRPC客户端
func New{{pascal .Model}}ServiceClient(ctx *bootstrap.Context) ({{.ApiPackage}}.{{pascal .Model}}ServiceClient, func(), error) {
	conn, err := grpc.DialInsecure(ctx.Context(),
		grpc.WithEndpoint("localhost:9000"),
	)
	if err != nil {
		return nil, nil, err
	}
	return {{.ApiPackage}}.New{{pascal .Model}}ServiceClient(conn), func() { conn.Close() }, nil
}
