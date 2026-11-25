package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func (r *AdminReviewRepository) ListEmployers(ctx context.Context, kind enums.EmployerType, status string, limit, offset int) ([]entities.EmployerReviewSummary, int64, error) {
	query := r.db.WithContext(ctx).Table("employers AS e").
		Joins("LEFT JOIN individual_employers AS i ON i.employer_id = e.id AND e.type = ?", enums.EmployerIndividual).
		Joins("LEFT JOIN legal_employers AS l ON l.employer_id = e.id AND e.type = ?", enums.EmployerLegal).
		Where("COALESCE(i.phone_verified_at, l.phone_verified_at) > ?", time.Time{})
	if kind != "" {
		query = query.Where("e.type = ?", kind)
	}
	if status != "all" {
		query = query.Where("COALESCE(i.review_status, l.review_status) = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]entities.EmployerReviewSummary, 0)
	err := query.Select(`e.id, e.type AS employer_type,
		COALESCE(l.company_name, CONCAT_WS(' ', i.first_name, i.last_name)) AS name,
		COALESCE(i.phone, l.phone) AS phone, COALESCE(i.province, l.province) AS province,
		COALESCE(i.city, l.city) AS city, COALESCE(i.review_status, l.review_status) AS review_status,
		COALESCE(i.created_at, l.created_at) AS created_at`).
		Order("created_at ASC, e.id ASC").Limit(limit).Offset(offset).Scan(&rows).Error
	return rows, total, err
}

func (r *AdminReviewRepository) FindEmployer(ctx context.Context, id uint64) (*entities.EmployerReviewProfile, error) {
	var employer models.EmployerModel
	err := r.db.WithContext(ctx).Take(&employer, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	repo := NewEmployerRepository(r.db)
	switch employer.Type {
	case enums.EmployerIndividual:
		account, err := repo.FindIndividualByID(ctx, id)
		if err != nil || account == nil || account.PhoneVerifiedAt.IsZero() {
			return nil, err
		}
		return &entities.EmployerReviewProfile{Individual: account}, nil
	case enums.EmployerLegal:
		account, err := repo.FindLegalByID(ctx, id)
		if err != nil || account == nil || account.PhoneVerifiedAt.IsZero() {
			return nil, err
		}
		return &entities.EmployerReviewProfile{Legal: account}, nil
	default:
		return nil, nil
	}
}

func (r *AdminReviewRepository) ReviewEmployer(ctx context.Context, id uint64, kind enums.EmployerType, review valueobjects.AdminReview) error {
	var profile any
	switch kind {
	case enums.EmployerIndividual:
		profile = &models.IndividualEmployerModel{}
	case enums.EmployerLegal:
		profile = &models.LegalEmployerModel{}
	default:
		return domainerrors.ErrReviewNotFound
	}

	status := enums.EmployerRejected
	if review.AdminApprovalStatus == enums.AdminReviewApproved {
		status = enums.EmployerApproved
	}
	result := r.db.WithContext(ctx).Model(profile).Where("employer_id = ? AND phone_verified_at > ?", id, time.Time{}).
		Updates(map[string]any{
			"review_status": status, "admin_review_message": review.AdminReviewMessage,
			"admin_reviewed_at": review.AdminReviewedAt, "admin_reviewed_by": review.AdminReviewedBy,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrReviewNotFound
	}
	return nil
}
