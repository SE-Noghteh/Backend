package entities

import (
	"time"
)

type IndividualEmployers struct {
	EmployerId         string     `gorm:"column:employer_id;primaryKey"`
	AddressId          *string    `gorm:"column:address_id"`
	FirstName          string     `gorm:"column:first_name"`
	LastName           string     `gorm:"column:last_name"`
	FatherName         *string    `gorm:"column:father_name"`
	NationalId         string     `gorm:"column:national_id"`
	BirthDate          *time.Time `gorm:"column:birth_date"`
	BrandName          *string    `gorm:"column:brand_name"`
	WebsiteUrl         *string    `gorm:"column:website_url"`
	HasBusinessLicense bool       `gorm:"column:has_business_license"`
}

func (IndividualEmployers) TableName() string { return "individual_employers" }
