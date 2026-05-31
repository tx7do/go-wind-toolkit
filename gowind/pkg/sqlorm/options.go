package sqlorm

type OrmType string

const (
	OrmTypeEnt  OrmType = "ent"
	OrmTypeGorm OrmType = "gorm"
)
