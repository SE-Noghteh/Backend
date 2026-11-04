package entities

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type LegalEmployer struct {
	Employer                   Employer
	CompanyName                string
	LegalEntityType            enums.LegalEntityType
	CompanyNationalID          string
	RegistrationNumber         string
	EconomicCode               string
	Province                   enums.Province
	City                       enums.City
	Address                    string
	PostalCode                 string
	RepresentativeFirstName    string
	RepresentativeLastName     string
	RepresentativePosition     enums.RepresentativePosition
	RepresentativeGender       enums.Gender
	RepresentativeNationalCode string
	HasTaxFile                 bool
	Activities                 []enums.Interest
	Website                    *string
	RequiresActivityLicense    bool
	Phone                      string
	Email                      *string
	PasswordHash               string
	EstablishmentDocument      valueobjects.StoredObject
	IdentityDocument           valueobjects.StoredObject
	ActivityLicense            *valueobjects.StoredObject
	AuthorizationDocument      *valueobjects.StoredObject
	VerificationVideo          valueobjects.StoredObject
	PhoneVerifiedAt            time.Time
	CreatedAt                  time.Time
	ReviewStatus               enums.EmployerReviewStatus
}
