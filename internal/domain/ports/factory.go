package ports

import (
	domainpostgres "zedlancer/backend/internal/domain/ports/postgres"
)

type RepositoryFactory interface {
	UserRepository() domainpostgres.UserRepository
	JobSeekerRepository() JobSeekerRepository
	TokenRevocationRepository() TokenRevocationRepository
}
