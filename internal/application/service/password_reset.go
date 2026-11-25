package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

type PasswordResetService struct {
	uow       ports.UnitOfWork
	passwords ports.PasswordHasher
	sender    ports.PasswordResetSender
}

func NewPasswordResetService(uow ports.UnitOfWork, passwords ports.PasswordHasher, sender ports.PasswordResetSender) *PasswordResetService {
	return &PasswordResetService{uow: uow, passwords: passwords, sender: sender}
}

func (s *PasswordResetService) Request(ctx context.Context, role enums.AccountRole, rawPhone string) error {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return err
	}
	return s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.PasswordResetRepository()
		account, err := repo.LockAccount(ctx, role, phone)
		if err != nil || account == nil || account.PhoneVerifiedAt.IsZero() {
			return err
		}
		previous, err := repo.Get(ctx, role, phone)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if resetBelongsTo(previous, account) && now.Sub(previous.OTP.SentAt) < otpResendDelay {
			return nil
		}
		otp, code, err := newSignupChallenge(phone, now)
		if err != nil {
			return err
		}
		challenge := &ports.PasswordResetChallenge{Role: role, AccountID: account.ID, SessionVersion: account.SessionVersion, OTP: *otp}
		if err := repo.Save(ctx, challenge); err != nil {
			return err
		}
		if err := s.sender.SendPasswordResetOTP(ctx, "98"+phone[1:], code); err != nil {
			return domainerrors.ErrSMSUnavailable
		}
		return nil
	})
}

func (s *PasswordResetService) VerifyOTP(ctx context.Context, role enums.AccountRole, rawPhone, code string) (string, error) {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return "", err
	}
	var token string
	var invalidCode bool
	err = s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.PasswordResetRepository()
		account, err := repo.LockAccount(ctx, role, phone)
		if err != nil {
			return err
		}
		challenge, err := repo.Get(ctx, role, phone)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if !resetBelongsTo(challenge, account) || challenge.TokenHash != nil || !now.Before(challenge.OTP.ExpiresAt) || challenge.OTP.Attempts >= maxOTPAttempts {
			return domainerrors.ErrInvalidOTP
		}
		otp := &challenge.OTP
		if !codePattern.MatchString(code) || subtle.ConstantTimeCompare([]byte(otpHash(otp.Salt, code)), []byte(otp.CodeHash)) != 1 {
			otp.Attempts++
			invalidCode = true
			return repo.Save(ctx, challenge)
		}
		var secret [32]byte
		if _, err := rand.Read(secret[:]); err != nil {
			return err
		}
		token = base64.RawURLEncoding.EncodeToString(secret[:])
		hash := resetTokenHash(secret[:])
		challenge.TokenHash = &hash
		otp.CodeHash, otp.Salt = "", ""
		otp.ExpiresAt = now.Add(otpLifetime)
		return repo.Save(ctx, challenge)
	})
	if err != nil {
		return "", err
	}
	if invalidCode {
		return "", domainerrors.ErrInvalidOTP
	}
	return token, nil
}

func (s *PasswordResetService) Reset(ctx context.Context, role enums.AccountRole, token, password string) error {
	if len(password) < 8 || len(password) > 72 {
		return fmt.Errorf("%w: new_password must contain 8 to 72 bytes", domainerrors.ErrInvalidSignup)
	}
	if (role == enums.RoleIndividualEmployer || role == enums.RoleLegalEmployer) && !strongEmployerPassword(password) {
		return fmt.Errorf("%w: new_password must contain 12 to 72 bytes with uppercase, lowercase, a digit, and a symbol", domainerrors.ErrInvalidSignup)
	}
	if len(token) != base64.RawURLEncoding.EncodedLen(32) {
		return domainerrors.ErrInvalidResetToken
	}
	secret, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil {
		return domainerrors.ErrInvalidResetToken
	}
	tokenHash := resetTokenHash(secret)
	return s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.PasswordResetRepository()
		grant, err := repo.FindByTokenHash(ctx, role, tokenHash)
		if err != nil {
			return err
		}
		if grant == nil {
			return domainerrors.ErrInvalidResetToken
		}
		phone := grant.OTP.Phone
		account, err := repo.LockAccount(ctx, role, phone)
		if err != nil {
			return err
		}
		// Re-read after locking to reject concurrent consumption or replacement.
		grant, err = repo.Get(ctx, role, phone)
		if err != nil {
			return err
		}
		if !resetBelongsTo(grant, account) || grant.TokenHash == nil || *grant.TokenHash != tokenHash || !time.Now().UTC().Before(grant.OTP.ExpiresAt) {
			return domainerrors.ErrInvalidResetToken
		}
		hash, err := s.passwords.Hash(password)
		if err != nil {
			return err
		}
		if err := repo.UpdatePassword(ctx, role, account.ID, hash); err != nil {
			return err
		}
		if err := repo.Delete(ctx, role, phone); err != nil {
			return err
		}
		switch role {
		case enums.RoleJobSeeker:
			if err := f.JobSeekerRepository().DeletePhoneChangeOTP(ctx, account.ID); err != nil {
				return err
			}
			return f.LoginAttemptRepository().DeleteLoginAttempt(ctx, phone)
		case enums.RoleParent:
			return f.ParentRepository().DeleteLoginAttempt(ctx, phone)
		default:
			return f.EmployerLoginAttemptRepository().DeleteLoginAttempt(ctx, phone)
		}
	})
}

func resetBelongsTo(challenge *ports.PasswordResetChallenge, account *ports.PasswordResetAccount) bool {
	return challenge != nil && account != nil && !account.PhoneVerifiedAt.IsZero() &&
		challenge.AccountID == account.ID && challenge.SessionVersion == account.SessionVersion
}

func resetTokenHash(secret []byte) string {
	hash := sha256.Sum256(secret)
	return hex.EncodeToString(hash[:])
}
