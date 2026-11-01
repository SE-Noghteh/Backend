package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

const (
	otpLifetime           = 5 * time.Minute
	otpResendDelay        = time.Minute
	pendingSignupLifetime = 30 * time.Minute
	maxOTPAttempts        = 5
)

type JobSeekerService struct {
	uow       ports.UnitOfWork
	sender    ports.OTPSender
	passwords ports.PasswordHasher
}

func NewJobSeekerService(uow ports.UnitOfWork, sender ports.OTPSender, passwords ports.PasswordHasher) *JobSeekerService {
	return &JobSeekerService{uow: uow, sender: sender, passwords: passwords}
}

func otpHash(salt, code string) string {
	sum := sha256.Sum256([]byte(salt + ":" + code))
	return hex.EncodeToString(sum[:])
}

// Signup validates the entire profile before storing a pending registration or sending SMS.
// An account becomes visible to login only after VerifyOTP succeeds.
func (s *JobSeekerService) Signup(ctx context.Context, in usecase.SignupJobSeeker) error {
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return err
	}
	if err := validateSignup(&in); err != nil {
		return err
	}
	hash, err := s.passwords.Hash(in.Password)
	if err != nil {
		return err
	}
	registration := &entities.JobSeeker{
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		NationalCode: in.NationalCode,
		City:         in.City,
		Address:      in.Address,
		BirthDate:    in.BirthDate,
		Gender:       in.Gender,
		Interests:    in.Interests,
		Phone:        phone,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	return s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		exists, err := repo.Exists(ctx, phone, in.NationalCode)
		if err != nil {
			return err
		}
		if exists {
			return ports.ErrSignupConflict
		}
		previous, err := repo.GetOTP(ctx, phone)
		if err != nil {
			return err
		}
		return s.sendChallenge(ctx, repo, registration, previous)
	})
}

// RequestOTP resends only for a previously validated, unexpired pending profile.
func (s *JobSeekerService) RequestOTP(ctx context.Context, rawPhone string) error {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return err
	}
	return s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		previous, err := repo.GetOTP(ctx, phone)
		if err != nil {
			return err
		}
		if !pendingSignupValid(previous, time.Now().UTC()) {
			return ports.ErrNoPendingSignup
		}
		return s.sendChallenge(ctx, repo, previous.Registration, previous)
	})
}

func pendingSignupValid(otp *entities.SignupOTP, now time.Time) bool {
	return otp != nil && otp.Registration != nil && now.Before(otp.Registration.CreatedAt.Add(pendingSignupLifetime))
}

func (s *JobSeekerService) sendChallenge(ctx context.Context, repo ports.JobSeekerRepository, registration *entities.JobSeeker, previous *entities.SignupOTP) error {
	now := time.Now().UTC()
	if previous != nil && now.Sub(previous.SentAt) < otpResendDelay {
		return ports.ErrOTPTooSoon
	}
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", number.Int64())
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	challenge := &entities.SignupOTP{
		Phone:        registration.Phone,
		Salt:         hex.EncodeToString(salt),
		SentAt:       now,
		ExpiresAt:    now.Add(otpLifetime),
		Registration: registration,
	}
	challenge.CodeHash = otpHash(challenge.Salt, code)
	if err := repo.SaveOTP(ctx, challenge); err != nil {
		return err
	}
	if err := s.sender.SendOTP(ctx, "98"+registration.Phone[1:], code); err != nil {
		return ports.ErrSMSUnavailable
	}
	return nil
}

func (s *JobSeekerService) VerifyOTP(ctx context.Context, in usecase.VerifySignupOTP) (*entities.JobSeeker, error) {
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return nil, err
	}
	var result *entities.JobSeeker
	var verificationErr error
	err = s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		otp, err := repo.GetOTP(ctx, phone)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if !pendingSignupValid(otp, now) || !now.Before(otp.ExpiresAt) || otp.Attempts >= maxOTPAttempts {
			verificationErr = ports.ErrInvalidOTP
			return nil
		}
		actual := otpHash(otp.Salt, in.OTP)
		if !codePattern.MatchString(in.OTP) || subtle.ConstantTimeCompare([]byte(actual), []byte(otp.CodeHash)) != 1 {
			otp.Attempts++
			verificationErr = ports.ErrInvalidOTP
			return repo.SaveOTP(ctx, otp) // Commit failed attempts rather than rolling them back.
		}
		exists, err := repo.Exists(ctx, phone, otp.Registration.NationalCode)
		if err != nil {
			return err
		}
		if exists {
			return ports.ErrSignupConflict
		}
		result = otp.Registration
		result.PhoneVerifiedAt, result.CreatedAt = now, now
		if err := repo.Create(ctx, result); err != nil {
			return err
		}
		return repo.DeleteOTP(ctx, phone)
	})
	if err != nil {
		return nil, err
	}
	if verificationErr != nil {
		return nil, verificationErr
	}
	return result, nil
}
