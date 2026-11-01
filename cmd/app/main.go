package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"zedlancer/backend/bootstrap"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

//go:generate go run -mod=mod github.com/swaggo/swag/cmd/swag init -g main.go -d .,../../internal --parseInternal -o ../../docs

// @title Zedlancer Backend API
// @version 1.0
// @description Submit signup information and password to automatically receive an SMS OTP, verify the code, then login to obtain a JWT. Birth dates use the Jalali calendar. Signup and OTP resend send real SMS messages.
// @BasePath /v1
// @accept json
// @produce json
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Swagger authorizes automatically after login. Manual entry accepts access_token or Bearer followed by access_token. Other API clients must send Authorization: Bearer <access_token>.
func main() {
	gin.DisableConsoleColor()
	ginEngine := gin.New()
	config := bootstrap.Run()

	app, err := wire.InitializeApplication(config)
	if err != nil {
		panic(err)
	}

	routes.SetUpRoutes(ginEngine, app)

	// ginEngine.Run()
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}

	log.Println("Server running on port:", port)

	if err := ginEngine.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
