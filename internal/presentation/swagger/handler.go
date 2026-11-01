package swagger

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "zedlancer/backend/docs"
)

//go:embed initializer.js
var initializer string

func Handler() gin.HandlerFunc {
	standard := ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
		ginSwagger.PersistAuthorization(true),
	)
	return func(ctx *gin.Context) {
		if ctx.Param("any") == "/swagger-initializer.js" {
			ctx.Header("Cache-Control", "no-store")
			ctx.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(initializer))
			return
		}
		standard(ctx)
	}
}
