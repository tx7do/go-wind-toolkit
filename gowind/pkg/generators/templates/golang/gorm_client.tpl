package client

import (
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	gormBootstrap "github.com/tx7do/kratos-bootstrap/database/gorm"

	"{{.Module}}/app/{{lower .Service}}/service/internal/data/gorm"
	"{{.Module}}/app/{{lower .Service}}/service/internal/data/gorm/dao"
)

// NewGormClient 创建GORM ORM数据库客户端
func NewGormClient(ctx *bootstrap.Context) (*gormCrud.Client, error) {
	l := ctx.NewLoggerHelper("gorm/data/{{lower .Service}}-service")

	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Data == nil {
		l.Fatalf("[GORM] failed getting config")
		return nil, nil
	}

	gorm.RegisterMigrateModels()

	gormClient, err := gormBootstrap.NewGormClient(cfg, l, nil)
	if err != nil {
		l.Fatalf("[GORM] failed creating client: %v", err)
		return nil, err
	}
	if gormClient == nil {
		l.Fatalf("[GORM] failed creating client")
		return nil, err
	}

	dao.SetDefault(gormClient.DB)

	return nil, err
}
