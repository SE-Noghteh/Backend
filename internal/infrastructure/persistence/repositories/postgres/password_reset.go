package postgres

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type PasswordResetRepository struct{ db *gorm.DB }

func NewPasswordResetRepository(db *gorm.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func resetAccountTable(role enums.AccountRole) (table, key string, err error) {
	switch role {
	case enums.RoleJobSeeker:
		return "job_seekers", "id", nil
	case enums.RoleParent:
		return "parents", "id", nil
	case enums.RoleIndividualEmployer:
		return "individual_employers", "employer_id", nil
	case enums.RoleLegalEmployer:
		return "legal_employers", "employer_id", nil
	default:
		return "", "", fmt.Errorf("unsupported password reset role")
	}
}

func (r *PasswordResetRepository) LockAccount(ctx context.Context, role enums.AccountRole, phone string) (*ports.PasswordResetAccount, error) {
	table, key, err := resetAccountTable(role)
	if err != nil {
		return nil, err
	}
	query := r.db.WithContext(ctx).Table(table).Where("phone = ?", phone)
	switch role {
	case enums.RoleJobSeeker:
		var id uint64
		found := query.Select(key).Scan(&id)
		if found.Error != nil || found.RowsAffected == 0 {
			return nil, found.Error
		}
		repo := NewJobSeekerRepository(r.db)
		// Match phone change and deletion, which acquire the account lock before the phone lock.
		if err := repo.LockAccount(ctx, id); err != nil {
			if errors.Is(err, domainerrors.ErrUnauthenticated) {
				return nil, nil
			}
			return nil, err
		}
		if err := repo.LockPhone(ctx, phone); err != nil {
			return nil, err
		}
		query = query.Where("id = ?", id)
	case enums.RoleParent:
		if err := NewParentRepository(r.db).LockPhone(ctx, phone); err != nil {
			return nil, err
		}
	case enums.RoleIndividualEmployer, enums.RoleLegalEmployer:
		if err := NewEmployerRepository(r.db).LockPhone(ctx, phone); err != nil {
			return nil, err
		}
	}
	var account ports.PasswordResetAccount
	result := query.Select(key + " AS id, phone, session_version, phone_verified_at").Scan(&account)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, result.Error
	}
	return &account, nil
}

func (r *PasswordResetRepository) Get(ctx context.Context, role enums.AccountRole, phone string) (*ports.PasswordResetChallenge, error) {
	return r.find(ctx, role, "phone = ?", phone)
}

func (r *PasswordResetRepository) FindByTokenHash(ctx context.Context, role enums.AccountRole, hash string) (*ports.PasswordResetChallenge, error) {
	return r.find(ctx, role, "token_hash = ?", hash)
}

func (r *PasswordResetRepository) find(ctx context.Context, role enums.AccountRole, condition, value string) (*ports.PasswordResetChallenge, error) {
	var row models.PasswordResetModel
	err := r.db.WithContext(ctx).Where("role = ?", role).Where(condition, value).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.PasswordResetChallenge{
		Role: row.Role, AccountID: row.AccountID, SessionVersion: row.SessionVersion, TokenHash: row.TokenHash,
		OTP: entities.SignupOTP{Phone: row.Phone, Salt: row.Salt, CodeHash: row.CodeHash, Attempts: row.Attempts, SentAt: row.SentAt, ExpiresAt: row.ExpiresAt},
	}, nil
}

func (r *PasswordResetRepository) Save(ctx context.Context, challenge *ports.PasswordResetChallenge) error {
	otp := challenge.OTP
	row := models.PasswordResetModel{
		Role: challenge.Role, Phone: otp.Phone, AccountID: challenge.AccountID, SessionVersion: challenge.SessionVersion,
		Salt: otp.Salt, CodeHash: otp.CodeHash, Attempts: otp.Attempts, SentAt: otp.SentAt, ExpiresAt: otp.ExpiresAt, TokenHash: challenge.TokenHash,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (r *PasswordResetRepository) Delete(ctx context.Context, role enums.AccountRole, phone string) error {
	return r.db.WithContext(ctx).Where("role = ? AND phone = ?", role, phone).Delete(&models.PasswordResetModel{}).Error
}

func (r *PasswordResetRepository) UpdatePassword(ctx context.Context, role enums.AccountRole, id uint64, hash string) error {
	table, key, err := resetAccountTable(role)
	if err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Table(table).Where(key+" = ?", id).Updates(map[string]any{
		"password_hash": hash, "session_version": gorm.Expr("session_version + 1"),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domainerrors.ErrUnauthenticated
	}
	return nil
}
