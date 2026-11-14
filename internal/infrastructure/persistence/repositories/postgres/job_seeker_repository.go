package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

type JobSeekerRepository struct{ db *gorm.DB }

func NewJobSeekerRepository(db *gorm.DB) *JobSeekerRepository { return &JobSeekerRepository{db: db} }
func (r *JobSeekerRepository) LockPhone(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", phone).Error
}
func (r *JobSeekerRepository) Exists(ctx context.Context, phone, nationalCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.JobSeekerModel{}).Where("phone = ? OR national_code = ?", phone, nationalCode).Count(&count).Error
	return count > 0, err
}
func (r *JobSeekerRepository) Create(ctx context.Context, seeker *entities.JobSeeker) error {
	model := jobSeekerToModel(seeker)
	err := r.db.WithContext(ctx).Create(model).Error
	var pgErr *pgconn.PgError
	if errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
		return domainerrors.ErrSignupConflict
	}
	if err != nil {
		return fmt.Errorf("create job seeker: %w", err)
	}
	seeker.ID = model.ID
	return nil
}

func (r *JobSeekerRepository) FindByPhone(ctx context.Context, phone string) (*entities.JobSeeker, error) {
	return r.find(ctx, "phone = ?", phone)
}
func (r *JobSeekerRepository) FindByID(ctx context.Context, id uint64) (*entities.JobSeeker, error) {
	return r.find(ctx, "id = ?", id)
}
func (r *JobSeekerRepository) find(ctx context.Context, query string, value any) (*entities.JobSeeker, error) {
	var model models.JobSeekerModel
	err := r.db.WithContext(ctx).Where(query, value).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return jobSeekerFromModel(&model)
}
