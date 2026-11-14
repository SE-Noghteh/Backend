package general

import (
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/valueobjects"
)

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
