package usecase

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type SignupJobSeeker struct {
	FirstName         string
	LastName          string
	NationalCode      string
	Province          enums.Province
	City              enums.City
	Address           string
	PostalCode        string
	BirthDate         valueobjects.JalaliDate
	Gender            enums.Gender
	Interests         []enums.Interest
	Phone             string
	Password          string
	IdentityDocument  ports.UploadFile
	VerificationVideo ports.UploadFile
}

type VerifySignupOTP struct {
	Phone string
	OTP   string
}

type JobSeekerService interface {
	RequestOTP(context.Context, string) error
	Signup(context.Context, SignupJobSeeker) error
	VerifyOTP(context.Context, VerifySignupOTP) (*entities.JobSeeker, error)
	VideoURL(context.Context, uint64) (string, error)
	RequestPhoneChange(context.Context, uint64, string) error
	VerifyPhoneChange(context.Context, uint64, VerifySignupOTP) (*entities.JobSeeker, error)
	DeleteAccount(context.Context, uint64, string) error
}
