package models

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type LegalEmployerModel struct {
	EmployerID                 uint64                       `gorm:"primaryKey;autoIncrement:false"`
	Employer                   EmployerModel                `gorm:"foreignKey:EmployerID;constraint:OnDelete:CASCADE"`
	CompanyName                string                       `gorm:"size:200;not null"`
	LegalEntityType            enums.LegalEntityType        `gorm:"size:30;not null;check:legal_employers_entity_type,legal_entity_type IN ('limited_liability','private_joint_stock','public_joint_stock','cooperative','institution','organization')"`
	CompanyNationalID          string                       `gorm:"size:11;not null;uniqueIndex"`
	RegistrationNumber         string                       `gorm:"size:30;not null"`
	EconomicCode               string                       `gorm:"size:20;not null"`
	Province                   enums.Province               `gorm:"size:40;not null"`
	City                       enums.City                   `gorm:"size:40;not null"`
	Address                    string                       `gorm:"size:1000;not null"`
	PostalCode                 string                       `gorm:"size:10;not null"`
	RepresentativeFirstName    string                       `gorm:"size:100;not null"`
	RepresentativeLastName     string                       `gorm:"size:100;not null"`
	RepresentativePosition     enums.RepresentativePosition `gorm:"size:30;not null;check:legal_employers_representative_position,representative_position IN ('ceo','authorized_signatory','attorney')"`
	RepresentativeGender       enums.Gender                 `gorm:"size:6;not null;check:legal_employers_representative_gender,representative_gender IN ('female','male')"`
	RepresentativeNationalCode string                       `gorm:"size:10;not null"`
	HasTaxFile                 bool                         `gorm:"not null"`
	Activities                 []enums.Interest             `gorm:"serializer:json;type:jsonb;not null"`
	Website                    *string                      `gorm:"size:2048"`
	RequiresActivityLicense    bool                         `gorm:"not null"`
	Phone                      string                       `gorm:"size:11;not null;uniqueIndex"`
	Email                      *string                      `gorm:"size:254"`
	PasswordHash               string                       `gorm:"size:60;not null"`
	EstablishmentDocument      valueobjects.StoredObject    `gorm:"serializer:json;type:jsonb;not null"`
	IdentityDocument           valueobjects.StoredObject    `gorm:"serializer:json;type:jsonb;not null"`
	ActivityLicense            *valueobjects.StoredObject   `gorm:"serializer:json;type:jsonb"`
	AuthorizationDocument      *valueobjects.StoredObject   `gorm:"serializer:json;type:jsonb"`
	VerificationVideo          valueobjects.StoredObject    `gorm:"serializer:json;type:jsonb;not null"`
	PhoneVerifiedAt            time.Time                    `gorm:"not null"`
	CreatedAt                  time.Time                    `gorm:"not null"`
	ReviewStatus               enums.EmployerReviewStatus   `gorm:"size:20;not null;default:pending_review;check:legal_employers_review_status,review_status IN ('pending_review','approved','rejected')"`
}

func (LegalEmployerModel) TableName() string { return "legal_employers" }
