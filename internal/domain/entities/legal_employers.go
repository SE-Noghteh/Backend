package entities

type LegalEmployers struct {
	EmployerId                   string  `gorm:"column:employer_id;primaryKey"`
	AddressId                    *string `gorm:"column:address_id"`
	OrganizationName             string  `gorm:"column:organization_name"`
	LegalEntityType              string  `gorm:"column:legal_entity_type"`
	NationalLegalId              string  `gorm:"column:national_legal_id"`
	RegistrationNumber           string  `gorm:"column:registration_number"`
	EconomicCode                 *string `gorm:"column:economic_code"`
	RepresentativeFirstName      string  `gorm:"column:representative_first_name"`
	RepresentativeLastName       string  `gorm:"column:representative_last_name"`
	RepresentativeRole           *string `gorm:"column:representative_role"`
	RepresentativeNationalId     string  `gorm:"column:representative_national_id"`
	HasRecentRegistrationChanges bool    `gorm:"column:has_recent_registration_changes"`
	WebsiteUrl                   *string `gorm:"column:website_url"`
}

func (LegalEmployers) TableName() string { return "legal_employers" }
