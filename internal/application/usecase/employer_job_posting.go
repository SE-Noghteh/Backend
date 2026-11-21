package usecase

import (
	"context"
	"zedlancer/backend/internal/domain/entities"
)

type SaveJobPosting struct {
	Title       string
	Description string
	Details     *entities.JobPostingDetails
}

type EmployerJobPostingService interface {
	Create(context.Context, string, SaveJobPosting) (uint64, error)
	Update(context.Context, string, uint64, SaveJobPosting) error
	Delete(context.Context, string, uint64) error
}
