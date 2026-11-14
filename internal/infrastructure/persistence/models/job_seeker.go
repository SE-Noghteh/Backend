package models

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type JobSeekerModel struct {
	ID              uint64           `gorm:"primaryKey"`
	FirstName       string           `gorm:"size:100;not null"`
	LastName        string           `gorm:"size:100;not null"`
	NationalCode    string           `gorm:"size:10;not null;uniqueIndex"`
	City            string           `gorm:"size:40;not null;check:job_seekers_city,city IN ('تهران','اصفهان','مشهد','شیراز','تبریز','کرج','قم','اهواز','رشت','یزد')"`
	Address         string           `gorm:"size:1000;not null"`
	BirthDate       string           `gorm:"size:10;not null"`
	Gender          string           `gorm:"size:6;not null;check:job_seekers_gender,gender IN ('female','male')"`
	Interests       []enums.Interest `gorm:"serializer:json;type:jsonb;not null"`
	Phone           string           `gorm:"size:11;not null;uniqueIndex"`
	PasswordHash    string           `gorm:"size:60;not null"`
	PhoneVerifiedAt time.Time        `gorm:"not null"`
	CreatedAt       time.Time        `gorm:"not null"`
	// Nullable for accounts created before video verification was required.
	VerificationVideo *valueobjects.StoredObject `gorm:"serializer:json;type:jsonb"`
}

func (JobSeekerModel) TableName() string { return "job_seekers" }
