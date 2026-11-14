package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/valueobjects"
)

func (m *signupMemory) LockAccount(ctx context.Context, id uint64) error {
	if m.seeker == nil || m.seeker.ID != id {
		return domainerrors.ErrUnauthenticated
	}
	return nil
}
func (m *signupMemory) UpdatePhone(ctx context.Context, id uint64, phone string) error {
	if err := m.LockAccount(ctx, id); err != nil {
		return err
	}
	m.seeker.Phone = phone
	m.seeker.PhoneVerifiedAt = time.Now().UTC()
	return nil
}
func (m *signupMemory) Delete(ctx context.Context, id uint64) error {
	if err := m.LockAccount(ctx, id); err != nil {
		return err
	}
	m.seeker = nil
	return nil
}
func (m *signupMemory) GetPhoneChangeOTP(ctx context.Context, id uint64) (*entities.SignupOTP, error) {
	if err := m.LockAccount(ctx, id); err != nil {
		return nil, err
	}
	return cloneOTP(m.phoneChange), nil
}
func (m *signupMemory) SavePhoneChangeOTP(ctx context.Context, id uint64, otp *entities.SignupOTP) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.phoneChange = cloneOTP(otp)
	return nil
}
func (m *signupMemory) DeletePhoneChangeOTP(context.Context, uint64) error {
	m.phoneChange = nil
	return nil
}

const changedPhone = "09129876543"

func phoneChangeService(t *testing.T) (*signupMemory, *JobSeekerService) {
	t.Helper()
	m := verifiedMemory(t)
	m.smsDestination = "989129876543"
	return m, newJobSeekerService(m)
}

func TestPhoneChangeLifecycle(t *testing.T) {
	m, s := phoneChangeService(t)
	ctx := context.Background()
	if err := s.RequestPhoneChange(ctx, 1, "+989129876543"); err != nil {
		t.Fatal(err)
	}
	if m.seeker.Phone != validSignup().Phone || m.phoneChange.Phone != changedPhone || m.phoneChange.CodeHash == m.smsCode || m.otp != nil {
		t.Fatal("request changed account or mixed signup and phone change challenges")
	}
	if err := s.RequestPhoneChange(ctx, 1, "09121111111"); !errors.Is(err, domainerrors.ErrOTPTooSoon) {
		t.Fatalf("destination change bypassed cooldown: %v", err)
	}
	seeker, err := s.VerifyPhoneChange(ctx, 1, usecase.VerifySignupOTP{Phone: changedPhone, OTP: m.smsCode})
	if err != nil || seeker.Phone != changedPhone || m.phoneChange != nil {
		t.Fatalf("verify: %v", err)
	}
	if _, err := s.VerifyPhoneChange(ctx, 1, usecase.VerifySignupOTP{Phone: changedPhone, OTP: m.smsCode}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("replayed OTP accepted")
	}
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	if _, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: validSignup().Phone, Password: validSignup().Password}); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
		t.Fatal("old phone still logs in")
	}
	if _, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: changedPhone, Password: validSignup().Password}); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Authenticate(ctx, "valid-token"); err != nil {
		t.Fatal("existing session lost after phone change", err)
	}
}

func TestPhoneChangeRejectsInvalidChallenges(t *testing.T) {
	for _, scenario := range []string{"expired", "wrong code", "wrong phone", "wrong account", "malformed", "attempt limit"} {
		t.Run(scenario, func(t *testing.T) {
			m, s := phoneChangeService(t)
			ctx := context.Background()
			if err := s.RequestPhoneChange(ctx, 1, changedPhone); err != nil {
				t.Fatal(err)
			}
			in := usecase.VerifySignupOTP{Phone: changedPhone, OTP: m.smsCode}
			id := uint64(1)
			switch scenario {
			case "expired":
				m.phoneChange.ExpiresAt = time.Now().Add(-time.Second)
			case "wrong code":
				in.OTP = "000000"
				if in.OTP == m.smsCode {
					in.OTP = "111111"
				}
			case "wrong phone":
				in.Phone = "09121111111"
			case "wrong account":
				id = 2
			case "malformed":
				in.OTP = "x"
			case "attempt limit":
				m.phoneChange.Attempts = maxOTPAttempts
			}
			if _, err := s.VerifyPhoneChange(ctx, id, in); err == nil {
				t.Fatal("invalid challenge accepted")
			}
			if m.seeker.Phone != validSignup().Phone {
				t.Fatal("invalid verification changed phone")
			}
			if scenario == "wrong code" || scenario == "wrong phone" || scenario == "malformed" {
				if m.phoneChange.Attempts != 1 {
					t.Fatal("failed attempt was rolled back")
				}
			}
		})
	}
}

func TestPhoneChangeAttemptsAndReplacement(t *testing.T) {
	m, s := phoneChangeService(t)
	ctx := context.Background()
	if err := s.RequestPhoneChange(ctx, 1, changedPhone); err != nil {
		t.Fatal(err)
	}
	code := m.smsCode
	for i := 0; i < maxOTPAttempts; i++ {
		if _, err := s.VerifyPhoneChange(ctx, 1, usecase.VerifySignupOTP{Phone: changedPhone, OTP: "bad"}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
			t.Fatal(err)
		}
	}
	if _, err := s.VerifyPhoneChange(ctx, 1, usecase.VerifySignupOTP{Phone: changedPhone, OTP: code}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("locked challenge accepted")
	}
	m.phoneChange.SentAt = time.Now().Add(-2 * time.Minute)
	m.smsDestination = "989121111111"
	if err := s.RequestPhoneChange(ctx, 1, "09121111111"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.VerifyPhoneChange(ctx, 1, usecase.VerifySignupOTP{Phone: changedPhone, OTP: code}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("replaced destination accepted")
	}
	if _, err := s.VerifyPhoneChange(ctx, 1, usecase.VerifySignupOTP{Phone: "09121111111", OTP: m.smsCode}); err != nil {
		t.Fatal(err)
	}
}

func TestPhoneChangeFailuresRollback(t *testing.T) {
	for _, scenario := range []string{"sms", "save", "commit", "same phone", "invalid phone", "occupied"} {
		t.Run(scenario, func(t *testing.T) {
			m, s := phoneChangeService(t)
			phone := changedPhone
			switch scenario {
			case "sms":
				m.smsErr = errors.New("SMS failure")
			case "save":
				m.saveErr = errors.New("save failure")
			case "commit":
				m.commitErr = errors.New("commit failure")
			case "same phone":
				phone = m.seeker.Phone
			case "invalid phone":
				phone = "123"
			case "occupied":
				m.occupiedPhone = phone
			}
			if err := s.RequestPhoneChange(context.Background(), 1, phone); err == nil {
				t.Fatal("invalid request accepted")
			}
			if m.phoneChange != nil {
				t.Fatal("failed request retained challenge")
			}
		})
	}
}

func TestDeleteAccountRequiresPasswordAndInvalidatesSessions(t *testing.T) {
	m, s := phoneChangeService(t)
	ctx := context.Background()
	if err := s.RequestPhoneChange(ctx, 1, changedPhone); err != nil {
		t.Fatal(err)
	}
	for _, password := range []string{"wrong-password", ""} {
		if err := s.DeleteAccount(ctx, 1, password); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
			t.Fatal(err)
		}
		if m.seeker == nil || m.phoneChange == nil {
			t.Fatal("bad password deleted account data")
		}
	}
	if m.attempt.Failures != 2 {
		t.Fatal("password failures not persisted")
	}
	if err := s.DeleteAccount(ctx, 1, validSignup().Password); err != nil {
		t.Fatal(err)
	}
	if m.seeker != nil || m.phoneChange != nil || m.attempt != nil {
		t.Fatal("deletion left account data")
	}
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	if _, err := auth.Authenticate(ctx, "valid-token"); !errors.Is(err, domainerrors.ErrUnauthenticated) {
		t.Fatal("deleted account token accepted")
	}
}

func TestDeleteAccountThrottleAndRollback(t *testing.T) {
	ctx := context.Background()
	m, s := phoneChangeService(t)
	for i := 0; i < maxLoginFailures; i++ {
		_ = s.DeleteAccount(ctx, 1, "wrong-password")
	}
	if err := s.DeleteAccount(ctx, 1, validSignup().Password); !errors.Is(err, domainerrors.ErrLoginThrottled) {
		t.Fatal("password confirmation not throttled", err)
	}
	m.attempt.WindowStartedAt = time.Now().Add(-16 * time.Minute)
	m.commitErr = errors.New("commit failure")
	if err := s.DeleteAccount(ctx, 1, validSignup().Password); err == nil || m.seeker == nil {
		t.Fatal("failed transaction deleted account")
	}
	m.commitErr = nil
	if err := s.DeleteAccount(ctx, 1, validSignup().Password); err != nil {
		t.Fatal(err)
	}
}

func TestPhoneChangeConflictAndCommitRollback(t *testing.T) {
	for _, scenario := range []string{"conflict", "commit"} {
		t.Run(scenario, func(t *testing.T) {
			m, s := phoneChangeService(t)
			ctx := context.Background()
			if err := s.RequestPhoneChange(ctx, 1, changedPhone); err != nil {
				t.Fatal(err)
			}
			if scenario == "conflict" {
				m.occupiedPhone = changedPhone
			} else {
				m.commitErr = errors.New("commit failed")
			}
			if _, err := s.VerifyPhoneChange(ctx, 1, usecase.VerifySignupOTP{Phone: changedPhone, OTP: m.smsCode}); err == nil {
				t.Fatal("failed verification accepted")
			}
			if m.seeker.Phone != validSignup().Phone || m.phoneChange == nil {
				t.Fatal("failed verification changed account or consumed code")
			}
		})
	}
}

func TestDeleteAccountVideoCleanupAfterCommit(t *testing.T) {
	m := verifiedMemory(t)
	video := *m.seeker.VerificationVideo
	storage := &employerMemory{objects: map[string]valueobjects.StoredObject{video.Key: video}}
	s := NewJobSeekerService(m, m, testPasswords{}, storage, storage)
	ctx := context.Background()
	m.commitErr = errors.New("commit failed")
	if err := s.DeleteAccount(ctx, 1, validSignup().Password); err == nil {
		t.Fatal("failed commit accepted")
	}
	if len(storage.deleted) != 0 || m.seeker == nil {
		t.Fatal("failed deletion removed video or account")
	}
	m.commitErr = nil
	if err := s.DeleteAccount(ctx, 1, validSignup().Password); err != nil {
		t.Fatal(err)
	}
	if len(storage.objects) != 0 || len(storage.deleted) != 1 || storage.deleted[0] != video.Key {
		t.Fatal("account video was not deleted")
	}
}
