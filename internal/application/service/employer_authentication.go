package service

import (
	"context"

	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type employerAuthentication[T any] struct {
	uow         ports.UnitOfWork
	passwords   ports.PasswordHasher
	tokens      ports.TokenManager
	role        enums.AccountRole
	findPhone   func(context.Context, ports.EmployerRepository, string) (*T, error)
	findID      func(context.Context, ports.EmployerRepository, uint64) (*T, error)
	credentials func(*T) (id uint64, hash string, verified bool)
}

func (a employerAuthentication[T]) login(ctx context.Context, rawPhone, password string) (*T, *valueobjects.AccessToken, error) {
	phone, err := normalizePhone(rawPhone)
	if err != nil || len(password) < 12 || len(password) > 72 {
		return nil, nil, domainerrors.ErrInvalidCredentials
	}
	var employer *T
	var token *valueobjects.AccessToken
	var authErr error
	err = a.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.EmployerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		attemptRepo := f.EmployerLoginAttemptRepository()
		attempt, err := nextLoginAttempt(ctx, attemptRepo, phone)
		if err != nil {
			return err
		}
		employer, err = a.findPhone(ctx, repo, phone)
		if err != nil {
			return err
		}
		var id uint64
		var hash string
		var verified bool
		if employer != nil {
			id, hash, verified = a.credentials(employer)
		}
		matches := a.passwords.Matches(hash, password)
		if !matches || employer == nil || !verified {
			attempt.Failures++
			authErr = domainerrors.ErrInvalidCredentials
			return attemptRepo.SaveLoginAttempt(ctx, attempt)
		}
		token, err = a.tokens.IssueAccessToken(id, a.role)
		if err != nil {
			return err
		}
		return attemptRepo.DeleteLoginAttempt(ctx, phone)
	})
	if err != nil {
		return nil, nil, err
	}
	if authErr != nil {
		return nil, nil, authErr
	}
	return employer, token, nil
}

func (a employerAuthentication[T]) authenticate(ctx context.Context, token string) (*T, error) {
	f := a.uow.Factory()
	principal, err := activePrincipal(ctx, f, a.tokens, token, a.role)
	if err != nil {
		return nil, err
	}
	e, err := a.findID(ctx, f.EmployerRepository(), principal.ID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, domainerrors.ErrUnauthenticated
	}
	_, _, verified := a.credentials(e)
	if !verified {
		return nil, domainerrors.ErrUnauthenticated
	}
	return e, nil
}
