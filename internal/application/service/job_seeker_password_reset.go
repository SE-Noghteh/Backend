package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

func (s *JobSeekerService) ForgotPassword(ctx context.Context, rawPhone string) error {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return err
	}
	account, err := s.uow.Factory().JobSeekerRepository().FindByPhone(ctx, phone)
	if err != nil {
		return err
	}
	// Return the same successful response for unknown and unverified accounts.
	if account == nil || account.PhoneVerifiedAt.IsZero() {
		return nil
	}
	return s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.JobSeekerRepository()
		if err := repo.LockAccount(ctx, account.ID); err != nil {
			if errors.Is(err, domainerrors.ErrUnauthenticated) {
				return nil
			}
			return err
		}
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		current, err := repo.FindByID(ctx, account.ID)
		if err != nil {
			return err
		}
		if current == nil || current.Phone != phone || current.PhoneVerifiedAt.IsZero() {
			return nil
		}
		previous, err := repo.GetPasswordResetOTP(ctx, current.ID)
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
		if err := repo.SavePasswordResetOTP(ctx, current.ID, challenge); err != nil {
			return err
		}
		if err := s.sender.SendOTP(ctx, "98"+phone[1:], code); err != nil {
			return domainerrors.ErrSMSUnavailable
		}
		return nil
	})
}

func (s *JobSeekerService) ResetPassword(ctx context.Context, rawPhone, code, password string) error {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return err
	}
	if len(password) < 8 || len(password) > 72 {
		return fmt.Errorf("%w: new_password must contain 8 to 72 bytes", domainerrors.ErrInvalidSignup)
	}
	account, err := s.uow.Factory().JobSeekerRepository().FindByPhone(ctx, phone)
	if err != nil {
		return err
	}
	if account == nil || account.PhoneVerifiedAt.IsZero() {
		return domainerrors.ErrInvalidOTP
	}
	var verificationErr error
	err = s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.JobSeekerRepository()
		if err := repo.LockAccount(ctx, account.ID); err != nil {
			if errors.Is(err, domainerrors.ErrUnauthenticated) {
				return domainerrors.ErrInvalidOTP
			}
			return err
		}
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		current, err := repo.FindByID(ctx, account.ID)
		if err != nil {
			return err
		}
		if current == nil || current.Phone != phone || current.PhoneVerifiedAt.IsZero() {
			return domainerrors.ErrInvalidOTP
		}
		otp, err := repo.GetPasswordResetOTP(ctx, current.ID)
		if err != nil {
			return err
		}
		if otp == nil || otp.Phone != phone || !time.Now().UTC().Before(otp.ExpiresAt) || otp.Attempts >= maxOTPAttempts {
			return domainerrors.ErrInvalidOTP
		}
		actual := otpHash(otp.Salt, code)
		if !codePattern.MatchString(code) || subtle.ConstantTimeCompare([]byte(actual), []byte(otp.CodeHash)) != 1 {
			otp.Attempts++
			verificationErr = domainerrors.ErrInvalidOTP
			return repo.SavePasswordResetOTP(ctx, current.ID, otp)
		}
		hash, err := s.passwords.Hash(password)
		if err != nil {
			return err
		}
		if err := repo.UpdatePassword(ctx, current.ID, hash); err != nil {
			return err
		}
		if err := repo.DeletePasswordResetOTP(ctx, current.ID); err != nil {
			return err
		}
		if err := repo.DeletePhoneChangeOTP(ctx, current.ID); err != nil {
			return err
		}
		return f.LoginAttemptRepository().DeleteLoginAttempt(ctx, phone)
	})
	if err != nil {
		return err
	}
	return verificationErr
}
