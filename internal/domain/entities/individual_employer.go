package entities

import (
	"time"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type IndividualEmployer struct {
	Employer          Employer
	FirstName         string
	LastName          string
	FatherName        string
	NationalCode      string
	Gender            enums.Gender
	BirthDate         valueobjects.JalaliDate
	Province          enums.Province
	City              enums.City
	Address           string
	PostalCode        string
	BusinessName      *string
	BusinessActivity  *string
	Website           *string
	HasLegalGuardian  bool
	Phone             string
	Email             *string
	PasswordHash      string
	IdentityDocument  valueobjects.StoredObject
	BusinessLicense   *valueobjects.StoredObject
	VerificationVideo valueobjects.StoredObject

	PhoneVerifiedAt time.Time
	CreatedAt       time.Time
	ReviewStatus    enums.EmployerReviewStatus
}
