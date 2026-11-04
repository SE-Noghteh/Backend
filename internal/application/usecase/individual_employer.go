package usecase

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type SignupIndividualEmployer struct {
	FirstName, LastName, FatherName, NationalCode string
	Gender                                        enums.Gender
	BirthDate                                     valueobjects.JalaliDate
	Province                                      enums.Province
	City                                          enums.City
	Address, PostalCode                           string
	BusinessName, BusinessActivity, Website       string
	HasLegalGuardian                              bool
	Phone, Email, Password                        string

	IdentityDocument  ports.UploadFile
	BusinessLicense   *ports.UploadFile
	VerificationVideo ports.UploadFile
}

type IndividualEmployerLoginResult struct {
	Employer    *entities.IndividualEmployer
	AccessToken *valueobjects.AccessToken
}

type IndividualEmployerService interface {
	RequestOTP(context.Context, string) error
	Signup(context.Context, SignupIndividualEmployer) error
	VerifyOTP(context.Context, VerifySignupOTP) (*entities.IndividualEmployer, error)
	Login(context.Context, string, string) (*IndividualEmployerLoginResult, error)
	Authenticate(context.Context, string) (*entities.IndividualEmployer, error)
	Logout(context.Context, string) error
	DocumentURL(context.Context, string, string) (string, error)
}
