package persistence

import (
	domainpostgres "zedlancer/backend/internal/domain/ports/postgres"
	"zedlancer/backend/internal/infrastructure/persistence/repositories/postgres"

	"gorm.io/gorm"
)

type RepositoryFactory struct {
	db *gorm.DB
}

func NewRepositoryFactory(db *gorm.DB) *RepositoryFactory {
	return &RepositoryFactory{
		db: db,
	}
}

func (f *RepositoryFactory) UserRepository() domainpostgres.UserRepository {
	return postgres.NewUserRepository(f.db)
}
