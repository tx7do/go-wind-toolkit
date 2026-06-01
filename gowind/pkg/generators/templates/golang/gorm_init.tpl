package gorm

import (
	gormCurd "github.com/tx7do/go-crud/gorm"

	"{{.Module}}/app/{{lower .Service}}/service/internal/data/gorm/models"
)

func init() {
	RegisterMigrateModels()
}

// RegisterMigrateModels registers all GORM models for migration.
func RegisterMigrateModels() {
	gormCurd.RegisterMigrateModels(
{{- range .Models}}
		&models.{{pascal .}}{},
{{- end}}
	)
}
