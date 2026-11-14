package usecase

import (
	"context"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type SignupLegalEmployer struct {
	CompanyName                                                string
	LegalEntityType                                            enums.LegalEntityType
	CompanyNationalID, RegistrationNumber, EconomicCode        string
	Province                                                   enums.Province
	City                                                       enums.City
	Address, PostalCode                                        string
	RepresentativeFirstName, RepresentativeLastName            string
	RepresentativePosition                                     enums.RepresentativePosition
	RepresentativeGender                                       enums.Gender
	RepresentativeNationalCode                                 string
	HasTaxFile                                                 bool
	Activities                                                 []enums.Interest
	Website                                                    string
	RequiresActivityLicense                                    bool
	Phone, Email, Password                                     string
	EstablishmentDocument, IdentityDocument, VerificationVideo ports.UploadFile
	ActivityLicense, AuthorizationDocument                     *ports.UploadFile
}

type LegalEmployerLoginResult struct {
	Employer    *entities.LegalEmployer
	AccessToken *valueobjects.AccessToken
}

type LegalEmployerService interface {
	Signup(context.Context, SignupLegalEmployer) error
	RequestOTP(context.Context, string) error
	VerifyOTP(context.Context, VerifySignupOTP) (*entities.LegalEmployer, error)
	Login(context.Context, string, string) (*LegalEmployerLoginResult, error)
	Authenticate(context.Context, string) (*entities.LegalEmployer, error)
	Logout(context.Context, string) error
	DocumentURL(context.Context, string, string) (string, error)
}
