package ports

import (
	"context"
	"time"
)

type TokenRevocationRepository interface {
	IsRevoked(context.Context, string) (bool, error)
	Revoke(context.Context, string, time.Time) error
}
