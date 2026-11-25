package ports

import (
	"context"
	"time"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
)

type PasswordResetAccount struct {
	ID              uint64
	Phone           string
	SessionVersion  uint64
	PhoneVerifiedAt time.Time
}

type PasswordResetChallenge struct {
	Role           enums.AccountRole
	AccountID      uint64
	SessionVersion uint64
	OTP            entities.SignupOTP
	TokenHash      *string
}

type PasswordResetRepository interface {
	LockAccount(context.Context, enums.AccountRole, string) (*PasswordResetAccount, error)
	Get(context.Context, enums.AccountRole, string) (*PasswordResetChallenge, error)
	FindByTokenHash(context.Context, enums.AccountRole, string) (*PasswordResetChallenge, error)
	Save(context.Context, *PasswordResetChallenge) error
	Delete(context.Context, enums.AccountRole, string) error
	UpdatePassword(context.Context, enums.AccountRole, uint64, string) error
}

type PasswordResetSender interface {
	SendPasswordResetOTP(context.Context, string, string) error
}
