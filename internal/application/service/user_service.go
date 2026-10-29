package service

import (
	"zedlancer/backend/internal/domain/ports"

)


type UserService struct {
	unitOfWork   ports.UnitOfWork

}

func NewUserService(unitOfWork ports.UnitOfWork) *UserService {
	return &UserService{
		unitOfWork:          unitOfWork,
	}
}