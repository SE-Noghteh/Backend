package general

import (
	"time"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
)

// LegalEmployerIdentity deliberately excludes password hashes and storage keys.
type LegalEmployerIdentity struct {
	ID                         uint64                       `json:"id"`
	EmployerType               enums.EmployerType           `json:"employer_type"`
	CompanyName                string                       `json:"company_name"`
	LegalEntityType            enums.LegalEntityType        `json:"legal_entity_type"`
	CompanyNationalID          string                       `json:"company_national_id"`
	RegistrationNumber         string                       `json:"registration_number"`
	EconomicCode               string                       `json:"economic_code"`
	Province                   enums.Province               `json:"province"`
	City                       enums.City                   `json:"city"`
	Address                    string                       `json:"address"`
	PostalCode                 string                       `json:"postal_code"`
	RepresentativeFirstName    string                       `json:"representative_first_name"`
	RepresentativeLastName     string                       `json:"representative_last_name"`
	RepresentativePosition     enums.RepresentativePosition `json:"representative_position"`
	RepresentativeGender       enums.Gender                 `json:"representative_gender"`
	RepresentativeNationalCode string                       `json:"representative_national_code"`
	HasTaxFile                 bool                         `json:"has_tax_file"`
	Activities                 []enums.Interest             `json:"activities"`
	Website                    *string                      `json:"website"`
	RequiresActivityLicense    bool                         `json:"requires_activity_license"`
	Phone                      string                       `json:"phone"`
	Email                      *string                      `json:"email"`
	ReviewStatus               enums.EmployerReviewStatus   `json:"review_status"`
	CreatedAt                  time.Time                    `json:"created_at"`
}

type LegalEmployerResponse struct {
	Data LegalEmployerIdentity `json:"data"`
}
type LegalEmployerLoginResponse struct {
	AccessTokenResponse
	Data LegalEmployerIdentity `json:"data"`
}

func identityFromLegalEmployer(e *entities.LegalEmployer) LegalEmployerIdentity {
	return LegalEmployerIdentity{
		ID: e.Employer.ID, EmployerType: e.Employer.Type, CompanyName: e.CompanyName, LegalEntityType: e.LegalEntityType,
		CompanyNationalID: e.CompanyNationalID, RegistrationNumber: e.RegistrationNumber, EconomicCode: e.EconomicCode,
		Province: e.Province, City: e.City, Address: e.Address, PostalCode: e.PostalCode,
		RepresentativeFirstName: e.RepresentativeFirstName, RepresentativeLastName: e.RepresentativeLastName,
		RepresentativePosition: e.RepresentativePosition, RepresentativeGender: e.RepresentativeGender, RepresentativeNationalCode: e.RepresentativeNationalCode,
		HasTaxFile: e.HasTaxFile, Activities: e.Activities, Website: e.Website, RequiresActivityLicense: e.RequiresActivityLicense,
		Phone: e.Phone, Email: e.Email, ReviewStatus: e.ReviewStatus, CreatedAt: e.CreatedAt,
	}
}
