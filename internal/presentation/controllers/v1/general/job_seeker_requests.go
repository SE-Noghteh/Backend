package general

import (
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type SignupRequest struct {
	FirstName    string            `json:"first_name" example:"علی" validate:"required"`
	LastName     string            `json:"last_name" example:"رضایی" validate:"required"`
	NationalCode string            `json:"national_code" example:"0013547895" validate:"required" minLength:"10" maxLength:"10"`
	City         enums.City        `json:"city" example:"تهران" validate:"required"`
	Address      string            `json:"address" example:"خیابان آزادی، پلاک ۱۰" validate:"required"`
	BirthDate    JalaliDateRequest `json:"birth_date" validate:"required"`
	Gender       enums.Gender      `json:"gender" example:"male" validate:"required"`
	Interests    []enums.Interest  `json:"interests" example:"برنامه نویسی,طراحی" validate:"required" minItems:"1" maxItems:"10" uniqueItems:"true"`
	Phone        string            `json:"phone" example:"09123456789" validate:"required"`
	Password     string            `json:"password" example:"choose-a-private-password" validate:"required" minLength:"8" maxLength:"72" format:"password"`
}

func (r SignupRequest) input() usecase.SignupJobSeeker {
	return usecase.SignupJobSeeker{
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		NationalCode: r.NationalCode,
		City:         r.City,
		Address:      r.Address,
		BirthDate:    r.BirthDate.value(),
		Gender:       r.Gender,
		Interests:    r.Interests,
		Phone:        r.Phone,
		Password:     r.Password,
	}
}

type VerifyOTPRequest struct {
	Phone string `json:"phone" example:"09123456789" validate:"required"`
	OTP   string `json:"otp" example:"123456" validate:"required" minLength:"6" maxLength:"6"`
}

func (r VerifyOTPRequest) input() usecase.VerifySignupOTP {
	return usecase.VerifySignupOTP{
		Phone: r.Phone,
		OTP:   r.OTP,
	}
}

type LoginRequest struct {
	Phone    string `json:"phone" example:"09123456789" validate:"required"`
	Password string `json:"password" example:"choose-a-private-password" format:"password" validate:"required"`
}

func (r LoginRequest) input() usecase.LoginJobSeeker {
	return usecase.LoginJobSeeker{
		Phone:    r.Phone,
		Password: r.Password,
	}
}

type JalaliDateRequest struct {
	Year  int `json:"year" example:"1380" validate:"required" minimum:"1" maximum:"3176"`
	Month int `json:"month" example:"1" validate:"required" minimum:"1" maximum:"12"`
	Day   int `json:"day" example:"1" validate:"required" minimum:"1" maximum:"31"`
}

func (d JalaliDateRequest) value() valueobjects.JalaliDate {
	return valueobjects.JalaliDate{Year: d.Year, Month: d.Month, Day: d.Day}
}

type ResendOTPRequest struct {
	Phone string `json:"phone" example:"09123456789" validate:"required"`
}
