package models

type SignupOTPModel struct {
	OTPChallengeModel
	Registration *JobSeekerModel `gorm:"serializer:json;type:jsonb"`
}

func (SignupOTPModel) TableName() string { return "job_seeker_signup_otps" }
