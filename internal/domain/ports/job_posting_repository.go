package ports

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
)

type JobPostingRepository interface {
	ListPublished(context.Context, entities.OpportunityFilters) (*entities.OpportunityPage, error)
	ListPublishedForJobSeeker(context.Context, entities.OpportunityFilters, entities.OpportunityDashboard) (*entities.OpportunityPage, error)
	SaveForJobSeeker(context.Context, uint64, uint64) error
	UnsaveForJobSeeker(context.Context, uint64, uint64) error
	FindPublishedOpportunity(context.Context, uint64) (*entities.PublishedOpportunity, error)
	Create(context.Context, *entities.JobPosting) error
	FindOwned(context.Context, uint64, uint64, entities.JobPostingType, bool) (*entities.JobPosting, error)
	UpdateOwned(context.Context, *entities.JobPosting) error
	DeleteOwned(context.Context, uint64, uint64) (bool, error)
}
