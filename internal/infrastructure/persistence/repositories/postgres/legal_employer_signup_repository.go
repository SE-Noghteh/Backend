package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type LegalEmployerSignupRepository struct{ db *gorm.DB }

func NewLegalEmployerSignupRepository(db *gorm.DB) *LegalEmployerSignupRepository {
	return &LegalEmployerSignupRepository{db: db}
}

func (r *LegalEmployerSignupRepository) GetSignup(ctx context.Context, phone string) (*entities.PendingLegalEmployerSignup, error) {
	var m models.LegalEmployerSignupOTPModel
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
	registration := legalEmployerFromModel(m.Registration)
	return &entities.PendingLegalEmployerSignup{Challenge: signupOTPFromFields(m.OTPChallengeModel), Registration: registration}, nil
}

func (r *LegalEmployerSignupRepository) SaveSignup(ctx context.Context, pending *entities.PendingLegalEmployerSignup) error {
	m := models.LegalEmployerSignupOTPModel{OTPChallengeModel: signupOTPFields(pending.Challenge), Registration: legalEmployerToModel(pending.Registration)}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&m).Error
}

func (r *LegalEmployerSignupRepository) DeleteSignup(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Delete(&models.LegalEmployerSignupOTPModel{}, "phone = ?", phone).Error
}
