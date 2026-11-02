package usecase

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/valueobjects"
)

type LoginJobSeeker struct {
	Phone    string
	Password string
}

type JobSeekerLoginResult struct {
	JobSeeker   *entities.JobSeeker
	AccessToken *valueobjects.AccessToken
}

type JobSeekerAuthService interface {
	Login(context.Context, LoginJobSeeker) (*JobSeekerLoginResult, error)
	Authenticate(context.Context, string) (*entities.JobSeeker, error)
	Logout(context.Context, string) error
}
