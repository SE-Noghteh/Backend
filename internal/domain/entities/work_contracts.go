package entities

import (
	"time"
)

type WorkContracts struct {
	CurrentVersionId *string    `gorm:"column:current_version_id"`
	Id               string     `gorm:"column:id;primaryKey"`
	JobApplicationId string     `gorm:"column:job_application_id;unique"`
	Title            string     `gorm:"column:title"`
	Description      string     `gorm:"column:description"`
	Status           string     `gorm:"column:status"`
	TotalAmount      int64      `gorm:"column:total_amount"`
	FundingStatus    string     `gorm:"column:funding_status"`
	FundedAt         *time.Time `gorm:"column:funded_at"`
	StartDate        *time.Time `gorm:"column:start_date"`
	EndDate          *time.Time `gorm:"column:end_date"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (WorkContracts) TableName() string { return "work_contracts" }
