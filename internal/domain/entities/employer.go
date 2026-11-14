package entities

import "zedlancer/backend/internal/domain/enums"

// Employer owns the identity shared by its individual or legal profile.
type Employer struct {
	ID   uint64
	Type enums.EmployerType
}
