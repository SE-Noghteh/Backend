package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type EmployerSignupRepository struct{ db *gorm.DB }

func NewEmployerSignupRepository(db *gorm.DB) *EmployerSignupRepository {
	return &EmployerSignupRepository{db: db}
}

func (r *EmployerSignupRepository) GetSignup(ctx context.Context, phone string) (*entities.PendingIndividualEmployerSignup, error) {
	var m models.EmployerSignupOTPModel
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("phone = ?", phone).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if m.Registration == nil {
		return nil, nil
	}
	registration, err := individualEmployerFromModel(m.Registration)
	if err != nil {
		return nil, err
	}
	return &entities.PendingIndividualEmployerSignup{Challenge: signupOTPFromFields(m.OTPChallengeModel), Registration: registration}, nil
}

func (r *EmployerSignupRepository) SaveSignup(ctx context.Context, pending *entities.PendingIndividualEmployerSignup) error {
	m := models.EmployerSignupOTPModel{OTPChallengeModel: signupOTPFields(pending.Challenge), Registration: individualEmployerToModel(pending.Registration)}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&m).Error
}

func (r *EmployerSignupRepository) DeleteSignup(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Delete(&models.EmployerSignupOTPModel{}, "phone = ?", phone).Error
}
