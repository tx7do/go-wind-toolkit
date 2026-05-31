package gorm

import (
	"github.com/tx7do/go-crud/gorm"

	"{{.Module}}/app/{{lower .Service}}/service/internal/data/gorm/models"
)

func init() {
	RegisterMigrateModels()
}

// RegisterMigrateModels registers all GORM models for migration.
func RegisterMigrateModels() {
	gorm.RegisterMigrateModels(
	)
}
