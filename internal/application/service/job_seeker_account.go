package service

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

func (s *JobSeekerService) RequestPhoneChange(ctx context.Context, id uint64, rawPhone string) error {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return err
	}
	return s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockAccount(ctx, id); err != nil {
			return err
		}
		seeker, err := repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if seeker == nil || seeker.PhoneVerifiedAt.IsZero() {
			return domainerrors.ErrUnauthenticated
		}
		if seeker.Phone == phone {
			return fmt.Errorf("%w: new phone must differ from current phone", domainerrors.ErrInvalidSignup)
		}
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		other, err := repo.FindByPhone(ctx, phone)
		if err != nil {
			return err
		}
		if other != nil {
			return domainerrors.ErrSignupConflict
		}
		previous, err := repo.GetPhoneChangeOTP(ctx, id)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if previous != nil && now.Sub(previous.SentAt) < otpResendDelay {
			return domainerrors.ErrOTPTooSoon
		}
		challenge, code, err := newSignupChallenge(phone, now)
		if err != nil {
			return err
		}
		if err := repo.SavePhoneChangeOTP(ctx, id, challenge); err != nil {
			return err
		}
		if err := s.sender.SendOTP(ctx, "98"+phone[1:], code); err != nil {
			return domainerrors.ErrSMSUnavailable
		}
		return nil
	})
}

func (s *JobSeekerService) VerifyPhoneChange(ctx context.Context, id uint64, in usecase.VerifySignupOTP) (*entities.JobSeeker, error) {
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return nil, err
	}
	var result *entities.JobSeeker
	var verificationErr error
	err = s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockAccount(ctx, id); err != nil {
			return err
		}
		seeker, err := repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if seeker == nil || seeker.PhoneVerifiedAt.IsZero() {
			return domainerrors.ErrUnauthenticated
		}
		otp, err := repo.GetPhoneChangeOTP(ctx, id)
		if err != nil {
			return err
		}
		if otp == nil || !time.Now().UTC().Before(otp.ExpiresAt) || otp.Attempts >= maxOTPAttempts {
			verificationErr = domainerrors.ErrInvalidOTP
			return nil
		}
		actual := otpHash(otp.Salt, in.OTP)
		if phone != otp.Phone || !codePattern.MatchString(in.OTP) || subtle.ConstantTimeCompare([]byte(actual), []byte(otp.CodeHash)) != 1 {
			otp.Attempts++
			verificationErr = domainerrors.ErrInvalidOTP
			return repo.SavePhoneChangeOTP(ctx, id, otp)
		}
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		other, err := repo.FindByPhone(ctx, phone)
		if err != nil {
			return err
		}
		if other != nil {
			return domainerrors.ErrSignupConflict
		}
		if err := repo.UpdatePhone(ctx, id, phone); err != nil {
			return err
		}
		if err := repo.DeletePhoneChangeOTP(ctx, id); err != nil {
			return err
		}
		result, err = repo.FindByID(ctx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	if verificationErr != nil {
		return nil, verificationErr
	}
	return result, nil
}

func (s *JobSeekerService) DeleteAccount(ctx context.Context, id uint64, password string) error {
	var videoKey string
	var authErr error
	err := s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockAccount(ctx, id); err != nil {
			return err
		}
		seeker, err := repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if seeker == nil || seeker.PhoneVerifiedAt.IsZero() {
			return domainerrors.ErrUnauthenticated
		}
		if err := repo.LockPhone(ctx, seeker.Phone); err != nil {
			return err
		}
		attemptRepo := factory.LoginAttemptRepository()
		attempt, err := nextLoginAttempt(ctx, attemptRepo, seeker.Phone)
		if err != nil {
			return err
		}
		if len(password) < 8 || len(password) > 72 || !s.passwords.Matches(seeker.PasswordHash, password) {
			attempt.Failures++
			authErr = domainerrors.ErrInvalidCredentials
			return attemptRepo.SaveLoginAttempt(ctx, attempt)
		}
		if err := repo.DeletePhoneChangeOTP(ctx, id); err != nil {
			return err
		}
		if err := attemptRepo.DeleteLoginAttempt(ctx, seeker.Phone); err != nil {
			return err
		}
		if err := repo.Delete(ctx, id); err != nil {
			return err
		}
		if seeker.VerificationVideo != nil {
			videoKey = seeker.VerificationVideo.Key
		}
		return nil
	})
	if err != nil {
		return err
	}
	if authErr != nil {
		return authErr
	}
	if videoKey != "" {
		cleanupEmployerUploads(ctx, s.storage, []string{videoKey})
	}
	return nil
}
