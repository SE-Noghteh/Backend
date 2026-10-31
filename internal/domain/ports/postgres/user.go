package domainpostgres

import (
	"zedlancer/backend/internal/domain/entities"
)

type UserRepository interface {
	CreateUser(user *entities.User) error
}
