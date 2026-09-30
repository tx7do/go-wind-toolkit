package models

import (
	"time"

	"github.com/google/uuid"
)

type PostStatus string

type Post struct {
	Base `gorm:"embedded"`

	ID        uint      `gorm:"primarykey;autoIncrement"`
	UUID      uuid.UUID
	Title     string     `gorm:"size:200;not null;comment:标题"`
	Body      string     `gorm:"type:text"`
	Status    PostStatus `gorm:"default:draft"`
	Views     int64      `gorm:"default:0"`
	Published *time.Time
	AuthorID  uint
	Author    *User `gorm:"foreignKey:AuthorID"`
	Comments  []Comment
}

func (Post) TableName() string {
	return "blog_posts"
}

type Comment struct {
	ID     uint `gorm:"primarykey"`
	Body   string
	PostID uint
}
