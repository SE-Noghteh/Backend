package entities

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type JobSeeker struct {
	SessionVersion uint64 `json:"-"`
	valueobjects.AdminReview
	ParentAdminApprovalStatus enums.AdminReviewStatus    `json:"-"`
	ParentID                  *uint64                    `json:"parent_id,omitempty"`
	ParentRelationship        string                     `json:"parent_relationship,omitempty"`
	ID                        uint64                     `json:"id"`
	FirstName                 string                     `json:"first_name"`
	LastName                  string                     `json:"last_name"`
	NationalCode              string                     `json:"national_code"`
	Province                  enums.Province             `json:"province,omitempty"`
	City                      enums.City                 `json:"city"`
	Address                   string                     `json:"address"`
	PostalCode                string                     `json:"postal_code,omitempty"`
	BirthDate                 valueobjects.JalaliDate    `json:"birth_date"`
	Gender                    enums.Gender               `json:"gender"`
	Interests                 []enums.Interest           `json:"interests"`
	Phone                     string                     `json:"phone"`
	PasswordHash              string                     `json:"-"`
	PhoneVerifiedAt           time.Time                  `json:"phone_verified_at"`
	CreatedAt                 time.Time                  `json:"created_at"`
	IdentityDocument          *valueobjects.StoredObject `json:"-"`
	VerificationVideo         *valueobjects.StoredObject `json:"verification_video,omitempty"`
}

// ApprovalAt derives acceptance from current reviews; it is not a cached flag.
func (s *JobSeeker) ApprovalAt(now time.Time) (valueobjects.JobSeekerApproval, error) {
	approval := valueobjects.JobSeekerApproval{ParentConnected: s.ParentID != nil, AdminApproved: s.AdminApprovalStatus == enums.AdminReviewApproved, AdminApprovedParent: s.ParentID != nil && s.ParentAdminApprovalStatus == enums.AdminReviewApproved, SignupStatus: valueobjects.JobSeekerPending, ParentApprovalRequired: true}
	birth, err := s.BirthDate.Gregorian()
	if err != nil {
		return approval, err
	}
	now = now.UTC()
	age := now.Year() - birth.Year()
	if now.Before(birth.AddDate(age, 0, 0)) {
		age--
	}
	approval.ParentApprovalRequired = age < 18
	if approval.AdminApproved && (!approval.ParentApprovalRequired || (approval.ParentConnected && approval.AdminApprovedParent)) {
		approval.SignupStatus = valueobjects.JobSeekerApproved
	}
	return approval, nil
}
