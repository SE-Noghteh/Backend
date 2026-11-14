package httpv1

import (
	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/presentation/middleware"
	"zedlancer/backend/wire"
)

func SetUpGeneralRoutes(v1 *gin.RouterGroup, app *wire.Application) {
	legalEmployers := v1.Group("/legal-employers")
	legalController := app.Controllers.GeneralControllers.LegalEmployerController
	legalEmployers.GET("/signup-options", legalController.Options)
	legalEmployers.POST("/signup", legalController.Signup)
	legalEmployers.POST("/otp", legalController.RequestOTP)
	legalEmployers.POST("/verify-otp", legalController.VerifyOTP)
	legalEmployers.POST("/login", legalController.Login)
	legalEmployers.POST("/logout", legalController.Logout)
	legalEmployers.GET("/me", legalController.Me)
	legalEmployers.GET("/documents/:kind", legalController.DocumentURL)
	employers := v1.Group("/individual-employers")
	employerController := app.Controllers.GeneralControllers.IndividualEmployerController
	employers.GET("/signup-options", employerController.Options)
	employers.POST("/otp", employerController.RequestOTP)
	employers.POST("/signup", employerController.Signup)
	employers.POST("/verify-otp", employerController.VerifyOTP)
	employers.POST("/login", employerController.Login)
	employers.POST("/logout", employerController.Logout)
	employers.GET("/me", employerController.Me)
	employers.GET("/documents/:kind", employerController.DocumentURL)
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
	jobSeekers.GET("/documents/video", auth.Require, controller.VideoURL)
	jobSeekers.POST("/phone-change", auth.Require, controller.RequestPhoneChange)
	jobSeekers.POST("/phone-change/verify", auth.Require, controller.VerifyPhoneChange)
	jobSeekers.DELETE("/me", auth.Require, controller.DeleteAccount)
	parentController := app.Controllers.GeneralControllers.ParentController
	jobSeekers.POST("/parent-approval", auth.Require, parentController.SetJobSeekerParent)
	parents := v1.Group("/parents")
	parents.POST("/invitations/validate", parentController.ValidateInvitation)
	parents.POST("/signup", parentController.Signup)
	parents.POST("/otp", parentController.ResendOTP)
	parents.POST("/verify-otp", parentController.VerifyOTP)
}
