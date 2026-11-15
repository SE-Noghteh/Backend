package usecase

import (
	"context"
	"time"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type JobSeekerParentInput struct {
	FirstName string
	LastName  string
	Phone     string
}

type ParentApprovalResult struct {
	valueobjects.JobSeekerApproval
	ParentID       *uint64    `json:"parent_id,omitempty"`
	InvitationCode string     `json:"invitation_code,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

type ParentInvitationInfo struct {
	FirstName          string    `json:"first_name"`
	LastName           string    `json:"last_name"`
	Phone              string    `json:"phone"`
	ExpiresAt          time.Time `json:"expires_at"`
	JobSeekerFirstName string    `json:"job_seeker_first_name"`
}

type SignupParent struct {
	Code             string
	Relationship     string
	Password         string
	ConsentAccepted  bool
	IdentityDocument ports.UploadFile
	ConsentVideo     ports.UploadFile
}

type ParentService interface {
	InviteParent(context.Context, uint64, JobSeekerParentInput) (*ParentApprovalResult, error)
	LinkRegisteredParent(context.Context, uint64, string) (*ParentApprovalResult, error)
	ValidateInvitation(context.Context, string) (*ParentInvitationInfo, error)
	Signup(context.Context, SignupParent) error
	ResendOTP(context.Context, string) error
	VerifyOTP(context.Context, string, string) (*entities.Parent, error)
}
