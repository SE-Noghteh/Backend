package ports

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
)

type EmployerRepository interface {
	LockPhone(context.Context, string) error
	IndividualExists(context.Context, string, string) (bool, error)
	CreateIndividual(context.Context, *entities.IndividualEmployer) error
	FindIndividualByPhone(context.Context, string) (*entities.IndividualEmployer, error)
	FindIndividualByID(context.Context, uint64) (*entities.IndividualEmployer, error)
	LegalExists(context.Context, string, string) (bool, error)
	CreateLegal(context.Context, *entities.LegalEmployer) error
	FindLegalByPhone(context.Context, string) (*entities.LegalEmployer, error)
	FindLegalByID(context.Context, uint64) (*entities.LegalEmployer, error)
}
