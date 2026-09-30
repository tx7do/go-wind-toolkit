package models

type Pet struct {
	ID      uint   `gorm:"primarykey;autoIncrement"`
	Name    string `gorm:"size:32"`
	OwnerID uint
	Owner   *User `gorm:"foreignKey:OwnerID;references:ID"`
}

type Group struct {
	ID    uint   `gorm:"primarykey"`
	Title string `gorm:"size:64;comment:组名"`
	Users []User `gorm:"many2many:user_groups"`
}
