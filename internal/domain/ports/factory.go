package ports

import (
	domainpostgres "zedlancer/backend/internal/domain/ports/postgres"
)

type RepositoryFactory interface {
	PasswordResetRepository() PasswordResetRepository
	AdminRepository() AdminRepository
	AdminReviewRepository() AdminReviewRepository
	ParentRepository() ParentRepository
	UserRepository() domainpostgres.UserRepository
	JobSeekerRepository() JobSeekerRepository
	SignupOTPRepository() SignupOTPRepository
	LoginAttemptRepository() LoginAttemptRepository
	TokenRevocationRepository() TokenRevocationRepository
	EmployerRepository() EmployerRepository
	EmployerSignupRepository() EmployerSignupRepository
	LegalEmployerSignupRepository() LegalEmployerSignupRepository
	EmployerLoginAttemptRepository() LoginAttemptRepository
	JobPostingRepository() JobPostingRepository
}
