package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

type memoryPasswordResetRepository struct{ memory *signupMemory }

func (m *signupMemory) PasswordResetRepository() ports.PasswordResetRepository {
	return &memoryPasswordResetRepository{memory: m}
}

func (m *signupMemory) SendPasswordResetOTP(ctx context.Context, destination, code string) error {
	return m.SendOTP(ctx, destination, code)
}

func clonePasswordReset(in *ports.PasswordResetChallenge) *ports.PasswordResetChallenge {
	if in == nil {
		return nil
	}
	copy := *in
	return &copy
}

func (r *memoryPasswordResetRepository) LockAccount(_ context.Context, role enums.AccountRole, phone string) (*ports.PasswordResetAccount, error) {
	seeker := r.memory.seeker
	if role != enums.RoleJobSeeker || seeker == nil || seeker.Phone != phone {
		return nil, nil
	}
	return &ports.PasswordResetAccount{ID: seeker.ID, Phone: seeker.Phone, SessionVersion: seeker.SessionVersion, PhoneVerifiedAt: seeker.PhoneVerifiedAt}, nil
}

func (r *memoryPasswordResetRepository) Get(_ context.Context, role enums.AccountRole, phone string) (*ports.PasswordResetChallenge, error) {
	c := r.memory.passwordReset
	if c == nil || c.Role != role || c.OTP.Phone != phone {
		return nil, nil
	}
	return clonePasswordReset(c), nil
}

func (r *memoryPasswordResetRepository) FindByTokenHash(_ context.Context, role enums.AccountRole, hash string) (*ports.PasswordResetChallenge, error) {
	c := r.memory.passwordReset
	if c == nil || c.Role != role || c.TokenHash == nil || *c.TokenHash != hash {
		return nil, nil
	}
	return clonePasswordReset(c), nil
}

func (r *memoryPasswordResetRepository) Save(_ context.Context, c *ports.PasswordResetChallenge) error {
	if r.memory.saveErr != nil {
		return r.memory.saveErr
	}
	r.memory.passwordReset = clonePasswordReset(c)
	return nil
}

func (r *memoryPasswordResetRepository) Delete(ctx context.Context, role enums.AccountRole, phone string) error {
	if c, _ := r.Get(ctx, role, phone); c != nil {
		r.memory.passwordReset = nil
	}
	return nil
}

func (r *memoryPasswordResetRepository) UpdatePassword(ctx context.Context, _ enums.AccountRole, id uint64, hash string) error {
	if err := r.memory.LockAccount(ctx, id); err != nil {
		return err
	}
	r.memory.seeker.PasswordHash = hash
	r.memory.seeker.SessionVersion++
	return nil
}

const newResetPassword = "my-new-password"

func TestPasswordResetLifecycleAndSessionInvalidation(t *testing.T) {
	m := verifiedMemory(t)
	s := NewPasswordResetService(m, testPasswords{}, m)
	ctx := context.Background()
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	oldLogin, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: m.seeker.Phone, Password: validSignup().Password})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Request(ctx, enums.RoleJobSeeker, "+989123456789"); err != nil {
		t.Fatal(err)
	}
	code := m.smsCode
	if m.passwordReset == nil || m.passwordReset.OTP.CodeHash == code || m.otp != nil || m.phoneChange != nil {
		t.Fatal("reset challenge not isolated or code stored in plaintext")
	}
	m.phoneChange = cloneOTP(&m.passwordReset.OTP)
	m.attempt = &entities.LoginAttempt{Phone: m.seeker.Phone, Failures: maxLoginFailures, WindowStartedAt: time.Now()}
	token, err := s.VerifyOTP(ctx, enums.RoleJobSeeker, m.seeker.Phone, code)
	if err != nil {
		t.Fatal(err)
	}
	if m.passwordReset.TokenHash == nil || *m.passwordReset.TokenHash == token || m.passwordReset.OTP.CodeHash != "" || m.seeker.SessionVersion != 0 {
		t.Fatal("OTP verification changed password or stored the raw token")
	}
	if _, err := s.VerifyOTP(ctx, enums.RoleJobSeeker, m.seeker.Phone, code); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("verified OTP was reusable")
	}
	if err := s.Reset(ctx, enums.RoleParent, token, newResetPassword); !errors.Is(err, domainerrors.ErrInvalidResetToken) {
		t.Fatal("reset token worked across account roles")
	}
	if err := s.Reset(ctx, enums.RoleJobSeeker, token, newResetPassword); err != nil {
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
	if err := s.Reset(ctx, enums.RoleJobSeeker, token, "another-password"); !errors.Is(err, domainerrors.ErrInvalidResetToken) {
		t.Fatal("OTP replay accepted", err)
	}
}
func TestForgotPasswordUnknownAndUnverifiedAccounts(t *testing.T) {
	for _, scenario := range []string{"unknown", "pending"} {
		t.Run(scenario, func(t *testing.T) {
			m := verifiedMemory(t)
			s := NewPasswordResetService(m, testPasswords{}, m)
			count := m.smsCount
			if scenario == "unknown" {
				m.seeker = nil
			} else {
				m.seeker.PhoneVerifiedAt = time.Time{}
			}
			if err := s.Request(context.Background(), enums.RoleJobSeeker, validSignup().Phone); err != nil {
				t.Fatal("unknown/unverified account response leaked existence", err)
			}
			if m.passwordReset != nil || m.smsCount != count {
				t.Fatal("unknown/unverified account received OTP")
			}
			if _, err := s.VerifyOTP(context.Background(), enums.RoleJobSeeker, validSignup().Phone, "123456"); !errors.Is(err, domainerrors.ErrInvalidOTP) {
				t.Fatal("unverified account password changed", err)
			}
		})
	}
}
func TestPasswordResetCooldownAttemptsAndResend(t *testing.T) {
	m := verifiedMemory(t)
	s := NewPasswordResetService(m, testPasswords{}, m)
	ctx := context.Background()
	phone := m.seeker.Phone
	if err := s.Request(ctx, enums.RoleJobSeeker, phone); err != nil {
		t.Fatal(err)
	}
	code, sends := m.smsCode, m.smsCount
	if err := s.Request(ctx, enums.RoleJobSeeker, phone); err != nil || m.smsCount != sends {
		t.Fatal("cooldown bypassed", err)
	}
	for i := 0; i < maxOTPAttempts; i++ {
		if _, err := s.VerifyOTP(ctx, enums.RoleJobSeeker, phone, "invalid"); !errors.Is(err, domainerrors.ErrInvalidOTP) {
			t.Fatal(err)
		}
	}
	if m.passwordReset.OTP.Attempts != maxOTPAttempts {
		t.Fatal("failed attempts rolled back")
	}
	if _, err := s.VerifyOTP(ctx, enums.RoleJobSeeker, phone, code); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("locked OTP accepted", err)
	}
	m.passwordReset.OTP.SentAt = time.Now().Add(-2 * time.Minute)
	if err := s.Request(ctx, enums.RoleJobSeeker, phone); err != nil {
		t.Fatal(err)
	}
	if m.passwordReset.OTP.Attempts != 0 {
		t.Fatal("new challenge kept previous attempts")
	}
	if _, err := s.VerifyOTP(ctx, enums.RoleJobSeeker, phone, m.smsCode); err != nil {
		t.Fatal(err)
	}
}
func TestPasswordResetRejectsInvalidOrMismatchedChallenge(t *testing.T) {
	for _, scenario := range []string{"expired", "wrong phone", "changed phone", "signup OTP", "phone-change OTP", "short password", "long password"} {
		t.Run(scenario, func(t *testing.T) {
			m := verifiedMemory(t)
			s := NewPasswordResetService(m, testPasswords{}, m)
			ctx := context.Background()
			phone := m.seeker.Phone
			if err := s.Request(ctx, enums.RoleJobSeeker, phone); err != nil {
				t.Fatal(err)
			}
			code := m.smsCode
			password := newResetPassword
			switch scenario {
			case "expired":
				m.passwordReset.OTP.ExpiresAt = time.Now().Add(-time.Second)
			case "wrong phone":
				phone = changedPhone
			case "changed phone":
				m.seeker.Phone = changedPhone
				phone = changedPhone
			case "signup OTP":
				m.otp = &m.passwordReset.OTP
				m.passwordReset = nil
			case "phone-change OTP":
				m.phoneChange = &m.passwordReset.OTP
				m.passwordReset = nil
			case "short password":
				password = "short"
			case "long password":
				password = strings.Repeat("x", 73)
			}
			token, err := s.VerifyOTP(ctx, enums.RoleJobSeeker, phone, code)
			if err == nil {
				err = s.Reset(ctx, enums.RoleJobSeeker, token, password)
			}
			if err == nil {
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
			s := NewPasswordResetService(m, testPasswords{}, m)
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
				if err := s.Request(ctx, enums.RoleJobSeeker, phone); err == nil || m.passwordReset != nil {
					t.Fatal("failed request retained challenge")
				}
				return
			}
			if err := s.Request(ctx, enums.RoleJobSeeker, phone); err != nil {
				t.Fatal(err)
			}
			token, err := s.VerifyOTP(ctx, enums.RoleJobSeeker, phone, m.smsCode)
			if err != nil {
				t.Fatal(err)
			}
			m.commitErr = errors.New("reset commit failed")
			if err := s.Reset(ctx, enums.RoleJobSeeker, token, newResetPassword); err == nil {
				t.Fatal("failed commit accepted")
			}
			if m.passwordReset == nil || m.seeker.PasswordHash != "hash:"+validSignup().Password || m.seeker.SessionVersion != 0 {
				t.Fatal("failed commit changed credentials or consumed OTP")
			}
		})
	}
}

func TestVerifiedPasswordResetTokenExpiresOrIsReplaced(t *testing.T) {
	for _, scenario := range []string{"expiry", "new OTP request"} {
		t.Run(scenario, func(t *testing.T) {
			m := verifiedMemory(t)
			s := NewPasswordResetService(m, testPasswords{}, m)
			ctx := context.Background()
			role, phone := enums.RoleJobSeeker, m.seeker.Phone
			if err := s.Request(ctx, role, phone); err != nil {
				t.Fatal(err)
			}
			token, err := s.VerifyOTP(ctx, role, phone, m.smsCode)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "expiry" {
				m.passwordReset.OTP.ExpiresAt = time.Now().Add(-time.Second)
			} else {
				m.passwordReset.OTP.SentAt = time.Now().Add(-2 * time.Minute)
				if err := s.Request(ctx, role, phone); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Reset(ctx, role, token, newResetPassword); !errors.Is(err, domainerrors.ErrInvalidResetToken) {
				t.Fatalf("old token accepted: %v", err)
			}
			if m.seeker.SessionVersion != 0 {
				t.Fatal("invalid reset changed account state")
			}
		})
	}
}
