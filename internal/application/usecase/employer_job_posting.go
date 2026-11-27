package usecase

import (
	"context"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

type SaveJobPosting struct {
	Title       string
	Description string
	Details     *entities.JobPostingDetails
}

type EmployerJobPostingService interface {
	ListOpportunities(context.Context, entities.OpportunityFilters) (*entities.OpportunityPage, error)
	ListJobSeekerOpportunities(context.Context, *entities.JobSeeker, entities.OpportunityTab, entities.OpportunityFilters) (*entities.OpportunityPage, error)
	UpdateJobSeekerSkills(context.Context, uint64, []string) error
	SaveOpportunity(context.Context, uint64, uint64) error
	UnsaveOpportunity(context.Context, uint64, uint64) error
	GetOpportunity(context.Context, uint64) (*OpportunityDetail, error)
	Create(context.Context, string, SaveJobPosting) (uint64, error)
	Get(context.Context, string, uint64, entities.JobPostingType) (*JobPostingReview, error)
	Update(context.Context, string, uint64, SaveJobPosting) error
	Delete(context.Context, string, uint64, entities.JobPostingType) error
	ReplaceAttachments(context.Context, string, uint64, []ports.UploadFile) error
}

const (
	MaxJobPostingAttachments     = 5
	MaxJobPostingAttachmentBytes = 10 * 1024 * 1024
)

type JobPostingReview struct {
	*entities.JobPosting
	Attachments []JobPostingAttachmentDownload `json:"attachments"`
}

type OpportunityDetail struct {
	*entities.PublishedOpportunity
	Attachments []JobPostingAttachmentDownload `json:"attachments"`
}

type JobPostingAttachmentDownload struct {
	Name        string `json:"name" example:"brief.pdf"`
	ContentType string `json:"content_type" example:"application/pdf"`
	// File size in bytes.
	Size int64 `json:"size" example:"102400"`
	// Private download URL valid for five minutes.
	URL string `json:"url" format:"uri" example:"https://files.example.com/job-postings/brief.pdf"`
}
