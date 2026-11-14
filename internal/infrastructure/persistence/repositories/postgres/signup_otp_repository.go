package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type SignupOTPRepository struct{ db *gorm.DB }

func NewSignupOTPRepository(db *gorm.DB) *SignupOTPRepository { return &SignupOTPRepository{db: db} }

func (r *SignupOTPRepository) GetOTP(ctx context.Context, phone string) (*entities.SignupOTP, error) {
	var model models.SignupOTPModel
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "phone = ?", phone).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var registration *entities.JobSeeker
	if model.Registration != nil {
		var err error
		registration, err = jobSeekerFromModel(model.Registration)
		if err != nil {
			return nil, err
		}
	}
	otp := signupOTPFromFields(model.OTPChallengeModel)
	otp.Registration = registration
	return otp, nil
}
func (r *SignupOTPRepository) SaveOTP(ctx context.Context, otp *entities.SignupOTP) error {
	model := models.SignupOTPModel{OTPChallengeModel: signupOTPFields(otp)}
	if otp.Registration != nil {
		model.Registration = jobSeekerToModel(otp.Registration)
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&model).Error
}
func (r *SignupOTPRepository) DeleteOTP(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Delete(&models.SignupOTPModel{}, "phone = ?", phone).Error
}
