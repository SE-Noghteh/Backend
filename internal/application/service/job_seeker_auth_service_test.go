package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

type testTokens struct{ issueErr error }

func (f testTokens) IssueAccessToken(id uint64) (*entities.AccessToken, error) {
	if f.issueErr != nil {
		return nil, f.issueErr
	}
	return &entities.AccessToken{Value: "valid-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (f testTokens) VerifyAccessToken(value string) (*entities.JobSeekerPrincipal, error) {
	if value != "valid-token" {
		return nil, ports.ErrUnauthenticated
	}
	return &entities.JobSeekerPrincipal{ID: 1, TokenID: "test-token-id", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func verifiedMemory(t *testing.T) *signupMemory {
	t.Helper()
	m := &signupMemory{}
	signup := NewJobSeekerService(m, m, testPasswords{})
	ctx := context.Background()
	if err := signup.Signup(ctx, validSignup()); err != nil {
		t.Fatal(err)
	}
	if _, err := signup.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: m.smsCode}); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestLoginAndAuthenticate(t *testing.T) {
	m := verifiedMemory(t)
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	ctx := context.Background()
	result, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: "+989123456789", Password: validSignup().Password})
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken.Value != "valid-token" || result.JobSeeker.ID != 1 {
		t.Fatal("missing login identity or token")
	}
	seeker, err := auth.Authenticate(ctx, result.AccessToken.Value)
	if err != nil || seeker.ID != 1 {
		t.Fatalf("authenticate: %v", err)
	}
	if _, err := auth.Authenticate(ctx, "invalid"); !errors.Is(err, ports.ErrUnauthenticated) {
		t.Fatal(err)
	}
	m.seeker = nil
	if _, err := auth.Authenticate(ctx, "valid-token"); !errors.Is(err, ports.ErrUnauthenticated) {
		t.Fatal("deleted account accepted")
	}
}

func TestLoginRejectsPendingUnknownAndWrongPassword(t *testing.T) {
	for _, scenario := range []string{"pending", "unknown", "wrong password", "unverified"} {
		t.Run(scenario, func(t *testing.T) {
			m := verifiedMemory(t)
			in := usecase.LoginJobSeeker{Phone: validSignup().Phone, Password: validSignup().Password}
			switch scenario {
			case "pending":
				m.otp = &entities.SignupOTP{Registration: m.seeker}
				m.seeker = nil
			case "unknown":
				in.Phone = "09120000000"
			case "wrong password":
				in.Password = "wrong-password"
			case "unverified":
				m.seeker.PhoneVerifiedAt = time.Time{}
			}
			auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
			if result, err := auth.Login(context.Background(), in); !errors.Is(err, ports.ErrInvalidCredentials) || result != nil {
				t.Fatalf("login accepted: %v", err)
			}
			if m.attempt == nil || m.attempt.Failures != 1 {
				t.Fatal("failed login attempt not persisted")
			}
		})
	}
}

func TestLoginThrottleAndReset(t *testing.T) {
	m := verifiedMemory(t)
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	ctx := context.Background()
	in := usecase.LoginJobSeeker{Phone: validSignup().Phone, Password: "wrong-password"}
	for i := 0; i < 5; i++ {
		if _, err := auth.Login(ctx, in); !errors.Is(err, ports.ErrInvalidCredentials) {
			t.Fatal(err)
		}
	}
	in.Password = validSignup().Password
	if _, err := auth.Login(ctx, in); !errors.Is(err, ports.ErrLoginThrottled) {
		t.Fatal("lockout bypassed with correct password")
	}
	m.attempt.WindowStartedAt = time.Now().Add(-16 * time.Minute)
	if _, err := auth.Login(ctx, in); err != nil {
		t.Fatal(err)
	}
	if m.attempt != nil {
		t.Fatal("successful login must clear attempt history")
	}
}

func TestTokenFailureDoesNotReportSuccessfulLogin(t *testing.T) {
	m := verifiedMemory(t)
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{issueErr: errors.New("signer unavailable")})
	if result, err := auth.Login(context.Background(), usecase.LoginJobSeeker{Phone: validSignup().Phone, Password: validSignup().Password}); err == nil || result != nil {
		t.Fatal("login ignored token issuance failure")
	}
}

func TestLogoutRevokesTokenAcrossServiceInstances(t *testing.T) {
	m := verifiedMemory(t)
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	ctx := context.Background()
	if _, err := auth.Authenticate(ctx, "valid-token"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Logout(ctx, "valid-token"); err != nil {
		t.Fatal(err)
	}
	other := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	if _, err := other.Authenticate(ctx, "valid-token"); !errors.Is(err, ports.ErrUnauthenticated) {
		t.Fatal("revoked token accepted")
	}
	if err := auth.Logout(ctx, "valid-token"); err != nil {
		t.Fatal("repeated logout must be idempotent")
	}
	if err := auth.Logout(ctx, "invalid"); !errors.Is(err, ports.ErrUnauthenticated) {
		t.Fatal("invalid token logged out")
	}
}

func TestRevocationDatabaseFailureFailsClosed(t *testing.T) {
	m := verifiedMemory(t)
	m.revocationErr = errors.New("database unavailable")
	auth := NewJobSeekerAuthService(m, testPasswords{}, testTokens{})
	if _, err := auth.Authenticate(context.Background(), "valid-token"); !errors.Is(err, m.revocationErr) {
		t.Fatal("revocation lookup failure ignored")
	}
	if err := auth.Logout(context.Background(), "valid-token"); !errors.Is(err, m.revocationErr) {
		t.Fatal("logout reported success without storing revocation")
	}
}
