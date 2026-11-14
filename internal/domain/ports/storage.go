package ports

import (
	"context"
	"io"
	"time"

	"zedlancer/backend/internal/domain/valueobjects"
)

type Storage interface {
	Put(context.Context, string, io.ReadSeeker, string, int64) (*valueobjects.StoredObject, error)
	Delete(context.Context, string) error
	PresignGet(context.Context, string, time.Duration) (string, error)
}
