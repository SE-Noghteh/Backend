package ports

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
)

type LoginAttemptRepository interface {
	GetLoginAttempt(context.Context, string) (*entities.LoginAttempt, error)
	SaveLoginAttempt(context.Context, *entities.LoginAttempt) error
	DeleteLoginAttempt(context.Context, string) error
}
