package general

import (
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
)

type LegalEmployerSignupRequest struct {
	CompanyName                string                       `form:"company_name"`
	LegalEntityType            enums.LegalEntityType        `form:"legal_entity_type"`
	CompanyNationalID          string                       `form:"company_national_id"`
	RegistrationNumber         string                       `form:"registration_number"`
	EconomicCode               string                       `form:"economic_code"`
	Province                   enums.Province               `form:"province"`
	City                       enums.City                   `form:"city"`
	Address                    string                       `form:"address"`
	PostalCode                 string                       `form:"postal_code"`
	RepresentativeFirstName    string                       `form:"representative_first_name"`
	RepresentativeLastName     string                       `form:"representative_last_name"`
	RepresentativePosition     enums.RepresentativePosition `form:"representative_position"`
	RepresentativeGender       enums.Gender                 `form:"representative_gender"`
	RepresentativeNationalCode string                       `form:"representative_national_code"`
	HasTaxFile                 bool                         `form:"has_tax_file"`
	Activities                 []enums.Interest             `form:"activities"`
	Website                    string                       `form:"website"`
	RequiresActivityLicense    bool                         `form:"requires_activity_license"`
	Phone                      string                       `form:"phone"`
	Email                      string                       `form:"email"`
	Password                   string                       `form:"password"`
}

func (r LegalEmployerSignupRequest) input() usecase.SignupLegalEmployer {
	return usecase.SignupLegalEmployer{
		CompanyName: r.CompanyName, LegalEntityType: r.LegalEntityType, CompanyNationalID: r.CompanyNationalID,
		RegistrationNumber: r.RegistrationNumber, EconomicCode: r.EconomicCode,
		Province: r.Province, City: r.City, Address: r.Address, PostalCode: r.PostalCode,
		RepresentativeFirstName: r.RepresentativeFirstName, RepresentativeLastName: r.RepresentativeLastName,
		RepresentativePosition: r.RepresentativePosition, RepresentativeGender: r.RepresentativeGender, RepresentativeNationalCode: r.RepresentativeNationalCode,
		HasTaxFile: r.HasTaxFile, Activities: r.Activities, Website: r.Website, RequiresActivityLicense: r.RequiresActivityLicense,
		Phone: r.Phone, Email: r.Email, Password: r.Password,
	}
}
