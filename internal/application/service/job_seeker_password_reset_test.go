package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

func (m *signupMemory) GetPasswordResetOTP(ctx context.Context, id uint64) (*entities.SignupOTP, error) {
	if err := m.LockAccount(ctx, id); err != nil {
		return nil, err
	}
	return cloneOTP(m.passwordReset), nil
}
func (m *signupMemory) SavePasswordResetOTP(_ context.Context, _ uint64, otp *entities.SignupOTP) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.passwordReset = cloneOTP(otp)
	return nil
}
func (m *signupMemory) DeletePasswordResetOTP(context.Context, uint64) error {
	m.passwordReset = nil
	return nil
}
func (m *signupMemory) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	if err := m.LockAccount(ctx, id); err != nil {
		return err
	}
	m.seeker.PasswordHash = hash
	m.seeker.SessionVersion++
	return nil
}

const newResetPassword = "my-new-password"

func TestPasswordResetLifecycleAndSessionInvalidation(t *testing.T) {
	m := verifiedMemory(t)
	s := newJobSeekerService(m)
	ctx := context.Background()
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	oldLogin, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: m.seeker.Phone, Password: validSignup().Password})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ForgotPassword(ctx, "+989123456789"); err != nil {
		t.Fatal(err)
	}
	code := m.smsCode
	if m.passwordReset == nil || m.passwordReset.CodeHash == code || m.otp != nil || m.phoneChange != nil {
		t.Fatal("reset challenge not isolated or code stored in plaintext")
	}
	m.phoneChange = cloneOTP(m.passwordReset)
	m.attempt = &entities.LoginAttempt{Phone: m.seeker.Phone, Failures: maxLoginFailures, WindowStartedAt: time.Now()}
	if err := s.ResetPassword(ctx, m.seeker.Phone, code, newResetPassword); err != nil {
		t.Fatal(err)
	}
	if m.passwordReset != nil || m.phoneChange != nil || m.attempt != nil || m.seeker.PasswordHash != "hash:"+newResetPassword || m.seeker.SessionVersion != 1 {
		t.Fatal("reset did not atomically update password/version and clear challenges")
	}
	if _, err := auth.Authenticate(ctx, oldLogin.AccessToken.Value); !errors.Is(err, domainerrors.ErrUnauthenticated) {
		t.Fatal("old session survived reset")
	}
	if _, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: m.seeker.Phone, Password: validSignup().Password}); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
		t.Fatal("old password still works")
	}
	fresh, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: m.seeker.Phone, Password: newResetPassword})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Authenticate(ctx, fresh.AccessToken.Value); err != nil {
		t.Fatal("new session was invalidated", err)
	}
	if err := s.ResetPassword(ctx, m.seeker.Phone, code, "another-password"); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("OTP replay accepted", err)
	}
}
func TestForgotPasswordUnknownAndUnverifiedAccounts(t *testing.T) {
	for _, scenario := range []string{"unknown", "pending"} {
		t.Run(scenario, func(t *testing.T) {
			m := verifiedMemory(t)
			s := newJobSeekerService(m)
			count := m.smsCount
			if scenario == "unknown" {
				m.seeker = nil
			} else {
				m.seeker.PhoneVerifiedAt = time.Time{}
			}
			if err := s.ForgotPassword(context.Background(), validSignup().Phone); err != nil {
				t.Fatal("unknown/unverified account response leaked existence", err)
			}
			if m.passwordReset != nil || m.smsCount != count {
				t.Fatal("unknown/unverified account received OTP")
			}
			if err := s.ResetPassword(context.Background(), validSignup().Phone, "123456", newResetPassword); !errors.Is(err, domainerrors.ErrInvalidOTP) {
				t.Fatal("unverified account password changed", err)
			}
		})
	}
}
func TestPasswordResetCooldownAttemptsAndResend(t *testing.T) {
	m := verifiedMemory(t)
	s := newJobSeekerService(m)
	ctx := context.Background()
	phone := m.seeker.Phone
	if err := s.ForgotPassword(ctx, phone); err != nil {
		t.Fatal(err)
	}
	code := m.smsCode
	if err := s.ForgotPassword(ctx, phone); !errors.Is(err, domainerrors.ErrOTPTooSoon) {
		t.Fatal("cooldown bypassed", err)
	}
	for i := 0; i < maxOTPAttempts; i++ {
		if err := s.ResetPassword(ctx, phone, "invalid", newResetPassword); !errors.Is(err, domainerrors.ErrInvalidOTP) {
			t.Fatal(err)
		}
	}
	if m.passwordReset.Attempts != maxOTPAttempts {
		t.Fatal("failed attempts rolled back")
	}
	if err := s.ResetPassword(ctx, phone, code, newResetPassword); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("locked OTP accepted", err)
	}
	m.passwordReset.SentAt = time.Now().Add(-2 * time.Minute)
	if err := s.ForgotPassword(ctx, phone); err != nil {
		t.Fatal(err)
	}
	if m.passwordReset.Attempts != 0 {
		t.Fatal("new challenge kept previous attempts")
	}
	if err := s.ResetPassword(ctx, phone, m.smsCode, newResetPassword); err != nil {
		t.Fatal(err)
	}
}
func TestPasswordResetRejectsInvalidOrMismatchedChallenge(t *testing.T) {
	for _, scenario := range []string{"expired", "wrong phone", "changed phone", "signup OTP", "phone-change OTP", "short password", "long password"} {
		t.Run(scenario, func(t *testing.T) {
			m := verifiedMemory(t)
			s := newJobSeekerService(m)
			ctx := context.Background()
			phone := m.seeker.Phone
			if err := s.ForgotPassword(ctx, phone); err != nil {
				t.Fatal(err)
			}
			code := m.smsCode
			password := newResetPassword
			switch scenario {
			case "expired":
				m.passwordReset.ExpiresAt = time.Now().Add(-time.Second)
			case "wrong phone":
				phone = changedPhone
			case "changed phone":
				m.seeker.Phone = changedPhone
				phone = changedPhone
			case "signup OTP":
				m.otp = m.passwordReset
				m.passwordReset = nil
			case "phone-change OTP":
				m.phoneChange = m.passwordReset
				m.passwordReset = nil
			case "short password":
				password = "short"
			case "long password":
				password = strings.Repeat("x", 73)
			}
			if err := s.ResetPassword(ctx, phone, code, password); err == nil {
				t.Fatal("invalid reset accepted")
			}
			if m.seeker.PasswordHash != "hash:"+validSignup().Password || m.seeker.SessionVersion != 0 {
				t.Fatal("invalid reset changed credentials")
			}
		})
	}
}
func TestPasswordResetFailuresRollback(t *testing.T) {
	for _, scenario := range []string{"sms", "save", "request commit", "reset commit"} {
		t.Run(scenario, func(t *testing.T) {
			m := verifiedMemory(t)
			s := newJobSeekerService(m)
			ctx := context.Background()
			phone := m.seeker.Phone
			switch scenario {
			case "sms":
				m.smsErr = errors.New("SMS failed")
			case "save":
				m.saveErr = errors.New("save failed")
			case "request commit":
				m.commitErr = errors.New("commit failed")
			}
			if scenario != "reset commit" {
				if err := s.ForgotPassword(ctx, phone); err == nil || m.passwordReset != nil {
					t.Fatal("failed request retained challenge")
				}
				return
			}
			if err := s.ForgotPassword(ctx, phone); err != nil {
				t.Fatal(err)
			}
			m.commitErr = errors.New("reset commit failed")
			if err := s.ResetPassword(ctx, phone, m.smsCode, newResetPassword); err == nil {
				t.Fatal("failed commit accepted")
			}
			if m.passwordReset == nil || m.seeker.PasswordHash != "hash:"+validSignup().Password || m.seeker.SessionVersion != 0 {
				t.Fatal("failed commit changed credentials or consumed OTP")
			}
		})
	}
}
