package entities

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
)

type EmployerReviewSummary struct {
	ID           uint64
	EmployerType enums.EmployerType
	Name         string
	Phone        string
	Province     enums.Province
	City         enums.City
	ReviewStatus enums.EmployerReviewStatus
	CreatedAt    time.Time
}

type EmployerReviewProfile struct {
	Individual *IndividualEmployer
	Legal      *LegalEmployer
}
