package ports

import (
	"context"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type AdminTokenManager interface {
	IssueAdminAccessToken(string) (*valueobjects.AccessToken, error)
	VerifyAdminAccessToken(string) (string, error)
}
type AdminRepository interface {
	LoginAttemptRepository
	LockUsername(context.Context, string) error
	FindByUsername(context.Context, string) (*entities.Admin, error)
}
type AdminReviewRepository interface {
	ListEmployers(context.Context, enums.EmployerType, string, int, int) ([]entities.EmployerReviewSummary, int64, error)
	FindEmployer(context.Context, uint64) (*entities.EmployerReviewProfile, error)
	ReviewEmployer(context.Context, uint64, enums.EmployerType, valueobjects.AdminReview) error
	ListJobSeekers(context.Context, string, int, int) ([]*entities.JobSeeker, int64, error)
	ListParents(context.Context, string, int, int) ([]*entities.Parent, int64, error)
	FindJobSeeker(context.Context, uint64) (*entities.JobSeeker, error)
	FindParent(context.Context, uint64) (*entities.Parent, error)
	ReviewJobSeeker(context.Context, uint64, valueobjects.AdminReview) error
	ReviewParent(context.Context, uint64, valueobjects.AdminReview) error
}
