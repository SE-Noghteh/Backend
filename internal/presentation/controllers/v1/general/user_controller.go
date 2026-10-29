package general

import (
	"zedlancer/backend/internal/application/service"
)

type GeneralUserController struct {
	userService *service.UserService
}

func NewGeneralUserController(userService *service.UserService) *GeneralUserController {
	return &GeneralUserController{
		userService: userService,
	}

}