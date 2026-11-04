package models

type EmployerSignupOTPModel struct {
	OTPChallengeModel
	Registration *IndividualEmployerModel `gorm:"serializer:json;type:jsonb"`
}

func (EmployerSignupOTPModel) TableName() string { return "employer_signup_otps" }
