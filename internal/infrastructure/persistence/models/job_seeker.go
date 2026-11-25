package models

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type JobSeekerModel struct {
	SessionVersion uint64 `gorm:"not null;default:0"`
	AdminReviewModel
	ParentAdminApprovalStatus enums.AdminReviewStatus    `gorm:"->;-:migration"`
	ParentID                  *uint64                    `gorm:"index"`
	ParentRelationship        string                     `gorm:"size:8"`
	ID                        uint64                     `gorm:"primaryKey"`
	FirstName                 string                     `gorm:"size:100;not null"`
	LastName                  string                     `gorm:"size:100;not null"`
	NationalCode              string                     `gorm:"size:10;not null;uniqueIndex"`
	Province                  string                     `gorm:"size:40"`
	City                      string                     `gorm:"size:40;not null"`
	Address                   string                     `gorm:"size:1000;not null"`
	PostalCode                string                     `gorm:"size:10"`
	BirthDate                 string                     `gorm:"size:10;not null"`
	Gender                    string                     `gorm:"size:6;not null;check:job_seekers_gender,gender IN ('female','male')"`
	Interests                 []enums.Interest           `gorm:"serializer:json;type:jsonb;not null"`
	Phone                     string                     `gorm:"size:11;not null;uniqueIndex"`
	PasswordHash              string                     `gorm:"size:60;not null"`
	PhoneVerifiedAt           time.Time                  `gorm:"not null"`
	CreatedAt                 time.Time                  `gorm:"not null"`
	IdentityDocument          *valueobjects.StoredObject `gorm:"serializer:json;type:jsonb"`
	// Nullable for accounts created before video verification was required.
	VerificationVideo *valueobjects.StoredObject `gorm:"serializer:json;type:jsonb"`
}

func (JobSeekerModel) TableName() string { return "job_seekers" }
