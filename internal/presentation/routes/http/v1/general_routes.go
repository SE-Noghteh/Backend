package httpv1

import (
	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/presentation/middleware"
	"zedlancer/backend/wire"
)

func SetUpGeneralRoutes(v1 *gin.RouterGroup, app *wire.Application) {
	jobSeekers := v1.Group("/job-seekers")
	controller := app.Controllers.GeneralControllers.JobSeekerController
	jobSeekers.GET("/signup-options", controller.Options)
	jobSeekers.POST("/otp", controller.RequestOTP)
	jobSeekers.POST("/signup", controller.Signup)
	jobSeekers.POST("/verify-otp", controller.VerifyOTP)
	authController := app.Controllers.GeneralControllers.JobSeekerAuthController
	jobSeekers.POST("/login", authController.Login)
	jobSeekers.POST("/logout", authController.Logout)
	auth := middleware.NewJobSeekerAuth(app.JobSeekerAuthService)
	jobSeekers.GET("/me", auth.Require, authController.Me)
}
