package enums

type AdminReviewStatus string

const (
	AdminReviewPending  AdminReviewStatus = "pending"
	AdminReviewApproved AdminReviewStatus = "approved"
	AdminReviewRejected AdminReviewStatus = "rejected"
)
