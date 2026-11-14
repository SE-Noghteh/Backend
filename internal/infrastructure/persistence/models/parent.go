package models

import (
	"time"
	"zedlancer/backend/internal/domain/valueobjects"
)

type ParentModel struct {
	Relationship      string                     `gorm:"size:6;not null"`
	ID                uint64                     `gorm:"primaryKey"`
	FirstName         string                     `gorm:"size:100;not null"`
	LastName          string                     `gorm:"size:100;not null"`
	Phone             string                     `gorm:"size:11;not null;uniqueIndex"`
	PasswordHash      string                     `gorm:"size:60;not null"`
	IdentityDocument  *valueobjects.StoredObject `gorm:"serializer:json;type:jsonb;not null"`
	ConsentVideo      *valueobjects.StoredObject `gorm:"serializer:json;type:jsonb;not null"`
	ConsentAcceptedAt time.Time                  `gorm:"not null"`
	PhoneVerifiedAt   time.Time                  `gorm:"not null"`
	CreatedAt         time.Time                  `gorm:"not null"`
}

func (ParentModel) TableName() string { return "parents" }

type ParentInvitationModel struct {
	JobSeekerID uint64    `gorm:"primaryKey;autoIncrement:false"`
	CodeHash    string    `gorm:"size:64;not null;uniqueIndex"`
	FirstName   string    `gorm:"size:100;not null"`
	LastName    string    `gorm:"size:100;not null"`
	Phone       string    `gorm:"size:11;not null;index"`
	ExpiresAt   time.Time `gorm:"not null;index"`
}

func (ParentInvitationModel) TableName() string { return "job_seeker_parent_invitations" }

type ParentSignupOTPModel struct {
	OTPChallengeModel
	InvitationHash string       `gorm:"size:64;not null;index"`
	Relationship   string       `gorm:"size:6;not null"`
	Registration   *ParentModel `gorm:"serializer:json;type:jsonb;not null"`
}

func (ParentSignupOTPModel) TableName() string { return "parent_signup_otps" }
