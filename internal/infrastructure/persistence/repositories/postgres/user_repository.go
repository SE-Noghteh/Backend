package postgres

import (

	"gorm.io/gorm"
	"zedlancer/backend/internal/domain/entities"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (up *UserRepository) CreateUser(user *entities.Users) error {
	return up.db.Create(user).Error
}
