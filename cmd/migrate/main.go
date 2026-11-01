package main

import (
	"log"
	"zedlancer/backend/internal/infrastructure/persistence"
)

func main() {
	db := persistence.NewPostgresDatabase()
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	log.Print("Job seeker migrations completed")
}
