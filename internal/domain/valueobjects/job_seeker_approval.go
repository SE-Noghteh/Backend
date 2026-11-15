package valueobjects

const (
	JobSeekerPending  = "pending"
	JobSeekerApproved = "approved"
)

type JobSeekerApproval struct {
	ParentConnected        bool   `json:"parent_connected"`
	AdminApproved          bool   `json:"admin_approved"`
	AdminApprovedParent    bool   `json:"admin_approved_parent"`
	SignupStatus           string `json:"signup_status" example:"pending" enums:"pending,approved"`
	ParentApprovalRequired bool   `json:"parent_approval_required"`
}
