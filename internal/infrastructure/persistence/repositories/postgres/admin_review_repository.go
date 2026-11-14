package postgres

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func adminReviewToModel(r valueobjects.AdminReview) models.AdminReviewModel {
	return models.AdminReviewModel{AdminApprovalStatus: r.AdminApprovalStatus, AdminReviewMessage: r.AdminReviewMessage, AdminReviewedAt: r.AdminReviewedAt, AdminReviewedBy: r.AdminReviewedBy}
}
func adminReviewFromModel(r models.AdminReviewModel) valueobjects.AdminReview {
	return valueobjects.AdminReview{AdminApprovalStatus: r.AdminApprovalStatus, AdminReviewMessage: r.AdminReviewMessage, AdminReviewedAt: r.AdminReviewedAt, AdminReviewedBy: r.AdminReviewedBy}
}

type AdminReviewRepository struct{ db *gorm.DB }

func NewAdminReviewRepository(db *gorm.DB) *AdminReviewRepository {
	return &AdminReviewRepository{db: db}
}
func reviewFilter(db *gorm.DB, status string) *gorm.DB {
	if status != "all" {
		return db.Where("admin_approval_status = ?", status)
	}
	return db
}
func (r *AdminReviewRepository) ListJobSeekers(ctx context.Context, status string, limit, offset int) ([]*entities.JobSeeker, int64, error) {
	cutoff, err := adultBirthCutoff(time.Now())
	if err != nil {
		return nil, 0, err
	}
	query := jobSeekerApprovalQuery(r.db.WithContext(ctx)).Where("(job_seekers.parent_id IS NOT NULL OR job_seekers.birth_date <= ?)", cutoff.String())
	approvedSQL := "(job_seekers.admin_approval_status = 'approved' AND (job_seekers.birth_date <= ? OR (job_seekers.parent_id IS NOT NULL AND COALESCE(approval_parent.admin_approval_status, 'pending') = 'approved')))"
	switch status {
	case "approved":
		query = query.Where(approvedSQL, cutoff.String())
	case "pending":
		query = query.Where("job_seekers.admin_approval_status <> ?", "rejected").Where("NOT "+approvedSQL, cutoff.String())
	case "rejected":
		query = query.Where("job_seekers.admin_approval_status = ?", "rejected")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.JobSeekerModel
	if err := query.Order("job_seekers.created_at ASC, job_seekers.id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*entities.JobSeeker, 0, len(rows))
	for _, row := range rows {
		seeker, err := jobSeekerFromModel(&row)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, seeker)
	}
	return result, total, nil
}

func adultBirthCutoff(now time.Time) (valueobjects.JalaliDate, error) {
	now = now.UTC()
	cutoff := time.Date(now.Year()-18, now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if cutoff.Month() != now.Month() {
		cutoff = time.Date(now.Year()-18, now.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	}
	return valueobjects.JalaliFromGregorian(cutoff)
}
func (r *AdminReviewRepository) ListParents(ctx context.Context, status string, limit, offset int) ([]*entities.Parent, int64, error) {
	query := reviewFilter(r.db.WithContext(ctx).Model(&models.ParentModel{}), status)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.ParentModel
	if err := query.Order("created_at ASC, id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*entities.Parent, 0, len(rows))
	for _, row := range rows {
		result = append(result, parentFromModel(&row))
	}
	return result, total, nil
}
func (r *AdminReviewRepository) FindJobSeeker(ctx context.Context, id uint64) (*entities.JobSeeker, error) {
	return NewJobSeekerRepository(r.db).FindByID(ctx, id)
}
func (r *AdminReviewRepository) FindParent(ctx context.Context, id uint64) (*entities.Parent, error) {
	var row models.ParentModel
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parentFromModel(&row), nil
}
func reviewUpdates(review valueobjects.AdminReview) map[string]any {
	return map[string]any{"admin_approval_status": review.AdminApprovalStatus, "admin_review_message": review.AdminReviewMessage, "admin_reviewed_at": review.AdminReviewedAt, "admin_reviewed_by": review.AdminReviewedBy}
}
func (r *AdminReviewRepository) ReviewJobSeeker(ctx context.Context, id uint64, review valueobjects.AdminReview) error {
	result := r.db.WithContext(ctx).Model(&models.JobSeekerModel{}).Where("id = ?", id).Updates(reviewUpdates(review))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrReviewNotFound
	}
	return nil
}
func (r *AdminReviewRepository) ReviewParent(ctx context.Context, id uint64, review valueobjects.AdminReview) error {
	result := r.db.WithContext(ctx).Model(&models.ParentModel{}).Where("id = ?", id).Updates(reviewUpdates(review))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrReviewNotFound
	}
	return nil
}
