package postgres

import (
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func signupOTPFields(otp *entities.SignupOTP) models.OTPChallengeModel {
	return models.OTPChallengeModel{Phone: otp.Phone, Salt: otp.Salt, CodeHash: otp.CodeHash, Attempts: otp.Attempts, SentAt: otp.SentAt, ExpiresAt: otp.ExpiresAt}
}
func signupOTPFromFields(m models.OTPChallengeModel) *entities.SignupOTP {
	return &entities.SignupOTP{Phone: m.Phone, Salt: m.Salt, CodeHash: m.CodeHash, Attempts: m.Attempts, SentAt: m.SentAt, ExpiresAt: m.ExpiresAt}
}
