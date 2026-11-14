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
