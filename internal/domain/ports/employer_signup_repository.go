package ports

import (
	"context"
	"zedlancer/backend/internal/domain/entities"
)

type PendingEmployerSignupRepository[T any] interface {
	GetSignup(context.Context, string) (*entities.PendingEmployerSignup[T], error)
	SaveSignup(context.Context, *entities.PendingEmployerSignup[T]) error
	DeleteSignup(context.Context, string) error
}

type EmployerSignupRepository = PendingEmployerSignupRepository[entities.IndividualEmployer]
type LegalEmployerSignupRepository = PendingEmployerSignupRepository[entities.LegalEmployer]
