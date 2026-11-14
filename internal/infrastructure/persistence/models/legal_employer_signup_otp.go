package models

type LegalEmployerSignupOTPModel struct {
	OTPChallengeModel
	Registration *LegalEmployerModel `gorm:"serializer:json;type:jsonb"`
}

func (LegalEmployerSignupOTPModel) TableName() string { return "legal_employer_signup_otps" }
