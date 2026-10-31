package persistence

import (
	"fmt"
	"zedlancer/backend/bootstrap"
	"zedlancer/backend/internal/domain/entities"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbInstance *gorm.DB
var dbOnce sync.Once

func NewPostgresDatabase() *gorm.DB {
	dbConfig := bootstrap.Run().Env.DB
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		dbConfig.Host,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Name,
		dbConfig.Port,
	)

	dbOnce.Do(func() {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			panic(fmt.Errorf("failed to connect database"))
		}

		dbInstance = db

		// seeder := seeder.NewDatabaseSeeder(db)
		// seeder.ClearAll()

		if err := db.AutoMigrate(
			&entities.User{},
		); err != nil {
			panic(err)
		}

		// if err := seeder.SeedAll(); err != nil {
		// 	panic(fmt.Errorf("failed to seed database, error: %s", err.Error()))
		// }

	})

	return dbInstance
}
