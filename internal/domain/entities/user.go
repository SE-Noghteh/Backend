package entities

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
)

type Users struct {
	AccountStatus           enums.AccountStatus `gorm:"column:account_status"`
	DeactivationRequestedAt *time.Time          `gorm:"column:deactivation_requested_at"`
	PhoneNumber             string              `gorm:"column:phone_number"`
	PasswordHash            string              `gorm:"column:password_hash"`
	PhoneVerifiedAt         *time.Time          `gorm:"column:phone_verified_at"`
	Email                   *string             `gorm:"column:email"`
	EmailVerifiedAt         *time.Time          `gorm:"column:email_verified_at"`
	TermsAcceptedAt         *time.Time          `gorm:"column:terms_accepted_at"`
	PrivacyAcceptedAt       *time.Time          `gorm:"column:privacy_accepted_at"`
	Id                      string              `gorm:"column:id;primaryKey"`
	CreatedAt               time.Time           `gorm:"column:created_at"`
	UpdatedAt               time.Time           `gorm:"column:updated_at"`
}

func (Users) TableName() string { return "users" }
