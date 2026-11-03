package entities

import (
	"time"
)

type Addresses struct {
	Id          string    `gorm:"column:id;primaryKey"`
	Province    string    `gorm:"column:province"`
	City        string    `gorm:"column:city"`
	PostalCode  string    `gorm:"column:postal_code"`
	FullAddress string    `gorm:"column:full_address"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (Addresses) TableName() string { return "addresses" }
