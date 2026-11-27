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

type JobPostingRepository struct{ db *gorm.DB }

func NewJobPostingRepository(db *gorm.DB) *JobPostingRepository {
	return &JobPostingRepository{db: db}
}

func (r *JobPostingRepository) Create(ctx context.Context, posting *entities.JobPosting) error {
	model := models.JobPostingModel{
		EmployerID: posting.EmployerID, Title: posting.Title,
		Description: posting.Description, Details: posting.Details, Status: "draft",
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}
	posting.ID, posting.Status = model.ID, model.Status
	posting.CreatedAt, posting.UpdatedAt = model.CreatedAt, model.UpdatedAt
	return nil
}

func (r *JobPostingRepository) FindOwned(ctx context.Context, id, employerID uint64, kind entities.JobPostingType, lock bool) (*entities.JobPosting, error) {
	query := r.db.WithContext(ctx).Where("id = ? AND employer_id = ? AND details->>'opportunity_type' = ?", id, employerID, kind)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var model models.JobPostingModel
	if err := query.Take(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrJobPostingNotFound
		}
		return nil, err
	}
	return &entities.JobPosting{
		ID: model.ID, EmployerID: model.EmployerID, Title: model.Title, Description: model.Description,
		Details: model.Details, Attachments: model.Attachments, Status: model.Status,
		CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}, nil
}

func (r *JobPostingRepository) UpdateOwned(ctx context.Context, posting *entities.JobPosting) error {
	fields := models.JobPostingModel{
		Title: posting.Title, Description: posting.Description, Details: posting.Details, Attachments: posting.Attachments,
	}
	return r.db.WithContext(ctx).Model(&models.JobPostingModel{}).
		Where("id = ? AND employer_id = ?", posting.ID, posting.EmployerID).
		Select("Title", "Description", "Details", "Attachments").Updates(&fields).Error
}

func (r *JobPostingRepository) DeleteOwned(ctx context.Context, id, employerID uint64) (bool, error) {
	result := r.db.WithContext(ctx).Where("id = ? AND employer_id = ?", id, employerID).
		Delete(&models.JobPostingModel{})
	return result.RowsAffected > 0, result.Error
}
