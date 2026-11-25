package usecase

import (
	"context"

	"zedlancer/backend/internal/domain/enums"
)

type PasswordResetService interface {
	Request(context.Context, enums.AccountRole, string) error
	VerifyOTP(context.Context, enums.AccountRole, string, string) (string, error)
	Reset(context.Context, enums.AccountRole, string, string) error
}
