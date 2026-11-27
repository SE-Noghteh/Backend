package entities

import (
	"encoding/json"
	"time"
)

// JobPosting is an employer-owned advertisement. Details carries the structured
// fields from the job-ad wizard; title and description remain searchable fields.
type JobPosting struct {
	ID          uint64                 `json:"id"`
	EmployerID  uint64                 `json:"-"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Details     json.RawMessage        `json:"details,omitempty" swaggertype:"object"`
	Attachments []JobPostingAttachment `json:"-"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type JobPostingAttachment struct {
	Name        string
	Key         string
	ContentType string
	Size        int64
}
