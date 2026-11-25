package entities

import (
	"time"
	"zedlancer/backend/internal/domain/valueobjects"
)

type Parent struct {
	SessionVersion uint64 `json:"-"`
	// ChildIDs is populated for admin responses from job seeker parent connections.
	ChildIDs []uint64 `json:"-"`
	valueobjects.AdminReview
	Relationship      string                     `json:"relationship"`
	ID                uint64                     `json:"id"`
	FirstName         string                     `json:"first_name"`
	LastName          string                     `json:"last_name"`
	Phone             string                     `json:"phone"`
	PasswordHash      string                     `json:"-"`
	IdentityDocument  *valueobjects.StoredObject `json:"identity_document"`
	ConsentVideo      *valueobjects.StoredObject `json:"consent_video"`
	ConsentAcceptedAt time.Time                  `json:"consent_accepted_at"`
	PhoneVerifiedAt   time.Time                  `json:"phone_verified_at"`
	CreatedAt         time.Time                  `json:"created_at"`
}

type ParentInvitation struct {
	JobSeekerID uint64
	CodeHash    string
	FirstName   string
	LastName    string
	Phone       string
	ExpiresAt   time.Time
}

type PendingParentSignup struct {
	InvitationHash string
	Relationship   string
	Registration   *Parent
	Challenge      *SignupOTP
}

// ParentConnectionRequest records a registered parent's decision for one job seeker.
type ParentConnectionRequest struct {
	FirstName   string
	LastName    string
	JobSeekerID uint64
	ParentID    uint64
	ExpiresAt   time.Time
	DecidedAt   *time.Time
	Approved    bool
}
