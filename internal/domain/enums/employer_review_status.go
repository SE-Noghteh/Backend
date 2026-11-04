package enums

type EmployerReviewStatus string

const (
	EmployerPendingReview EmployerReviewStatus = "pending_review"
	EmployerApproved      EmployerReviewStatus = "approved"
	EmployerRejected      EmployerReviewStatus = "rejected"
)
