package ports

import (
	domainpostgres "zedlancer/backend/internal/domain/ports/postgres"
)

type RepositoryFactory interface {
	UserRepository() domainpostgres.UserRepository
	JobSeekerRepository() JobSeekerRepository
	SignupOTPRepository() SignupOTPRepository
	LoginAttemptRepository() LoginAttemptRepository
	TokenRevocationRepository() TokenRevocationRepository
	EmployerRepository() EmployerRepository
	EmployerSignupRepository() EmployerSignupRepository
	LegalEmployerSignupRepository() LegalEmployerSignupRepository
	EmployerLoginAttemptRepository() LoginAttemptRepository
}
