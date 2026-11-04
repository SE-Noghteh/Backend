package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func (r *EmployerRepository) LegalExists(ctx context.Context, phone, national string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.LegalEmployerModel{}).Where("phone = ? OR company_national_id = ?", phone, national).Count(&count).Error
	return count > 0, err
}

func (r *EmployerRepository) CreateLegal(ctx context.Context, employer *entities.LegalEmployer) error {
	parent := models.EmployerModel{Type: enums.EmployerLegal}
	if err := r.db.WithContext(ctx).Create(&parent).Error; err != nil {
		return err
	}
	model := legalEmployerToModel(employer)
	model.EmployerID = parent.ID
	err := r.db.WithContext(ctx).Omit(clause.Associations).Create(model).Error
	var pgErr *pgconn.PgError
	if errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
		return domainerrors.ErrSignupConflict
	}
	if err != nil {
		return err
	}
	employer.Employer = entities.Employer{ID: parent.ID, Type: parent.Type}
	return nil
}

func (r *EmployerRepository) FindLegalByPhone(ctx context.Context, phone string) (*entities.LegalEmployer, error) {
	return r.findLegal(ctx, "legal_employers.phone = ?", phone)
}

func (r *EmployerRepository) FindLegalByID(ctx context.Context, id uint64) (*entities.LegalEmployer, error) {
	return r.findLegal(ctx, "legal_employers.employer_id = ?", id)
}

func (r *EmployerRepository) findLegal(ctx context.Context, query string, arg any) (*entities.LegalEmployer, error) {
	var model models.LegalEmployerModel
	err := r.db.WithContext(ctx).Joins("JOIN employers ON employers.id = legal_employers.employer_id AND employers.type = ?", enums.EmployerLegal).Where(query, arg).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return legalEmployerFromModel(&model), nil
}
