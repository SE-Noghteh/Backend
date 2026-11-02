package persistence

import (
	"gorm.io/gorm"

	"zedlancer/backend/internal/domain/ports"
	domainpostgres "zedlancer/backend/internal/domain/ports/postgres"
	"zedlancer/backend/internal/infrastructure/persistence/repositories/postgres"
)

type RepositoryFactory struct {
	db *gorm.DB
}

func (f *RepositoryFactory) JobSeekerRepository() ports.JobSeekerRepository {
	return postgres.NewJobSeekerRepository(f.db)
}

func (f *RepositoryFactory) TokenRevocationRepository() ports.TokenRevocationRepository {
	return postgres.NewTokenRevocationRepository(f.db)
}

func (f *RepositoryFactory) SignupOTPRepository() ports.SignupOTPRepository {
	return postgres.NewSignupOTPRepository(f.db)
}

func (f *RepositoryFactory) LoginAttemptRepository() ports.LoginAttemptRepository {
	return postgres.NewLoginAttemptRepository(f.db)
}

func NewRepositoryFactory(db *gorm.DB) *RepositoryFactory {
	return &RepositoryFactory{
		db: db,
	}
}

func (f *RepositoryFactory) UserRepository() domainpostgres.UserRepository {
	return postgres.NewUserRepository(f.db)
}
