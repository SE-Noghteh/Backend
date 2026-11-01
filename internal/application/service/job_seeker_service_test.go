package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
	domainpostgres "zedlancer/backend/internal/domain/ports/postgres"
)

type signupMemory struct {
	otp           *entities.SignupOTP
	seeker        *entities.JobSeeker
	attempt       *entities.LoginAttempt
	smsCode       string
	smsCount      int
	smsErr        error
	createErr     error
	revoked       map[string]time.Time
	revocationErr error
}

func (m *signupMemory) Factory() ports.RepositoryFactory { return m }
func (m *signupMemory) WithTransaction(fn func(ports.RepositoryFactory) error) error {
	previousOTP, previousSeeker, previousAttempt := cloneOTP(m.otp), cloneSeeker(m.seeker), cloneAttempt(m.attempt)
	err := fn(m)
	if err != nil {
		m.otp, m.seeker, m.attempt = previousOTP, previousSeeker, previousAttempt
	}
	return err
}
func cloneSeeker(in *entities.JobSeeker) *entities.JobSeeker {
	if in == nil {
		return nil
	}
	out := *in
	out.Interests = append([]entities.Interest(nil), in.Interests...)
	return &out
}
func cloneOTP(in *entities.SignupOTP) *entities.SignupOTP {
	if in == nil {
		return nil
	}
	out := *in
	out.Registration = cloneSeeker(in.Registration)
	return &out
}
func cloneAttempt(in *entities.LoginAttempt) *entities.LoginAttempt {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
func (m *signupMemory) UserRepository() domainpostgres.UserRepository              { return nil }
func (m *signupMemory) JobSeekerRepository() ports.JobSeekerRepository             { return m }
func (m *signupMemory) TokenRevocationRepository() ports.TokenRevocationRepository { return m }
func (m *signupMemory) IsRevoked(_ context.Context, tokenID string) (bool, error) {
	_, revoked := m.revoked[tokenID]
	return revoked, m.revocationErr
}
func (m *signupMemory) Revoke(_ context.Context, tokenID string, expiresAt time.Time) error {
	if m.revocationErr != nil {
		return m.revocationErr
	}
	if m.revoked == nil {
		m.revoked = make(map[string]time.Time)
	}
	m.revoked[tokenID] = expiresAt
	return nil
}
func (m *signupMemory) LockPhone(context.Context, string) error { return nil }
func (m *signupMemory) Exists(_ context.Context, phone, national string) (bool, error) {
	return m.seeker != nil && (m.seeker.Phone == phone || m.seeker.NationalCode == national), nil
}
func (m *signupMemory) GetOTP(_ context.Context, phone string) (*entities.SignupOTP, error) {
	if m.otp == nil || m.otp.Phone != phone {
		return nil, nil
	}
	return cloneOTP(m.otp), nil
}
func (m *signupMemory) SaveOTP(_ context.Context, otp *entities.SignupOTP) error {
	m.otp = cloneOTP(otp)
	return nil
}
func (m *signupMemory) DeleteOTP(context.Context, string) error { m.otp = nil; return nil }
func (m *signupMemory) Create(_ context.Context, seeker *entities.JobSeeker) error {
	if m.createErr != nil {
		return m.createErr
	}
	seeker.ID = 1
	m.seeker = cloneSeeker(seeker)
	return nil
}
func (m *signupMemory) FindByPhone(_ context.Context, phone string) (*entities.JobSeeker, error) {
	if m.seeker != nil && m.seeker.Phone == phone {
		return cloneSeeker(m.seeker), nil
	}
	return nil, nil
}
func (m *signupMemory) FindByID(_ context.Context, id uint64) (*entities.JobSeeker, error) {
	if m.seeker != nil && m.seeker.ID == id {
		return cloneSeeker(m.seeker), nil
	}
	return nil, nil
}
func (m *signupMemory) GetLoginAttempt(_ context.Context, phone string) (*entities.LoginAttempt, error) {
	if m.attempt != nil && m.attempt.Phone == phone {
		return cloneAttempt(m.attempt), nil
	}
	return nil, nil
}
func (m *signupMemory) SaveLoginAttempt(_ context.Context, in *entities.LoginAttempt) error {
	m.attempt = cloneAttempt(in)
	return nil
}
func (m *signupMemory) DeleteLoginAttempt(context.Context, string) error { m.attempt = nil; return nil }
func (m *signupMemory) SendOTP(_ context.Context, destination, code string) error {
	if destination != "989123456789" {
		return errors.New("wrong destination")
	}
	if m.smsErr != nil {
		return m.smsErr
	}
	m.smsCode = code
	m.smsCount++
	return nil
}

type testPasswords struct{}

func (testPasswords) Hash(password string) (string, error) { return "hash:" + password, nil }
func (testPasswords) Matches(hash, password string) bool {
	return hash != "" && hash == "hash:"+password
}

func validSignup() usecase.SignupJobSeeker {
	return usecase.SignupJobSeeker{FirstName: "علی", LastName: "رضایی", NationalCode: "0013547895", City: entities.CityTehran, Address: "خیابان آزادی", BirthDate: entities.JalaliDate{Year: 1380, Month: 1, Day: 1}, Gender: entities.GenderMale, Interests: []entities.Interest{entities.InterestProgramming}, Phone: "09123456789", Password: "a-good-password"}
}

func TestSignupAutomaticallySendsOTPAndRequiresVerification(t *testing.T) {
	m := &signupMemory{}
	s := NewJobSeekerService(m, m, testPasswords{})
	ctx := context.Background()
	in := validSignup()
	in.Phone = "+989123456789"
	if err := s.Signup(ctx, in); err != nil {
		t.Fatal(err)
	}
	if m.smsCount != 1 || m.seeker != nil || m.otp.Registration == nil {
		t.Fatal("signup must send OTP and persist only a pending registration")
	}
	if m.otp.CodeHash == m.smsCode || m.otp.Salt == "" {
		t.Fatal("OTP was not hashed")
	}
	if m.otp.Registration.PasswordHash == in.Password || !strings.HasPrefix(m.otp.Registration.PasswordHash, "hash:") {
		t.Fatal("plaintext password persisted")
	}
	if err := s.RequestOTP(ctx, in.Phone); !errors.Is(err, ports.ErrOTPTooSoon) {
		t.Fatalf("cooldown: %v", err)
	}
	verify := usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode}
	seeker, err := s.VerifyOTP(ctx, verify)
	if err != nil {
		t.Fatal(err)
	}
	if m.otp != nil || seeker.ID == 0 || seeker.PhoneVerifiedAt.IsZero() || seeker.Phone != "09123456789" {
		t.Fatal("OTP must be consumed and verified account persisted")
	}
	if _, err := s.VerifyOTP(ctx, verify); !errors.Is(err, ports.ErrInvalidOTP) {
		t.Fatalf("OTP replay: %v", err)
	}
	if err := s.Signup(ctx, in); !errors.Is(err, ports.ErrSignupConflict) {
		t.Fatalf("duplicate signup: %v", err)
	}
}

func TestOTPExpiryAndLockout(t *testing.T) {
	for _, scenario := range []string{"expired code", "expired registration", "wrong code", "malformed code"} {
		t.Run(scenario, func(t *testing.T) {
			m := &signupMemory{}
			s := NewJobSeekerService(m, m, testPasswords{})
			ctx := context.Background()
			if err := s.Signup(ctx, validSignup()); err != nil {
				t.Fatal(err)
			}
			wrong := "000000"
			if wrong == m.smsCode {
				wrong = "111111"
			}
			if scenario == "expired code" {
				m.otp.ExpiresAt = time.Now().Add(-time.Second)
			}
			if scenario == "expired registration" {
				m.otp.Registration.CreatedAt = time.Now().Add(-31 * time.Minute)
			}
			if scenario == "malformed code" {
				wrong = "abc"
			}
			for i := 0; i < maxOTPAttempts; i++ {
				if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: wrong}); !errors.Is(err, ports.ErrInvalidOTP) {
					t.Fatalf("attempt %d: %v", i, err)
				}
			}
			if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: m.smsCode}); !errors.Is(err, ports.ErrInvalidOTP) {
				t.Fatalf("expired/locked code accepted: %v", err)
			}
			if strings.HasSuffix(scenario, "code") && scenario != "expired code" && m.otp.Attempts != 5 {
				t.Fatal("failed attempts not committed")
			}
		})
	}
}

func TestResendRequiresPendingProfileAndInvalidatesOldCode(t *testing.T) {
	m := &signupMemory{}
	s := NewJobSeekerService(m, m, testPasswords{})
	ctx := context.Background()
	if err := s.RequestOTP(ctx, validSignup().Phone); !errors.Is(err, ports.ErrNoPendingSignup) {
		t.Fatal(err)
	}
	if err := s.Signup(ctx, validSignup()); err != nil {
		t.Fatal(err)
	}
	oldSalt := m.otp.Salt
	m.otp.SentAt = time.Now().Add(-2 * time.Minute)
	if err := s.RequestOTP(ctx, validSignup().Phone); err != nil {
		t.Fatal(err)
	}
	if m.otp.Salt == oldSalt || m.smsCount != 2 {
		t.Fatal("resend must replace the challenge")
	}
	m.otp.Registration.CreatedAt = time.Now().Add(-31 * time.Minute)
	if err := s.RequestOTP(ctx, validSignup().Phone); !errors.Is(err, ports.ErrNoPendingSignup) {
		t.Fatal(err)
	}
}

func TestSignupRollsBackOnSMSFailure(t *testing.T) {
	m := &signupMemory{smsErr: errors.New("provider down")}
	s := NewJobSeekerService(m, m, testPasswords{})
	if err := s.Signup(context.Background(), validSignup()); !errors.Is(err, ports.ErrSMSUnavailable) {
		t.Fatal(err)
	}
	if m.otp != nil || m.seeker != nil {
		t.Fatal("failed delivery must roll back pending signup")
	}
}

func TestVerificationRetainsCodeOnDatabaseFailure(t *testing.T) {
	m := &signupMemory{}
	s := NewJobSeekerService(m, m, testPasswords{})
	ctx := context.Background()
	if err := s.Signup(ctx, validSignup()); err != nil {
		t.Fatal(err)
	}
	m.createErr = errors.New("database unavailable")
	if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: m.smsCode}); err == nil {
		t.Fatal("database error not returned")
	}
	if m.otp == nil || m.seeker != nil {
		t.Fatal("failed creation must not consume code")
	}
}

func TestSignupRejectsInvalidProfileBeforeSendingSMS(t *testing.T) {
	for _, mutate := range []func(*usecase.SignupJobSeeker){
		func(in *usecase.SignupJobSeeker) { in.City = "invalid" },
		func(in *usecase.SignupJobSeeker) { in.Interests = []entities.Interest{"invalid"} },
		func(in *usecase.SignupJobSeeker) {
			in.Interests = []entities.Interest{entities.InterestDesign, entities.InterestDesign}
		},
		func(in *usecase.SignupJobSeeker) { in.NationalCode = "1111111111" },
		func(in *usecase.SignupJobSeeker) { in.BirthDate = entities.JalaliDate{Year: 1404, Month: 12, Day: 30} },
		func(in *usecase.SignupJobSeeker) { in.BirthDate = entities.JalaliDate{Year: 1500, Month: 1, Day: 1} },
		func(in *usecase.SignupJobSeeker) { in.Password = "short" },
		func(in *usecase.SignupJobSeeker) { in.Phone = "123" },
	} {
		m := &signupMemory{}
		s := NewJobSeekerService(m, m, testPasswords{})
		in := validSignup()
		mutate(&in)
		if err := s.Signup(context.Background(), in); !errors.Is(err, ports.ErrInvalidSignup) {
			t.Fatalf("invalid input accepted: %v", err)
		}
		if m.smsCount != 0 || m.otp != nil {
			t.Fatal("invalid signup sent SMS or persisted data")
		}
	}
}
