package entities

import (
	"time"
)

type JobSeekers struct {
	Bio                  *string    `gorm:"column:bio"`
	Interests            *string    `gorm:"column:interests"`
	AvailabilityNote     *string    `gorm:"column:availability_note"`
	PublicVisibility     *string    `gorm:"column:public_visibility"`
	Id                   string     `gorm:"column:id;primaryKey"`
	UserId               string     `gorm:"column:user_id;unique"`
	AddressId            *string    `gorm:"column:address_id"`
	ParentId             *string    `gorm:"column:parent_id"`
	FirstName            string     `gorm:"column:first_name"`
	LastName             string     `gorm:"column:last_name"`
	NationalId           string     `gorm:"column:national_id"`
	Gender               *string    `gorm:"column:gender"`
	BirthDate            time.Time  `gorm:"column:birth_date"`
	ParentApprovalStatus string     `gorm:"column:parent_approval_status"`
	ParentApprovedAt     *time.Time `gorm:"column:parent_approved_at"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
}

func (JobSeekers) TableName() string { return "job_seekers" }
