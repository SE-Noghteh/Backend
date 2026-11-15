package postgres

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func (r *JobSeekerRepository) GetPasswordResetOTP(ctx context.Context, id uint64) (*entities.SignupOTP, error) {
	var row models.JobSeekerPasswordResetOTPModel
	err := r.db.WithContext(ctx).Where("job_seeker_id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entities.SignupOTP{Phone: row.Phone, Salt: row.Salt, CodeHash: row.CodeHash, Attempts: row.Attempts, SentAt: row.SentAt, ExpiresAt: row.ExpiresAt}, nil
}
func (r *JobSeekerRepository) SavePasswordResetOTP(ctx context.Context, id uint64, otp *entities.SignupOTP) error {
	row := models.JobSeekerPasswordResetOTPModel{JobSeekerID: id, Phone: otp.Phone, Salt: otp.Salt, CodeHash: otp.CodeHash, Attempts: otp.Attempts, SentAt: otp.SentAt, ExpiresAt: otp.ExpiresAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "job_seeker_id"}}, DoUpdates: clause.AssignmentColumns([]string{"phone", "salt", "code_hash", "attempts", "sent_at", "expires_at"})}).Create(&row).Error
}
func (r *JobSeekerRepository) DeletePasswordResetOTP(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("job_seeker_id = ?", id).Delete(&models.JobSeekerPasswordResetOTPModel{}).Error
}
func (r *JobSeekerRepository) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	result := r.db.WithContext(ctx).Model(&models.JobSeekerModel{}).Where("id = ?", id).Updates(map[string]any{"password_hash": hash, "session_version": gorm.Expr("session_version + 1")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrUnauthenticated
	}
	return nil
}
