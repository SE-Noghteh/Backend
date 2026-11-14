package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type ParentRepository struct{ db *gorm.DB }

func NewParentRepository(db *gorm.DB) *ParentRepository { return &ParentRepository{db: db} }
func (r *ParentRepository) LockPhone(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 2))", phone).Error
}
func parentToModel(p *entities.Parent) *models.ParentModel {
	return &models.ParentModel{ID: p.ID, Relationship: p.Relationship, FirstName: p.FirstName, LastName: p.LastName, Phone: p.Phone, PasswordHash: p.PasswordHash, IdentityDocument: p.IdentityDocument, ConsentVideo: p.ConsentVideo, ConsentAcceptedAt: p.ConsentAcceptedAt, PhoneVerifiedAt: p.PhoneVerifiedAt, CreatedAt: p.CreatedAt}
}
func parentFromModel(p *models.ParentModel) *entities.Parent {
	return &entities.Parent{ID: p.ID, Relationship: p.Relationship, FirstName: p.FirstName, LastName: p.LastName, Phone: p.Phone, PasswordHash: p.PasswordHash, IdentityDocument: p.IdentityDocument, ConsentVideo: p.ConsentVideo, ConsentAcceptedAt: p.ConsentAcceptedAt, PhoneVerifiedAt: p.PhoneVerifiedAt, CreatedAt: p.CreatedAt}
}
func (r *ParentRepository) FindByPhone(ctx context.Context, phone string) (*entities.Parent, error) {
	var row models.ParentModel
	err := r.db.WithContext(ctx).Where("phone = ?", phone).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parentFromModel(&row), nil
}
func (r *ParentRepository) Create(ctx context.Context, p *entities.Parent) error {
	row := parentToModel(p)
	err := r.db.WithContext(ctx).Create(row).Error
	var pgErr *pgconn.PgError
	if errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
		return domainerrors.ErrSignupConflict
	}
	if err == nil {
		p.ID = row.ID
	}
	return err
}
func (r *ParentRepository) SaveInvitation(ctx context.Context, i *entities.ParentInvitation) error {
	row := models.ParentInvitationModel{JobSeekerID: i.JobSeekerID, CodeHash: i.CodeHash, FirstName: i.FirstName, LastName: i.LastName, Phone: i.Phone, ExpiresAt: i.ExpiresAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "job_seeker_id"}}, DoUpdates: clause.AssignmentColumns([]string{"code_hash", "first_name", "last_name", "phone", "expires_at"})}).Create(&row).Error
}
func (r *ParentRepository) FindInvitation(ctx context.Context, hash string) (*entities.ParentInvitation, error) {
	var row models.ParentInvitationModel
	err := r.db.WithContext(ctx).Where("code_hash = ?", hash).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entities.ParentInvitation{JobSeekerID: row.JobSeekerID, CodeHash: row.CodeHash, FirstName: row.FirstName, LastName: row.LastName, Phone: row.Phone, ExpiresAt: row.ExpiresAt}, nil
}
func (r *ParentRepository) DeleteInvitation(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Where("job_seeker_id = ?", id).Delete(&models.ParentInvitationModel{}).Error
}
func (r *ParentRepository) GetPending(ctx context.Context, phone string) (*entities.PendingParentSignup, error) {
	var row models.ParentSignupOTPModel
	err := r.db.WithContext(ctx).Where("phone = ?", phone).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Registration == nil {
		return nil, nil
	}
	return &entities.PendingParentSignup{InvitationHash: row.InvitationHash, Relationship: row.Relationship, Registration: parentFromModel(row.Registration), Challenge: signupOTPFromFields(row.OTPChallengeModel)}, nil
}
func (r *ParentRepository) SavePending(ctx context.Context, p *entities.PendingParentSignup) error {
	row := models.ParentSignupOTPModel{OTPChallengeModel: signupOTPFields(p.Challenge), InvitationHash: p.InvitationHash, Relationship: p.Relationship, Registration: parentToModel(p.Registration)}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "phone"}}, DoUpdates: clause.AssignmentColumns([]string{"salt", "code_hash", "attempts", "sent_at", "expires_at", "invitation_hash", "relationship", "registration"})}).Create(&row).Error
}
func (r *ParentRepository) DeletePending(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Where("phone = ?", phone).Delete(&models.ParentSignupOTPModel{}).Error
}
func (r *ParentRepository) SetSignupStage(ctx context.Context, id uint64, status string, parentID *uint64, relationship string) error {
	result := r.db.WithContext(ctx).Model(&models.JobSeekerModel{}).Where("id = ?", id).Updates(map[string]any{"signup_status": status, "parent_id": parentID, "parent_relationship": relationship})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrUnauthenticated
	}
	return nil
}
