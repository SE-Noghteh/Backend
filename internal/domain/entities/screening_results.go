package entities

import (
	"time"
)

type ScreeningResults struct {
	Id                  string     `gorm:"column:id;primaryKey"`
	JobPostingVersionId string     `gorm:"column:job_posting_version_id"`
	RuleCode            string     `gorm:"column:rule_code"`
	FlaggedText         *string    `gorm:"column:flagged_text"`
	Reason              *string    `gorm:"column:reason"`
	DecidedByUserId     *string    `gorm:"column:decided_by_user_id"`
	DecisionReason      *string    `gorm:"column:decision_reason"`
	DecidedAt           *time.Time `gorm:"column:decided_at"`
}

func (ScreeningResults) TableName() string { return "screening_results" }
