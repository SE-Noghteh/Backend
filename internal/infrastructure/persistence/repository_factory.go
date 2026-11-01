package persistence

import (
	"zedlancer/backend/internal/domain/ports"
	domainpostgres "zedlancer/backend/internal/domain/ports/postgres"
	"zedlancer/backend/internal/infrastructure/persistence/repositories/postgres"

	"gorm.io/gorm"
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

func NewRepositoryFactory(db *gorm.DB) *RepositoryFactory {
	return &RepositoryFactory{
		db: db,
	}
}

func (f *RepositoryFactory) UserRepository() domainpostgres.UserRepository {
	return postgres.NewUserRepository(f.db)
}
