package postgres

import (
	"zedlancer/backend/internal/domain/entities"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}


func (up *UserRepository) CreateUser(user *entities.User) error {
	return up.db.Create(user).Error
}