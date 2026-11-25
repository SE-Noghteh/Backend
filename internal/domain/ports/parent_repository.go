package ports

import (
	"context"
	"time"
	"zedlancer/backend/internal/domain/entities"
)

type ParentRepository interface {
	LoginAttemptRepository
	FindByID(context.Context, uint64) (*entities.Parent, error)
	ListChildren(context.Context, uint64) ([]entities.JobSeeker, error)
	RecentInvitation(context.Context, string, time.Time) (bool, error)
	SaveConnectionRequest(context.Context, *entities.ParentConnectionRequest) error
	FindConnectionRequest(context.Context, uint64) (*entities.ParentConnectionRequest, error)
	ListConnectionRequests(context.Context, uint64, time.Time) ([]entities.ParentConnectionRequest, error)
	DeleteConnectionRequest(context.Context, uint64) error
	LockPhone(context.Context, string) error
	FindByPhone(context.Context, string) (*entities.Parent, error)
	Create(context.Context, *entities.Parent) error
	SaveInvitation(context.Context, *entities.ParentInvitation) error
	FindInvitation(context.Context, string) (*entities.ParentInvitation, error)
	DeleteInvitation(context.Context, uint64) error
	GetPending(context.Context, string) (*entities.PendingParentSignup, error)
	SavePending(context.Context, *entities.PendingParentSignup) error
	DeletePending(context.Context, string) error
	SetParentConnection(context.Context, uint64, *uint64, string) error
}
