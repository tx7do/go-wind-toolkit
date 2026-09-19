package sqlorm

import (
	"context"
	"errors"
	"strings"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlorm/internal/ent/entimport"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlorm/internal/gorm"
)

func Importer(
	ctx context.Context,
	orm string,
	drv, dsn,
	schemaPath, daoPath *string,
	includeTables, excludeTables []string,
) error {
	switch OrmType(strings.ToLower(strings.TrimSpace(orm))) {
	case OrmTypeEnt:
		return entimport.Importer(ctx, dsn, schemaPath, includeTables, excludeTables)

	case OrmTypeGorm:
		// gorm://<dir> 数据源:从用户已有的 gorm model 源码回转生成 DAO。
		if dsn != nil && strings.HasPrefix(*dsn, "gorm://") {
			return gorm.ImporterFromGoSchema(ctx, *dsn, schemaPath, daoPath, includeTables)
		}
		return gorm.Importer(ctx, drv, dsn, schemaPath, daoPath, includeTables, excludeTables)

	default:
		return errors.New("sql2orm: unsupported orm type: " + orm)
	}
}
