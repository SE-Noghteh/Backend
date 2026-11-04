package entities

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type JobSeeker struct {
	ID                uint64                     `json:"id"`
	FirstName         string                     `json:"first_name"`
	LastName          string                     `json:"last_name"`
	NationalCode      string                     `json:"national_code"`
	City              enums.City                 `json:"city"`
	Address           string                     `json:"address"`
	BirthDate         valueobjects.JalaliDate    `json:"birth_date"`
	Gender            enums.Gender               `json:"gender"`
	Interests         []enums.Interest           `json:"interests"`
	Phone             string                     `json:"phone"`
	PasswordHash      string                     `json:"-"`
	PhoneVerifiedAt   time.Time                  `json:"phone_verified_at"`
	CreatedAt         time.Time                  `json:"created_at"`
	VerificationVideo *valueobjects.StoredObject `json:"verification_video,omitempty"`
}
