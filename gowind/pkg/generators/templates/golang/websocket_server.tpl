package server

import (
	"errors"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"

	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/tx7do/kratos-transport/transport/websocket"
)

type WebsocketMiddlewares []middleware.Middleware

// NewWebsocketMiddleware 创建中间件
func NewWebsocketMiddleware(
	ctx *bootstrap.Context,
) WebsocketMiddlewares {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(ctx.GetLogger()))

	return ms
}

// NewWebsocketServer creates a WebSocket server.
// 该传输以消息类型驱动(srv.RegisterMessageHandler),不注册 gRPC/REST proto 服务,
// 因此没有服务形参;业务处理器在下方 register:route 锚点后手工登记。
func NewWebsocketServer(
	ctx *bootstrap.Context,

	middlewares WebsocketMiddlewares,
) (*websocket.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Websocket == nil {
		return nil, nil
	}

	wsCfg := cfg.Server.Websocket

	var opts []websocket.ServerOption
	if wsCfg.Network != "" {
		opts = append(opts, websocket.WithNetwork(wsCfg.Network))
	}
	if wsCfg.Addr != "" {
		opts = append(opts, websocket.WithAddress(wsCfg.Addr))
	}
	if wsCfg.Path != "" {
		opts = append(opts, websocket.WithPath(wsCfg.Path))
	}
	if wsCfg.Codec != "" {
		opts = append(opts, websocket.WithCodec(wsCfg.Codec))
	}

	srv := websocket.NewServer(opts...)
	if srv == nil {
		return nil, errors.New("server: failed to create websocket server")
	}

	// register:route ── 新模块路由在此行后注册(make register 工具锚点,勿删)
	// WebSocket 以消息类型驱动,手动登记处理函数:
	// srv.RegisterMessageHandler(websocket.NetMessageType(1001), func(sid websocket.SessionID, payload websocket.MessagePayload) error {
	// 	return nil
	// }, func() any { return &YourMessage{} })

	return srv, nil
}
