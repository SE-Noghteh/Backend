package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func (r *ParentRepository) ListChildren(ctx context.Context, parentID uint64) ([]entities.JobSeeker, error) {
	var rows []models.JobSeekerModel
	if err := jobSeekerApprovalQuery(r.db.WithContext(ctx)).Where("job_seekers.parent_id = ?", parentID).Order("job_seekers.id").Find(&rows).Error; err != nil {
		return nil, err
	}
	children := make([]entities.JobSeeker, 0, len(rows))
	for i := range rows {
		child, err := jobSeekerFromModel(&rows[i])
		if err != nil {
			return nil, err
		}
		children = append(children, *child)
	}
	return children, nil
}

func (r *ParentRepository) RecentInvitation(ctx context.Context, phone string, expiresAfter time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ParentInvitationModel{}).Where("phone = ? AND expires_at > ?", phone, expiresAfter).Count(&count).Error
	return count > 0, err
}

func (r *ParentRepository) SaveConnectionRequest(ctx context.Context, request *entities.ParentConnectionRequest) error {
	row := models.ParentConnectionRequestModel{
		JobSeekerID: request.JobSeekerID, ParentID: request.ParentID, ExpiresAt: request.ExpiresAt,
		DecidedAt: request.DecidedAt, Approved: request.Approved,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "job_seeker_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"parent_id", "expires_at", "decided_at", "approved"}),
	}).Create(&row).Error
}

func (r *ParentRepository) FindConnectionRequest(ctx context.Context, jobSeekerID uint64) (*entities.ParentConnectionRequest, error) {
	var row models.ParentConnectionRequestModel
	err := r.db.WithContext(ctx).Take(&row, "job_seeker_id = ?", jobSeekerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := entities.ParentConnectionRequest{
		JobSeekerID: row.JobSeekerID, ParentID: row.ParentID, ExpiresAt: row.ExpiresAt,
		DecidedAt: row.DecidedAt, Approved: row.Approved,
	}
	return &result, nil
}

func (r *ParentRepository) ListConnectionRequests(ctx context.Context, parentID uint64, now time.Time) ([]entities.ParentConnectionRequest, error) {
	requests := make([]entities.ParentConnectionRequest, 0)
	err := r.db.WithContext(ctx).Model(&models.ParentConnectionRequestModel{}).
		Select("job_seeker_parent_connection_requests.*, job_seekers.first_name, job_seekers.last_name").
		Joins("JOIN job_seekers ON job_seekers.id = job_seeker_parent_connection_requests.job_seeker_id").
		Where("job_seeker_parent_connection_requests.parent_id = ? AND decided_at IS NULL AND expires_at > ?", parentID, now).
		Order("job_seeker_id").Scan(&requests).Error
	return requests, err
}

func (r *ParentRepository) DeleteConnectionRequest(ctx context.Context, jobSeekerID uint64) error {
	return r.db.WithContext(ctx).Delete(&models.ParentConnectionRequestModel{}, "job_seeker_id = ?", jobSeekerID).Error
}

func (r *ParentRepository) GetLoginAttempt(ctx context.Context, phone string) (*entities.LoginAttempt, error) {
	return (&LoginAttemptRepository{db: r.db, table: "parent_login_attempts"}).GetLoginAttempt(ctx, phone)
}

func (r *ParentRepository) SaveLoginAttempt(ctx context.Context, attempt *entities.LoginAttempt) error {
	return (&LoginAttemptRepository{db: r.db, table: "parent_login_attempts"}).SaveLoginAttempt(ctx, attempt)
}

func (r *ParentRepository) DeleteLoginAttempt(ctx context.Context, phone string) error {
	return (&LoginAttemptRepository{db: r.db, table: "parent_login_attempts"}).DeleteLoginAttempt(ctx, phone)
}
