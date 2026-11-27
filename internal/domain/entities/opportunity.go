package entities

import (
	"strings"
	"time"
)

var opportunityTextReplacer = strings.NewReplacer("ي", "ی", "ك", "ک", "‌", " ")

func NormalizeOpportunityText(value string) string {
	return strings.Join(strings.Fields(opportunityTextReplacer.Replace(strings.ToLower(value))), " ")
}

type OpportunitySort string

const (
	OpportunityNewest          OpportunitySort = "newest"
	OpportunityClosestDeadline OpportunitySort = "closest_deadline"
	OpportunityHighestAmount   OpportunitySort = "highest_amount"
	OpportunityBestMatch       OpportunitySort = "best_match"
)

type OpportunityAgeGroup string

const (
	OpportunityUnder18 OpportunityAgeGroup = "under_18"
	Opportunity18To23  OpportunityAgeGroup = "18_to_23"
)

type OpportunityDuration string

const (
	OpportunityUnderThreeDays OpportunityDuration = "under_3_days"
	OpportunityUpToWeek       OpportunityDuration = "up_to_week"
	OpportunityUpToMonth      OpportunityDuration = "up_to_month"
	OpportunityOverMonth      OpportunityDuration = "over_month"
)

type OpportunityFilters struct {
	Title          string
	Sort           OpportunitySort
	Categories     []string
	WorkModes      []string
	AgeGroups      []OpportunityAgeGroup
	Types          []JobPostingType
	SkillLevels    []PostingSkillLevel
	MinAmountToman *int64
	MaxAmountToman *int64
	Durations      []OpportunityDuration
	Cities         []string
	Limit          int
	Offset         int
}

type OpportunityTab string

const (
	OpportunityTabAll        OpportunityTab = "all"
	OpportunityTabForMe      OpportunityTab = "for_me"
	OpportunityTabWork       OpportunityTab = "work"
	OpportunityTabInternship OpportunityTab = "internship"
	OpportunityTabSaved      OpportunityTab = "saved"
)

type OpportunityDashboard struct {
	JobSeekerID uint64
	Skills      []string
	Tab         OpportunityTab
}

type Opportunity struct {
	ID                     uint64            `json:"id"`
	Title                  string            `json:"title"`
	EmployerName           string            `json:"employer_name,omitempty"`
	Category               string            `json:"category"`
	City                   string            `json:"city,omitempty"`
	WorkMode               string            `json:"work_mode" enums:"online,onsite,hybrid"`
	OpportunityType        JobPostingType    `json:"opportunity_type" swaggertype:"string" enums:"work,internship"`
	SkillLevel             PostingSkillLevel `json:"skill_level,omitempty" swaggertype:"string" enums:"beginner,intermediate,advanced"`
	MinimumAge             int               `json:"minimum_age"`
	MaximumAge             int               `json:"maximum_age"`
	RequiresParentApproval bool              `json:"requires_parent_approval"`
	ApplyDeadline          string            `json:"apply_deadline" format:"date"`
	// Project budget or monthly internship stipend, in whole toman.
	AmountToman  int64     `json:"amount_toman"`
	DurationDays int64     `json:"duration_days"`
	CreatedAt    time.Time `json:"created_at"`
	// Percentage of distinct posting skills present in the job seeker's profile. Dashboard only.
	MatchPercent *int  `json:"match_percent,omitempty" minimum:"0" maximum:"100"`
	Saved        *bool `json:"saved,omitempty" gorm:"column:is_saved"`
}

type OpportunityPage struct {
	Data   []Opportunity `json:"data"`
	Total  int64         `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type PublishedOpportunity struct {
	Opportunity
	Description           string                 `json:"description"`
	Details               *JobPostingDetails     `json:"details"`
	EmployerVerified      bool                   `json:"employer_verified"`
	AcceptingApplications bool                   `json:"accepting_applications"`
	PublishedAt           *time.Time             `json:"published_at,omitempty"`
	Attachments           []JobPostingAttachment `json:"-"`
}
