package service

import (
	"context"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

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
		return nil, domainerrors.ErrInvalidCredentials
	}
	var result *usecase.JobSeekerLoginResult
	var authErr error
	err = s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		attemptRepo := factory.LoginAttemptRepository()
		attempt, err := nextLoginAttempt(ctx, attemptRepo, phone)
		if err != nil {
			return err
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
			authErr = domainerrors.ErrInvalidCredentials
			return attemptRepo.SaveLoginAttempt(ctx, attempt)
		}
		token, err := s.tokens.IssueAccessToken(seeker.ID, enums.RoleJobSeeker)
		if err != nil {
			return err
		}
		if err := attemptRepo.DeleteLoginAttempt(ctx, phone); err != nil {
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
	principal, err := activePrincipal(ctx, s.uow.Factory(), s.tokens, token, enums.RoleJobSeeker)
	if err != nil {
		return nil, err
	}
	seeker, err := s.uow.Factory().JobSeekerRepository().FindByID(ctx, principal.ID)
	if err != nil {
		return nil, err
	}
	if seeker == nil || seeker.PhoneVerifiedAt.IsZero() {
		return nil, domainerrors.ErrUnauthenticated
	}
	return seeker, nil
}

// Logout revokes only this access token. Repeated logout is safe until it expires.
func (s *JobSeekerAuthService) Logout(ctx context.Context, token string) error {
	return revokeToken(ctx, s.uow.Factory(), s.tokens, token, enums.RoleJobSeeker)
}
