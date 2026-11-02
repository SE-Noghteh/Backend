package ports

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
)

type SignupOTPRepository interface {
	GetOTP(context.Context, string) (*entities.SignupOTP, error)
	SaveOTP(context.Context, *entities.SignupOTP) error
	DeleteOTP(context.Context, string) error
}
