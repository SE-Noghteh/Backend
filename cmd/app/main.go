package main

import (
	"github.com/gin-gonic/gin"
	"zedlancer/backend/bootstrap"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
	"os"
	"log"
)

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
