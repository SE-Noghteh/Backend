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

type EmployerRepository struct{ db *gorm.DB }

func NewEmployerRepository(db *gorm.DB) *EmployerRepository { return &EmployerRepository{db: db} }

func (r *EmployerRepository) LockPhone(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "employer:"+phone).Error
}

func (r *EmployerRepository) IndividualExists(ctx context.Context, phone, national string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.IndividualEmployerModel{}).Where("phone = ? OR national_code = ?", phone, national).Count(&count).Error
	return count > 0, err
}

func (r *EmployerRepository) CreateIndividual(ctx context.Context, employer *entities.IndividualEmployer) error {
	parent := models.EmployerModel{Type: enums.EmployerIndividual}
	if err := r.db.WithContext(ctx).Create(&parent).Error; err != nil {
		return err
	}
	model := individualEmployerToModel(employer)
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

func (r *EmployerRepository) FindIndividualByPhone(ctx context.Context, phone string) (*entities.IndividualEmployer, error) {
	return r.find(ctx, "individual_employers.phone = ?", phone)
}

func (r *EmployerRepository) FindIndividualByID(ctx context.Context, id uint64) (*entities.IndividualEmployer, error) {
	return r.find(ctx, "individual_employers.employer_id = ?", id)
}

func (r *EmployerRepository) find(ctx context.Context, query string, arg any) (*entities.IndividualEmployer, error) {
	var model models.IndividualEmployerModel
	err := r.db.WithContext(ctx).Joins("JOIN employers ON employers.id = individual_employers.employer_id AND employers.type = ?", enums.EmployerIndividual).Where(query, arg).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return individualEmployerFromModel(&model)
}
