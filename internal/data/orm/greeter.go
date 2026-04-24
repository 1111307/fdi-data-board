package orm

import "time"

type greeterColumns struct {
	CreatedAt string
	UpdatedAt string
	Id        string
	User      string
}

var GreeterColumns = greeterColumns{
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	Id:        "id",
	User:      "user",
}

// GreeterDo 数据库实体
type GreeterDo struct {
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	Id        int       `gorm:"primaryKey;autoIncrement;comment:id"`
	User      string    `gorm:"index;unique;not null;comment:user"`
}

func (GreeterDo) TableName() string {
	return "greeter"
}
