package models

import (
	"time"

	"gorm.io/gorm"
)

// Base 仅被其它模型匿名嵌入,自身不视为模型。
type Base struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	gorm.Model
	Name    string `gorm:"size:64;not null;comment:用户名"`
	Email   string `json:"email" gorm:"column:email_addr"`
	Age     int    `gorm:"default:0"`
	Active  *bool
	Secret  string   `gorm:"-"`
	Tags    []string
	Meta    map[string]string
	Pets    []Pet    `gorm:"foreignKey:OwnerID"`
	Groups  []Group  `gorm:"many2many:user_groups"`
	Profile *Profile
}

type Profile struct {
	ID     uint   `gorm:"primarykey"`
	UserID uint
	Bio    string `gorm:"type:text"`
	User   *User  `gorm:"foreignKey:UserID"`
}
