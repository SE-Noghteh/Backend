package usecase

import (
	"context"
	"zedlancer/backend/internal/domain/entities"
)

type LoginJobSeeker struct {
	Phone    string `json:"phone" example:"09123456789" validate:"required"`
	Password string `json:"password" example:"choose-a-private-password" format:"password" validate:"required"`
}

type JobSeekerLoginResult struct {
	JobSeeker   *entities.JobSeeker
	AccessToken *entities.AccessToken
}

type JobSeekerAuthService interface {
	Login(context.Context, LoginJobSeeker) (*JobSeekerLoginResult, error)
	Authenticate(context.Context, string) (*entities.JobSeeker, error)
	Logout(context.Context, string) error
}
