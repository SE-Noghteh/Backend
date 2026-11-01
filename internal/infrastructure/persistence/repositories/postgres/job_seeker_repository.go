package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

// Persistence models keep GORM and SQL details out of the domain.
type JobSeekerModel struct {
	ID              uint64              `gorm:"primaryKey"`
	FirstName       string              `gorm:"size:100;not null"`
	LastName        string              `gorm:"size:100;not null"`
	NationalCode    string              `gorm:"size:10;not null;uniqueIndex"`
	City            string              `gorm:"size:40;not null;check:job_seekers_city,city IN ('تهران','اصفهان','مشهد','شیراز','تبریز','کرج','قم','اهواز','رشت','یزد')"`
	Address         string              `gorm:"size:1000;not null"`
	BirthDate       string              `gorm:"size:10;not null"`
	Gender          string              `gorm:"size:6;not null;check:job_seekers_gender,gender IN ('female','male')"`
	Interests       []entities.Interest `gorm:"serializer:json;type:jsonb;not null"`
	Phone           string              `gorm:"size:11;not null;uniqueIndex"`
	PasswordHash    string              `gorm:"size:60;not null"`
	PhoneVerifiedAt time.Time           `gorm:"not null"`
	CreatedAt       time.Time           `gorm:"not null"`
}

func (JobSeekerModel) TableName() string { return "job_seekers" }

type SignupOTPModel struct {
	Phone        string          `gorm:"primaryKey;size:11"`
	Salt         string          `gorm:"size:64;not null"`
	CodeHash     string          `gorm:"size:64;not null"`
	Attempts     int             `gorm:"not null"`
	SentAt       time.Time       `gorm:"not null"`
	ExpiresAt    time.Time       `gorm:"not null;index"`
	Registration *JobSeekerModel `gorm:"serializer:json;type:jsonb"`
}

func (SignupOTPModel) TableName() string { return "job_seeker_signup_otps" }

type JobSeekerRepository struct{ db *gorm.DB }

func NewJobSeekerRepository(db *gorm.DB) *JobSeekerRepository { return &JobSeekerRepository{db: db} }
func (r *JobSeekerRepository) LockPhone(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", phone).Error
}
func (r *JobSeekerRepository) Exists(ctx context.Context, phone, nationalCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&JobSeekerModel{}).Where("phone = ? OR national_code = ?", phone, nationalCode).Count(&count).Error
	return count > 0, err
}
func (r *JobSeekerRepository) GetOTP(ctx context.Context, phone string) (*entities.SignupOTP, error) {
	var model SignupOTPModel
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "phone = ?", phone).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var registration *entities.JobSeeker
	if model.Registration != nil {
		var err error
		registration, err = jobSeekerFromModel(model.Registration)
		if err != nil {
			return nil, err
		}
	}
	return &entities.SignupOTP{Phone: model.Phone, Salt: model.Salt, CodeHash: model.CodeHash, Attempts: model.Attempts, SentAt: model.SentAt, ExpiresAt: model.ExpiresAt, Registration: registration}, nil
}
func (r *JobSeekerRepository) SaveOTP(ctx context.Context, otp *entities.SignupOTP) error {
	model := SignupOTPModel{Phone: otp.Phone, Salt: otp.Salt, CodeHash: otp.CodeHash, Attempts: otp.Attempts, SentAt: otp.SentAt, ExpiresAt: otp.ExpiresAt}
	if otp.Registration != nil {
		model.Registration = jobSeekerToModel(otp.Registration)
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&model).Error
}
func (r *JobSeekerRepository) DeleteOTP(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Delete(&SignupOTPModel{}, "phone = ?", phone).Error
}
func (r *JobSeekerRepository) Create(ctx context.Context, seeker *entities.JobSeeker) error {
	model := jobSeekerToModel(seeker)
	err := r.db.WithContext(ctx).Create(model).Error
	var pgErr *pgconn.PgError
	if errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
		return ports.ErrSignupConflict
	}
	if err != nil {
		return fmt.Errorf("create job seeker: %w", err)
	}
	seeker.ID = model.ID
	return nil
}

func jobSeekerToModel(seeker *entities.JobSeeker) *JobSeekerModel {
	return &JobSeekerModel{ID: seeker.ID, FirstName: seeker.FirstName, LastName: seeker.LastName, NationalCode: seeker.NationalCode, City: string(seeker.City), Address: seeker.Address, BirthDate: seeker.BirthDate.String(), Gender: string(seeker.Gender), Interests: seeker.Interests, Phone: seeker.Phone, PasswordHash: seeker.PasswordHash, PhoneVerifiedAt: seeker.PhoneVerifiedAt, CreatedAt: seeker.CreatedAt}
}

func jobSeekerFromModel(model *JobSeekerModel) (*entities.JobSeeker, error) {
	var birth entities.JalaliDate
	if n, err := fmt.Sscanf(model.BirthDate, "%d/%d/%d", &birth.Year, &birth.Month, &birth.Day); err != nil || n != 3 {
		return nil, fmt.Errorf("invalid persisted birth date")
	}
	return &entities.JobSeeker{ID: model.ID, FirstName: model.FirstName, LastName: model.LastName, NationalCode: model.NationalCode, City: entities.City(model.City), Address: model.Address, BirthDate: birth, Gender: entities.Gender(model.Gender), Interests: model.Interests, Phone: model.Phone, PasswordHash: model.PasswordHash, PhoneVerifiedAt: model.PhoneVerifiedAt, CreatedAt: model.CreatedAt}, nil
}

func (r *JobSeekerRepository) FindByPhone(ctx context.Context, phone string) (*entities.JobSeeker, error) {
	return r.find(ctx, "phone = ?", phone)
}
func (r *JobSeekerRepository) FindByID(ctx context.Context, id uint64) (*entities.JobSeeker, error) {
	return r.find(ctx, "id = ?", id)
}
func (r *JobSeekerRepository) find(ctx context.Context, query string, value any) (*entities.JobSeeker, error) {
	var model JobSeekerModel
	err := r.db.WithContext(ctx).Where(query, value).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return jobSeekerFromModel(&model)
}

type LoginAttemptModel struct {
	Phone           string    `gorm:"primaryKey;size:11"`
	Failures        int       `gorm:"not null"`
	WindowStartedAt time.Time `gorm:"not null;index"`
}

func (LoginAttemptModel) TableName() string { return "job_seeker_login_attempts" }

func (r *JobSeekerRepository) GetLoginAttempt(ctx context.Context, phone string) (*entities.LoginAttempt, error) {
	var model LoginAttemptModel
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("phone = ?", phone).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entities.LoginAttempt{Phone: model.Phone, Failures: model.Failures, WindowStartedAt: model.WindowStartedAt}, nil
}
func (r *JobSeekerRepository) SaveLoginAttempt(ctx context.Context, attempt *entities.LoginAttempt) error {
	model := LoginAttemptModel{Phone: attempt.Phone, Failures: attempt.Failures, WindowStartedAt: attempt.WindowStartedAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&model).Error
}
func (r *JobSeekerRepository) DeleteLoginAttempt(ctx context.Context, phone string) error {
	return r.db.WithContext(ctx).Delete(&LoginAttemptModel{}, "phone = ?", phone).Error
}
