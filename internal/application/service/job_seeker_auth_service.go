package service

import (
	"context"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

const loginAttemptWindow = 15 * time.Minute
const maxLoginFailures = 5

type JobSeekerAuthService struct {
	uow       ports.UnitOfWork
	passwords ports.PasswordHasher
	tokens    ports.TokenManager
}

func NewJobSeekerAuthService(uow ports.UnitOfWork, passwords ports.PasswordHasher, tokens ports.TokenManager) *JobSeekerAuthService {
	return &JobSeekerAuthService{uow: uow, passwords: passwords, tokens: tokens}
}

func (s *JobSeekerAuthService) Login(ctx context.Context, in usecase.LoginJobSeeker) (*usecase.JobSeekerLoginResult, error) {
	phone, err := normalizePhone(in.Phone)
	if err != nil || len(in.Password) < 8 || len(in.Password) > 72 {
		return nil, ports.ErrInvalidCredentials
	}
	var result *usecase.JobSeekerLoginResult
	var authErr error
	err = s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		now := time.Now().UTC()
		attempt, err := repo.GetLoginAttempt(ctx, phone)
		if err != nil {
			return err
		}
		if attempt == nil || !now.Before(attempt.WindowStartedAt.Add(loginAttemptWindow)) {
			attempt = &entities.LoginAttempt{Phone: phone, WindowStartedAt: now}
		}
		if attempt.Failures >= maxLoginFailures {
			authErr = ports.ErrLoginThrottled
			return nil
		}
		seeker, err := repo.FindByPhone(ctx, phone)
		if err != nil {
			return err
		}
		hash := ""
		if seeker != nil {
			hash = seeker.PasswordHash
		}
		// The password adapter performs a dummy comparison for unknown phone numbers.
		matches := s.passwords.Matches(hash, in.Password)
		if !matches || seeker == nil || seeker.PhoneVerifiedAt.IsZero() {
			attempt.Failures++
			authErr = ports.ErrInvalidCredentials
			return repo.SaveLoginAttempt(ctx, attempt)
		}
		token, err := s.tokens.IssueAccessToken(seeker.ID)
		if err != nil {
			return err
		}
		if err := repo.DeleteLoginAttempt(ctx, phone); err != nil {
			return err
		}
		result = &usecase.JobSeekerLoginResult{JobSeeker: seeker, AccessToken: token}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if authErr != nil {
		return nil, authErr
	}
	return result, nil
}

func (s *JobSeekerAuthService) Authenticate(ctx context.Context, token string) (*entities.JobSeeker, error) {
	principal, err := s.tokens.VerifyAccessToken(token)
	if err != nil {
		return nil, ports.ErrUnauthenticated
	}
	revoked, err := s.uow.Factory().TokenRevocationRepository().IsRevoked(ctx, principal.TokenID)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, ports.ErrUnauthenticated
	}
	seeker, err := s.uow.Factory().JobSeekerRepository().FindByID(ctx, principal.ID)
	if err != nil {
		return nil, err
	}
	if seeker == nil || seeker.PhoneVerifiedAt.IsZero() {
		return nil, ports.ErrUnauthenticated
	}
	return seeker, nil
}

// Logout revokes only this access token. Repeated logout is safe until it expires.
func (s *JobSeekerAuthService) Logout(ctx context.Context, token string) error {
	principal, err := s.tokens.VerifyAccessToken(token)
	if err != nil {
		return ports.ErrUnauthenticated
	}
	return s.uow.Factory().TokenRevocationRepository().Revoke(ctx, principal.TokenID, principal.ExpiresAt)
}
