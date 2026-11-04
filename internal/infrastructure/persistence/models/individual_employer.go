package models

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type IndividualEmployerModel struct {
	EmployerID        uint64                     `gorm:"primaryKey;autoIncrement:false"`
	Employer          EmployerModel              `gorm:"foreignKey:EmployerID;constraint:OnDelete:CASCADE"`
	FirstName         string                     `gorm:"size:100;not null"`
	LastName          string                     `gorm:"size:100;not null"`
	FatherName        string                     `gorm:"size:100;not null"`
	NationalCode      string                     `gorm:"size:10;not null;uniqueIndex"`
	Gender            enums.Gender               `gorm:"size:6;not null;check:individual_employers_gender,gender IN ('female','male')"`
	BirthDate         string                     `gorm:"size:10;not null"`
	Province          enums.Province             `gorm:"size:40;not null"`
	City              enums.City                 `gorm:"size:40;not null"`
	Address           string                     `gorm:"size:1000;not null"`
	PostalCode        string                     `gorm:"size:10;not null"`
	BusinessName      *string                    `gorm:"size:200"`
	BusinessActivity  *string                    `gorm:"size:1000"`
	Website           *string                    `gorm:"size:2048"`
	HasLegalGuardian  bool                       `gorm:"not null"`
	Phone             string                     `gorm:"size:11;not null;uniqueIndex"`
	Email             *string                    `gorm:"size:254"`
	PasswordHash      string                     `gorm:"size:60;not null"`
	IdentityDocument  valueobjects.StoredObject  `gorm:"serializer:json;type:jsonb;not null"`
	BusinessLicense   *valueobjects.StoredObject `gorm:"serializer:json;type:jsonb"`
	VerificationVideo valueobjects.StoredObject  `gorm:"serializer:json;type:jsonb;not null"`

	PhoneVerifiedAt time.Time                  `gorm:"not null"`
	CreatedAt       time.Time                  `gorm:"not null"`
	ReviewStatus    enums.EmployerReviewStatus `gorm:"size:20;not null;default:pending_review;check:individual_employers_review_status,review_status IN ('pending_review','approved','rejected')"`
}

func (IndividualEmployerModel) TableName() string { return "individual_employers" }
