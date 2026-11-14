package ports

import (
	"context"
	"zedlancer/backend/internal/domain/entities"
)

type ParentRepository interface {
	LockPhone(context.Context, string) error
	FindByPhone(context.Context, string) (*entities.Parent, error)
	Create(context.Context, *entities.Parent) error
	SaveInvitation(context.Context, *entities.ParentInvitation) error
	FindInvitation(context.Context, string) (*entities.ParentInvitation, error)
	DeleteInvitation(context.Context, uint64) error
	GetPending(context.Context, string) (*entities.PendingParentSignup, error)
	SavePending(context.Context, *entities.PendingParentSignup) error
	DeletePending(context.Context, string) error
	SetSignupStage(context.Context, uint64, string, *uint64, string) error
}
