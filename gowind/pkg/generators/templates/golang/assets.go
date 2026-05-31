package golang

import _ "embed"

var TemplateMap = map[string][]byte{
	"main.tpl":         MainTemplate,
	"wire.tpl":         WireTemplate,
	"wire_set.tpl":     WireSetTemplate,
	"grpc_server.tpl":  GrpcServerTemplate,
	"rest_server.tpl":  RestServerTemplate,
	"ent_client.tpl":   EntClientTemplate,
	"gorm_client.tpl":  GormClientTemplate,
	"gorm_init.tpl":    GormInitTemplate,
	"redis_client.tpl": RedisClientTemplate,
	"ent_repo.tpl":     EntRepoTemplate,
	"gorm_repo.tpl":    GormRepoTemplate,
	"service.tpl":      ServiceTemplate,
	"assets.tpl":       AssetsTemplate,
}

//go:embed main.tpl
var MainTemplate []byte

//go:embed wire.tpl
var WireTemplate []byte

//go:embed wire_set.tpl
var WireSetTemplate []byte

//go:embed grpc_server.tpl
var GrpcServerTemplate []byte

//go:embed rest_server.tpl
var RestServerTemplate []byte

//go:embed ent_client.tpl
var EntClientTemplate []byte

//go:embed gorm_client.tpl
var GormClientTemplate []byte

//go:embed gorm_init.tpl
var GormInitTemplate []byte

//go:embed redis_client.tpl
var RedisClientTemplate []byte

//go:embed ent_repo.tpl
var EntRepoTemplate []byte

//go:embed gorm_repo.tpl
var GormRepoTemplate []byte

//go:embed service.tpl
var ServiceTemplate []byte

//go:embed assets.tpl
var AssetsTemplate []byte
