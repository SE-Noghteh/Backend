package ports

import (
	"context"
	"errors"
	"zedlancer/backend/internal/domain/entities"
)

var (
	ErrInvalidSignup  = errors.New("invalid signup")
	ErrInvalidOTP     = errors.New("invalid or expired OTP")
	ErrSignupConflict = errors.New("phone or national code already registered")
	ErrOTPTooSoon     = errors.New("wait before requesting another OTP")
	ErrSMSUnavailable = errors.New("SMS service unavailable")
)

type OTPSender interface {
	SendOTP(context.Context, string, string) error
}

// Repository methods run within the application's unit of work.
type JobSeekerRepository interface {
	LockPhone(context.Context, string) error
	Exists(context.Context, string, string) (bool, error)
	GetOTP(context.Context, string) (*entities.SignupOTP, error)
	SaveOTP(context.Context, *entities.SignupOTP) error
	DeleteOTP(context.Context, string) error
	Create(context.Context, *entities.JobSeeker) error
	FindByPhone(context.Context, string) (*entities.JobSeeker, error)
	FindByID(context.Context, uint64) (*entities.JobSeeker, error)
	GetLoginAttempt(context.Context, string) (*entities.LoginAttempt, error)
	SaveLoginAttempt(context.Context, *entities.LoginAttempt) error
	DeleteLoginAttempt(context.Context, string) error
}
