package entities

import (
	"time"
)

type Employers struct {
	Id                 string    `gorm:"column:id;primaryKey"`
	UserId             string    `gorm:"column:user_id;unique"`
	EmployerType       string    `gorm:"column:employer_type"`
	RegistrationStatus string    `gorm:"column:registration_status"`
	VerificationStatus string    `gorm:"column:verification_status"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (Employers) TableName() string { return "employers" }
