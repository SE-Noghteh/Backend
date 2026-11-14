package entities

import (
	"time"
)

type AvailabilitySlots struct {
	Id          string     `gorm:"column:id;primaryKey"`
	JobSeekerId string     `gorm:"column:job_seeker_id"`
	Kind        *string    `gorm:"column:kind"`
	FromAt      *time.Time `gorm:"column:from_at"`
	ToAt        *time.Time `gorm:"column:to_at"`
}

func (AvailabilitySlots) TableName() string { return "availability_slots" }
