package entities

type JobPostingType string

const (
	WorkPosting       JobPostingType = "work"
	InternshipPosting JobPostingType = "internship"
)

type PostingSkillLevel string

const (
	PostingSkillBeginner     PostingSkillLevel = "beginner"
	PostingSkillIntermediate PostingSkillLevel = "intermediate"
	PostingSkillAdvanced     PostingSkillLevel = "advanced"
)

func (level PostingSkillLevel) Valid() bool {
	return level == PostingSkillBeginner || level == PostingSkillIntermediate || level == PostingSkillAdvanced
}

// JobPostingDetails contains the work and internship wizard terms. Monetary
// values are whole toman. Dates use Gregorian YYYY-MM-DD for API interchange.
type JobPostingDetails struct {
	OpportunityType        JobPostingType    `json:"opportunity_type" enums:"work,internship"`
	Category               string            `json:"category"`
	Skills                 []string          `json:"skills"`
	SkillLevel             PostingSkillLevel `json:"skill_level,omitempty"`
	ApplyDeadline          string            `json:"apply_deadline" example:"2027-01-01"`
	WorkMode               string            `json:"work_mode" enums:"online,onsite,hybrid"`
	MinimumAge             int               `json:"minimum_age" example:"14"`
	MaximumAge             int               `json:"maximum_age" example:"18"`
	AcceptUnder18          bool              `json:"accept_under_18"`
	RequiresParentApproval bool              `json:"requires_parent_approval"`
	DeliveryDeadline       string            `json:"delivery_deadline,omitempty"`
	DurationDays           int               `json:"duration_days,omitempty"`
	Deliverables           []string          `json:"deliverables,omitempty"`
	DeliveryFormats        []string          `json:"delivery_formats,omitempty"`
	BudgetToman            int64             `json:"budget_toman,omitempty"`
	Internship             *InternshipTerms  `json:"internship,omitempty"`
}

type InternshipTerms struct {
	LearningOutcomes string `json:"learning_outcomes" binding:"required" example:"برنامه‌ریزی تقویم محتوا، نوشتن کپشن و کار با ابزارهای طراحی."`
	// Intern duties. Maximum 50,000 bytes.
	Responsibilities string `json:"responsibilities" binding:"required" example:"همکاری در تولید ۱۰ پست در ماه زیر نظر مربی."`
	Prerequisites    string `json:"prerequisites,omitempty" example:"آشنایی مقدماتی با کانوا"`
	// Gregorian start date. Must be after apply_deadline.
	StartDate         string `json:"start_date" binding:"required" format:"date" example:"2027-02-01"`
	DurationMonths    int    `json:"duration_months" binding:"required" minimum:"1" maximum:"24" example:"3"`
	HoursPerWeek      int    `json:"hours_per_week" binding:"required" minimum:"1" maximum:"168" example:"6"`
	IssuesCertificate bool   `json:"issues_certificate" example:"true"`
	// False for an unpaid internship.
	HasMonthlyStipend bool `json:"has_monthly_stipend" example:"true"`
	// Whole toman. Positive when has_monthly_stipend is true. Omit or use zero when false.
	MonthlyStipendToman int64 `json:"monthly_stipend_toman,omitempty" minimum:"0" example:"500000"`
}
