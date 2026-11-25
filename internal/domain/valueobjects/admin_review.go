package valueobjects

import (
	"time"
	"zedlancer/backend/internal/domain/enums"
)

type AdminReview struct {
	AdminApprovalStatus enums.AdminReviewStatus `json:"admin_approval_status"`
	AdminReviewMessage  string                  `json:"admin_review_message,omitempty"`
	AdminReviewedAt     *time.Time              `json:"admin_reviewed_at,omitempty"`
	AdminReviewedBy     string                  `json:"-"`
}

// EmployerReviewMetadata accompanies the employer profile's existing review_status.
type EmployerReviewMetadata struct {
	Message    string     `json:"admin_review_message,omitempty"`
	ReviewedAt *time.Time `json:"admin_reviewed_at,omitempty"`
	ReviewedBy string     `json:"-"`
}
