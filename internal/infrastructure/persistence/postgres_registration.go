package persistence

import (
	"fmt"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"zedlancer/backend/bootstrap"
	"zedlancer/backend/internal/infrastructure/persistence/models"
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
			&models.AdminModel{},
			&models.AdminLoginAttemptModel{},
			&models.JobSeekerModel{},
			&models.SignupOTPModel{},
			&models.JobSeekerPhoneChangeOTPModel{},
			&models.PasswordResetModel{},
			&models.ParentModel{},
			&models.ParentInvitationModel{},
			&models.ParentSignupOTPModel{},
			&models.ParentConnectionRequestModel{},
			&models.ParentLoginAttemptModel{},
			&models.LoginAttemptModel{},
			&models.RevokedAccessTokenModel{},
			&models.EmployerModel{},
			&models.IndividualEmployerModel{},
			&models.LegalEmployerModel{},
			&models.EmployerSignupOTPModel{},
			&models.LegalEmployerSignupOTPModel{},
			&models.EmployerLoginAttemptModel{},
			&models.JobPostingModel{},
		); err != nil {
			panic(err)
		}
		// Parent connection is the flow state; remove the obsolete stage column.
		if err := db.Exec("ALTER TABLE job_seekers DROP COLUMN IF EXISTS signup_status").Error; err != nil {
			panic(err)
		}
		// Existing databases still have the ten-city allowlist. Signup validates against the shared catalog.
		if err := db.Exec("ALTER TABLE job_seekers DROP CONSTRAINT IF EXISTS job_seekers_city").Error; err != nil {
			panic(err)
		}
		if err := migrateLegacyEmployerTerms(db); err != nil {
			panic(err)
		}

		// if err := seeder.SeedAll(); err != nil {
		// 	panic(fmt.Errorf("failed to seed database, error: %s", err.Error()))
		// }

	})

	return dbInstance
}
