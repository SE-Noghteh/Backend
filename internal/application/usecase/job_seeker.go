package usecase

import (
	"context"
	"zedlancer/backend/internal/domain/entities"
)

type SignupJobSeeker struct {
	FirstName    string              `json:"first_name" example:"علی" validate:"required"`
	LastName     string              `json:"last_name" example:"رضایی" validate:"required"`
	NationalCode string              `json:"national_code" example:"0013547895" validate:"required" minLength:"10" maxLength:"10"`
	City         entities.City       `json:"city" example:"تهران" validate:"required"`
	Address      string              `json:"address" example:"خیابان آزادی، پلاک ۱۰" validate:"required"`
	BirthDate    entities.JalaliDate `json:"birth_date" validate:"required"`
	Gender       entities.Gender     `json:"gender" example:"male" validate:"required"`
	Interests    []entities.Interest `json:"interests" example:"برنامه نویسی,طراحی" validate:"required" minItems:"1" maxItems:"10" uniqueItems:"true"`
	Phone        string              `json:"phone" example:"09123456789" validate:"required"`
	Password     string              `json:"password" example:"choose-a-private-password" validate:"required" minLength:"8" maxLength:"72" format:"password"`
}

type VerifySignupOTP struct {
	Phone string `json:"phone" example:"09123456789" validate:"required"`
	OTP   string `json:"otp" example:"123456" validate:"required" minLength:"6" maxLength:"6"`
}

type JobSeekerService interface {
	RequestOTP(context.Context, string) error
	Signup(context.Context, SignupJobSeeker) error
	VerifyOTP(context.Context, VerifySignupOTP) (*entities.JobSeeker, error)
}
