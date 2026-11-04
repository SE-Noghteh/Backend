package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"zedlancer/backend/internal/domain/entities"
)

const (
	otpLifetime           = 5 * time.Minute
	otpResendDelay        = time.Minute
	pendingSignupLifetime = 30 * time.Minute
	maxOTPAttempts        = 5
)

func otpHash(salt, code string) string {
	sum := sha256.Sum256([]byte(salt + ":" + code))
	return hex.EncodeToString(sum[:])
}

func newSignupChallenge(phone string, now time.Time) (*entities.SignupOTP, string, error) {
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, "", err
	}
	code := fmt.Sprintf("%06d", number.Int64())
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, "", err
	}
	challenge := &entities.SignupOTP{Phone: phone, Salt: hex.EncodeToString(salt), SentAt: now, ExpiresAt: now.Add(otpLifetime)}
	challenge.CodeHash = otpHash(challenge.Salt, code)
	return challenge, code, nil
}
