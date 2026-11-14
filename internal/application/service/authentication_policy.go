package service

import (
	"context"
	"time"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

const loginAttemptWindow = 15 * time.Minute
const maxLoginFailures = 5

func nextLoginAttempt(ctx context.Context, repo ports.LoginAttemptRepository, phone string) (*entities.LoginAttempt, error) {
	attempt, err := repo.GetLoginAttempt(ctx, phone)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if attempt == nil || !now.Before(attempt.WindowStartedAt.Add(loginAttemptWindow)) {
		attempt = &entities.LoginAttempt{Phone: phone, WindowStartedAt: now}
	}
	if attempt.Failures >= maxLoginFailures {
		return nil, domainerrors.ErrLoginThrottled
	}
	return attempt, nil
}

func activePrincipal(ctx context.Context, factory ports.RepositoryFactory, tokens ports.TokenManager, token string, role enums.AccountRole) (*valueobjects.Principal, error) {
	principal, err := tokens.VerifyAccessToken(token, role)
	if err != nil {
		return nil, domainerrors.ErrUnauthenticated
	}
	revoked, err := factory.TokenRevocationRepository().IsRevoked(ctx, principal.TokenID)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, domainerrors.ErrUnauthenticated
	}
	return principal, nil
}

func revokeToken(ctx context.Context, factory ports.RepositoryFactory, tokens ports.TokenManager, token string, role enums.AccountRole) error {
	principal, err := tokens.VerifyAccessToken(token, role)
	if err != nil {
		return domainerrors.ErrUnauthenticated
	}
	return factory.TokenRevocationRepository().Revoke(ctx, principal.TokenID, principal.ExpiresAt)
}
