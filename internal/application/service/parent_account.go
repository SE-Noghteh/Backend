package service

import (
	"context"
	"errors"
	"time"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type ParentAccountService struct {
	uow       ports.UnitOfWork
	passwords ports.PasswordHasher
	tokens    ports.ParentTokenManager
}

func NewParentAccountService(uow ports.UnitOfWork, passwords ports.PasswordHasher, tokens ports.ParentTokenManager) *ParentAccountService {
	return &ParentAccountService{uow: uow, passwords: passwords, tokens: tokens}
}

func (s *ParentAccountService) Login(ctx context.Context, rawPhone, password string) (*entities.Parent, *valueobjects.AccessToken, error) {
	phone, err := normalizePhone(rawPhone)
	if err != nil || len(password) < 8 || len(password) > 72 {
		return nil, nil, domainerrors.ErrInvalidCredentials
	}

	var parent *entities.Parent
	var token *valueobjects.AccessToken
	var authErr error
	err = s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.ParentRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		attempt, err := nextLoginAttempt(ctx, repo, phone)
		if err != nil {
			return err
		}
		parent, err = repo.FindByPhone(ctx, phone)
		if err != nil {
			return err
		}
		hash := ""
		if parent != nil {
			hash = parent.PasswordHash
		}
		if !s.passwords.Matches(hash, password) || parent == nil || parent.PhoneVerifiedAt.IsZero() {
			attempt.Failures++
			authErr = domainerrors.ErrInvalidCredentials
			return repo.SaveLoginAttempt(ctx, attempt)
		}
		token, err = s.tokens.IssueParentAccessToken(parent.ID, parent.SessionVersion)
		if err != nil {
			return err
		}
		return repo.DeleteLoginAttempt(ctx, phone)
	})
	if err != nil {
		return nil, nil, err
	}
	if authErr != nil {
		return nil, nil, authErr
	}
	return parent, token, nil
}

func (s *ParentAccountService) Authenticate(ctx context.Context, token string) (*entities.Parent, error) {
	f := s.uow.Factory()
	principal, err := activePrincipal(ctx, f, s.tokens, token, enums.RoleParent)
	if err != nil {
		return nil, err
	}
	parent, err := f.ParentRepository().FindByID(ctx, principal.ID)
	if err != nil {
		return nil, err
	}
	if parent == nil || parent.PhoneVerifiedAt.IsZero() || parent.SessionVersion != principal.SessionVersion {
		return nil, domainerrors.ErrUnauthenticated
	}
	return parent, nil
}

func (s *ParentAccountService) Logout(ctx context.Context, token string) error {
	return revokeToken(ctx, s.uow.Factory(), s.tokens, token, enums.RoleParent)
}

func (s *ParentAccountService) Children(ctx context.Context, id uint64) ([]entities.JobSeeker, error) {
	return s.uow.Factory().ParentRepository().ListChildren(ctx, id)
}

func (s *ParentAccountService) ConnectionRequests(ctx context.Context, id uint64) ([]entities.ParentConnectionRequest, error) {
	return s.uow.Factory().ParentRepository().ListConnectionRequests(ctx, id, time.Now().UTC())
}

func (s *ParentAccountService) DecideConnection(ctx context.Context, parent *entities.Parent, jobSeekerID uint64, approved bool) error {
	return s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.ParentRepository()
		if err := repo.LockPhone(ctx, parent.Phone); err != nil {
			return err
		}
		current, err := repo.FindByPhone(ctx, parent.Phone)
		if err != nil {
			return err
		}
		if current == nil || current.ID != parent.ID || current.PhoneVerifiedAt.IsZero() || current.SessionVersion != parent.SessionVersion {
			return domainerrors.ErrUnauthenticated
		}
		seekers := f.JobSeekerRepository()
		if err := seekers.LockAccount(ctx, jobSeekerID); err != nil {
			if errors.Is(err, domainerrors.ErrUnauthenticated) {
				return domainerrors.ErrParentRequestNotFound
			}
			return err
		}
		request, err := repo.FindConnectionRequest(ctx, jobSeekerID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if request == nil || request.ParentID != current.ID || request.DecidedAt != nil || !now.Before(request.ExpiresAt) {
			return domainerrors.ErrParentRequestNotFound
		}
		if approved {
			if err := repo.SetParentConnection(ctx, jobSeekerID, &current.ID, current.Relationship); err != nil {
				return err
			}
		}
		request.DecidedAt = &now
		request.Approved = approved
		return repo.SaveConnectionRequest(ctx, request)
	})
}

