package entities

import (
	"encoding/json"
	"time"
)

type JobPostingVersions struct {
	Id               string           `gorm:"column:id;primaryKey"`
	JobPostingId     string           `gorm:"column:job_posting_id"`
	VersionNo        int32            `gorm:"column:version_no"`
	Title            string           `gorm:"column:title"`
	Description      string           `gorm:"column:description"`
	Eligibility      *string          `gorm:"column:eligibility"`
	CollabStyle      *string          `gorm:"column:collab_style"`
	Duration         *string          `gorm:"column:duration"`
	ExpectedHours    *int32           `gorm:"column:expected_hours"`
	Capacity         int32            `gorm:"column:capacity"`
	ApplyDeadline    *time.Time       `gorm:"column:apply_deadline"`
	DeliveryDeadline *time.Time       `gorm:"column:delivery_deadline"`
	ExpectedOutput   *string          `gorm:"column:expected_output"`
	Amount           *float64         `gorm:"column:amount"`
	PayFormula       *string          `gorm:"column:pay_formula"`
	PayStatus        *string          `gorm:"column:pay_status"`
	MaxRevisions     *int32           `gorm:"column:max_revisions"`
	LocationInfo     *json.RawMessage `gorm:"column:location_info"`
	CreatedAt        *time.Time       `gorm:"column:created_at"`
}

func (JobPostingVersions) TableName() string { return "job_posting_versions" }
