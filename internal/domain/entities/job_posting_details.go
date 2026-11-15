package entities

// JobPostingDetails contains the work and internship wizard terms. Monetary
// values are whole toman. Dates use Gregorian YYYY-MM-DD for API interchange.
type JobPostingDetails struct {
	OpportunityType        string           `json:"opportunity_type" enums:"work,internship"`
	Category               string           `json:"category"`
	Skills                 []string         `json:"skills"`
	ApplyDeadline          string           `json:"apply_deadline" example:"2027-01-01"`
	WorkMode               string           `json:"work_mode" enums:"online,onsite,hybrid"`
	MinimumAge             int              `json:"minimum_age" example:"14"`
	MaximumAge             int              `json:"maximum_age" example:"18"`
	AcceptUnder18          bool             `json:"accept_under_18"`
	RequiresParentApproval bool             `json:"requires_parent_approval"`
	DeliveryDeadline       string           `json:"delivery_deadline,omitempty"`
	BudgetToman            int64            `json:"budget_toman,omitempty"`
	Internship             *InternshipTerms `json:"internship,omitempty"`
}

type InternshipTerms struct {
	LearningOutcomes    string `json:"learning_outcomes"`
	Responsibilities    string `json:"responsibilities"`
	Prerequisites       string `json:"prerequisites,omitempty"`
	StartDate           string `json:"start_date" example:"2027-02-01"`
	DurationMonths      int    `json:"duration_months"`
	HoursPerWeek        int    `json:"hours_per_week"`
	IssuesCertificate   bool   `json:"issues_certificate"`
	HasMonthlyStipend   bool   `json:"has_monthly_stipend"`
	MonthlyStipendToman int64  `json:"monthly_stipend_toman,omitempty"`
}
