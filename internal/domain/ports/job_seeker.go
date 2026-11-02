package ports

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
)

// Repository methods run within the application's unit of work.
type JobSeekerRepository interface {
	LockPhone(context.Context, string) error
	Exists(context.Context, string, string) (bool, error)
	Create(context.Context, *entities.JobSeeker) error
	FindByPhone(context.Context, string) (*entities.JobSeeker, error)
	FindByID(context.Context, uint64) (*entities.JobSeeker, error)
}
