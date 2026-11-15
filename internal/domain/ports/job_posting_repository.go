package ports

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
)

type JobPostingRepository interface {
	Create(context.Context, *entities.JobPosting) error
	UpdateOwned(context.Context, uint64, uint64, string, string, []byte) (bool, error)
	DeleteOwned(context.Context, uint64, uint64) (bool, error)
}
