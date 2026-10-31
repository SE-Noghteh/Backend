package httpv1

import (
	"github.com/gin-gonic/gin"
	"zedlancer/backend/wire"
)

func SetUpGeneralRoutes(v1 *gin.RouterGroup, app *wire.Application) {
	// auth := v1.Group("/auth")
	// {
	// 	auth.POST("/register", app.Controllers.GeneralControllers.GeneralUserController.Register)
	// }
}
