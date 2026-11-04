package service

import (
	"context"
	"crypto/subtle"
	"time"

	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

// Both employer profiles use the same OTP lifecycle and transaction rules.
type employerSignupFlow[T any] struct {
	uow        ports.UnitOfWork
	sender     ports.OTPSender
	repository func(ports.RepositoryFactory) ports.PendingEmployerSignupRepository[T]
	createdAt  func(*T) time.Time
	finalize   func(context.Context, ports.RepositoryFactory, *T) error
}

func (f employerSignupFlow[T]) valid(p *entities.PendingEmployerSignup[T], now time.Time) bool {
	return p != nil && p.Challenge != nil && p.Registration != nil && now.Before(f.createdAt(p.Registration).Add(pendingSignupLifetime))
}

func sendEmployerChallenge[T any](ctx context.Context, sender ports.OTPSender, repo ports.PendingEmployerSignupRepository[T], phone string, pending *entities.PendingEmployerSignup[T]) error {
	challenge, code, err := newSignupChallenge(phone, time.Now().UTC())
	if err != nil {
		return err
	}
	pending.Challenge = challenge
	if err := repo.SaveSignup(ctx, pending); err != nil {
		return err
	}
	if err := sender.SendOTP(ctx, "98"+phone[1:], code); err != nil {
		return domainerrors.ErrSMSUnavailable
	}
	return nil
}

func (f employerSignupFlow[T]) resend(ctx context.Context, rawPhone string) error {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return err
	}
	return f.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		if err := factory.EmployerRepository().LockPhone(ctx, phone); err != nil {
			return err
		}
		repo := f.repository(factory)
		pending, err := repo.GetSignup(ctx, phone)
		if err != nil {
			return err
		}
		if !f.valid(pending, time.Now().UTC()) {
			return domainerrors.ErrNoPendingSignup
		}
		if time.Since(pending.Challenge.SentAt) < otpResendDelay {
			return domainerrors.ErrOTPTooSoon
		}
		return sendEmployerChallenge(ctx, f.sender, repo, phone, pending)
	})
}

func (f employerSignupFlow[T]) verify(ctx context.Context, rawPhone, code string) (*T, error) {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return nil, err
	}
	var result *T
	var verificationErr error
	err = f.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		if err := factory.EmployerRepository().LockPhone(ctx, phone); err != nil {
			return err
		}
		repo := f.repository(factory)
		pending, err := repo.GetSignup(ctx, phone)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if !f.valid(pending, now) {
			verificationErr = domainerrors.ErrInvalidOTP
			return nil
		}
		otp := pending.Challenge
		if !now.Before(otp.ExpiresAt) || otp.Attempts >= maxOTPAttempts {
			verificationErr = domainerrors.ErrInvalidOTP
			return nil
		}
		actual := otpHash(otp.Salt, code)
		if !codePattern.MatchString(code) || subtle.ConstantTimeCompare([]byte(actual), []byte(otp.CodeHash)) != 1 {
			otp.Attempts++
			verificationErr = domainerrors.ErrInvalidOTP
			return repo.SaveSignup(ctx, pending)
		}
		if err := f.finalize(ctx, factory, pending.Registration); err != nil {
			return err
		}
		result = pending.Registration
		return repo.DeleteSignup(ctx, phone)
	})
	if err != nil {
		return nil, err
	}
	if verificationErr != nil {
		return nil, verificationErr
	}
	return result, nil
}
