package postgres

import (
	"context"

	"gorm.io/gorm"

	"zedlancer/backend/internal/domain/entities"
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

func (r *JobPostingRepository) UpdateOwned(ctx context.Context, id, employerID uint64, title, description string, details []byte) (bool, error) {
	result := r.db.WithContext(ctx).Model(&models.JobPostingModel{}).
		Where("id = ? AND employer_id = ?", id, employerID).
		Updates(map[string]any{"title": title, "description": description, "details": details})
	return result.RowsAffected > 0, result.Error
}

func (r *JobPostingRepository) DeleteOwned(ctx context.Context, id, employerID uint64) (bool, error) {
	result := r.db.WithContext(ctx).Where("id = ? AND employer_id = ?", id, employerID).
		Delete(&models.JobPostingModel{})
	return result.RowsAffected > 0, result.Error
}
