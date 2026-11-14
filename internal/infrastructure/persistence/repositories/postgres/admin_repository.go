package postgres

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type AdminRepository struct{ db *gorm.DB }

func NewAdminRepository(db *gorm.DB) *AdminRepository { return &AdminRepository{db: db} }
func (r *AdminRepository) LockUsername(ctx context.Context, username string) error {
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 3))", username).Error
}
func (r *AdminRepository) FindByUsername(ctx context.Context, username string) (*entities.Admin, error) {
	var row models.AdminModel
	err := r.db.WithContext(ctx).Where("username = ?", username).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entities.Admin{Username: row.Username, PasswordHash: row.Password}, nil
}
func (r *AdminRepository) GetLoginAttempt(ctx context.Context, username string) (*entities.LoginAttempt, error) {
	var row models.AdminLoginAttemptModel
	err := r.db.WithContext(ctx).Where("username = ?", username).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entities.LoginAttempt{Phone: row.Username, Failures: row.Failures, WindowStartedAt: row.WindowStartedAt}, nil
}
func (r *AdminRepository) SaveLoginAttempt(ctx context.Context, a *entities.LoginAttempt) error {
	row := models.AdminLoginAttemptModel{Username: a.Phone, Failures: a.Failures, WindowStartedAt: a.WindowStartedAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "username"}}, DoUpdates: clause.AssignmentColumns([]string{"failures", "window_started_at"})}).Create(&row).Error
}
func (r *AdminRepository) DeleteLoginAttempt(ctx context.Context, username string) error {
	return r.db.WithContext(ctx).Where("username = ?", username).Delete(&models.AdminLoginAttemptModel{}).Error
}
