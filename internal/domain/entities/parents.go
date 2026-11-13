package entities

import (
	"time"
)

type Parents struct {
	Id        string    `gorm:"column:id;primaryKey"`
	UserId    string    `gorm:"column:user_id;unique"`
	AddressId *string   `gorm:"column:address_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Parents) TableName() string { return "parents" }
