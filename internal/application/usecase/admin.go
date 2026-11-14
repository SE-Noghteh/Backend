package usecase

import (
	"context"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/valueobjects"
)

type AdminLoginResult struct {
	Username    string
	AccessToken *valueobjects.AccessToken
}
type AdminListFilter struct {
	Status string
	Limit  int
	Offset int
}
type AdminReviewDecision struct {
	Approved bool
	Message  string
	Reviewer string
}
type AdminService interface {
	Login(context.Context, string, string) (*AdminLoginResult, error)
	Authenticate(context.Context, string) (string, error)
	JobSeekers(context.Context, AdminListFilter) ([]*entities.JobSeeker, int64, error)
	Parents(context.Context, AdminListFilter) ([]*entities.Parent, int64, error)
	JobSeeker(context.Context, uint64) (*entities.JobSeeker, error)
	Parent(context.Context, uint64) (*entities.Parent, error)
	JobSeekerVideoURL(context.Context, uint64) (string, error)
	ParentDocumentURL(context.Context, uint64, string) (string, error)
	ReviewJobSeeker(context.Context, uint64, AdminReviewDecision) (*entities.JobSeeker, error)
	ReviewParent(context.Context, uint64, AdminReviewDecision) (*entities.Parent, error)
}
