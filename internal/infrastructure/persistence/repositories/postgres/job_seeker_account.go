package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func (r *JobSeekerRepository) LockAccount(ctx context.Context, id uint64) error {
	var row models.JobSeekerModel
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domainerrors.ErrUnauthenticated
	}
	return err
}

func (r *JobSeekerRepository) UpdatePhone(ctx context.Context, id uint64, phone string) error {
	result := r.db.WithContext(ctx).Model(&models.JobSeekerModel{}).Where("id = ?", id).Updates(map[string]any{"phone": phone, "phone_verified_at": time.Now().UTC()})
	var pgErr *pgconn.PgError
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) || (errors.As(result.Error, &pgErr) && pgErr.Code == "23505") {
		return domainerrors.ErrSignupConflict
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrUnauthenticated
	}
	return nil
}

func (r *JobSeekerRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Where("job_seeker_id = ?", id).Delete(&models.ParentInvitationModel{}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Delete(&models.JobSeekerModel{}, id).Error
}

func (r *JobSeekerRepository) GetPhoneChangeOTP(ctx context.Context, id uint64) (*entities.SignupOTP, error) {
	var row models.JobSeekerPhoneChangeOTPModel
	err := r.db.WithContext(ctx).Where("job_seeker_id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entities.SignupOTP{Phone: row.Phone, Salt: row.Salt, CodeHash: row.CodeHash, Attempts: row.Attempts, SentAt: row.SentAt, ExpiresAt: row.ExpiresAt}, nil
}

func (r *JobSeekerRepository) SavePhoneChangeOTP(ctx context.Context, id uint64, otp *entities.SignupOTP) error {
	row := models.JobSeekerPhoneChangeOTPModel{JobSeekerID: id, Phone: otp.Phone, Salt: otp.Salt, CodeHash: otp.CodeHash, Attempts: otp.Attempts, SentAt: otp.SentAt, ExpiresAt: otp.ExpiresAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "job_seeker_id"}}, DoUpdates: clause.AssignmentColumns([]string{"phone", "salt", "code_hash", "attempts", "sent_at", "expires_at"})}).Create(&row).Error
}

func (r *JobSeekerRepository) DeletePhoneChangeOTP(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("job_seeker_id = ?", id).Delete(&models.JobSeekerPhoneChangeOTPModel{}).Error
}
