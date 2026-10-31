package routes

import (
	"github.com/gin-gonic/gin"
	httpv1 "zedlancer/backend/internal/presentation/routes/http/v1"
	"zedlancer/backend/wire"


)

func SetUpRoutes(ginEngine *gin.Engine, app *wire.Application) {
	v1 := ginEngine.Group("/v1")
	httpv1.SetUpGeneralRoutes(v1, app)

}
