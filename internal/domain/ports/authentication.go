package ports

import (
	"context"
	"errors"
	"time"
	"zedlancer/backend/internal/domain/entities"
)

var (
	ErrInvalidCredentials = errors.New("invalid phone number or password")
	ErrUnauthenticated    = errors.New("invalid or expired access token")
	ErrLoginThrottled     = errors.New("too many login attempts; try again in 15 minutes")
	ErrNoPendingSignup    = errors.New("submit valid signup information before requesting an OTP")
)

type PasswordHasher interface {
	Hash(string) (string, error)
	Matches(hash, password string) bool
}

// TokenRevocationRepository stores token IDs rather than bearer credentials.
type TokenRevocationRepository interface {
	IsRevoked(context.Context, string) (bool, error)
	Revoke(context.Context, string, time.Time) error
}

type TokenManager interface {
	IssueAccessToken(uint64) (*entities.AccessToken, error)
	VerifyAccessToken(string) (*entities.JobSeekerPrincipal, error)
}
