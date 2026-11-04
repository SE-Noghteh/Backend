package models

import "zedlancer/backend/internal/domain/enums"

type EmployerModel struct {
	ID   uint64             `gorm:"primaryKey"`
	Type enums.EmployerType `gorm:"size:10;not null;check:employers_type,type IN ('individual','legal')"`
}

func (EmployerModel) TableName() string { return "employers" }
