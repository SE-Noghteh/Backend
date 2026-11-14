package httpv1

import (
	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/presentation/middleware"
	"zedlancer/backend/wire"
)

func SetUpAdminRoutes(v1 *gin.RouterGroup, app *wire.Application) {
	controller := app.Controllers.GeneralControllers.AdminController
	admin := v1.Group("/admin")
	admin.POST("/login", controller.Login)
	auth := middleware.NewAdminAuth(app.AdminService)
	admin.Use(auth.Require)
	admin.GET("/job-seekers", controller.JobSeekers)
	admin.GET("/job-seekers/:id", controller.JobSeeker)
	admin.GET("/job-seekers/:id/documents/video", controller.JobSeekerVideo)
	admin.POST("/job-seekers/:id/review", controller.ReviewJobSeeker)
	admin.GET("/parents", controller.Parents)
	admin.GET("/parents/:id", controller.Parent)
	admin.GET("/parents/:id/documents/:kind", controller.ParentDocument)
	admin.POST("/parents/:id/review", controller.ReviewParent)
}
