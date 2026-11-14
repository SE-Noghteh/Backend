package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type LoginAttemptRepository struct {
	db    *gorm.DB
	table string
}

func NewLoginAttemptRepository(db *gorm.DB) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db, table: "job_seeker_login_attempts"}
}

func (r *LoginAttemptRepository) GetLoginAttempt(ctx context.Context, phone string) (*entities.LoginAttempt, error) {
	var model models.LoginAttemptModel
	err := r.db.WithContext(ctx).Table(r.table).Clauses(clause.Locking{Strength: "UPDATE"}).Where("phone = ?", phone).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entities.LoginAttempt{Phone: model.Phone, Failures: model.Failures, WindowStartedAt: model.WindowStartedAt}, nil
}
func (r *LoginAttemptRepository) SaveLoginAttempt(ctx context.Context, attempt *entities.LoginAttempt) error {
	model := models.LoginAttemptModel{Phone: attempt.Phone, Failures: attempt.Failures, WindowStartedAt: attempt.WindowStartedAt}
	return r.db.WithContext(ctx).Table(r.table).Clauses(clause.OnConflict{UpdateAll: true}).Create(&model).Error
}
func (r *LoginAttemptRepository) DeleteLoginAttempt(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Table(r.table).Delete(&models.LoginAttemptModel{}, "phone = ?", phone).Error
}

func NewEmployerLoginAttemptRepository(db *gorm.DB) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db, table: "employer_login_attempts"}
}
