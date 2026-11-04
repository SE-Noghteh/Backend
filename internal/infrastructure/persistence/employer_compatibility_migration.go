package persistence

import "gorm.io/gorm"

// Preserve historical terms timestamps while allowing signup without that retired field.
// New databases do not create this column.
func migrateLegacyEmployerTerms(db *gorm.DB) error {
	if db.Migrator().HasColumn("individual_employers", "terms_accepted_at") {
		return db.Exec("ALTER TABLE individual_employers ALTER COLUMN terms_accepted_at DROP NOT NULL").Error
	}
	return nil
}
